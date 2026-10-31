package v1_4_11

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"time"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/app/upgrades"
	"github.com/initia-labs/initia/app/upgrades/v1_4_11/clamm"
	movetypes "github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

// the dex arb operator wallet (init172ccg0snhl0l40392u2ls66nux3z4gxcr9wcyr)
var gateOperator = "0xf2b1843e13bfdffabe255715f86b53e1a22aa0d8"

const (
	// keyless sink that holds the forged balances
	// sha256("quarantine")[:20] = init1cttxw40x6z6z77j5rp7qyr6vsrmhjcs2v3vlmu, no key exists
	quarantineSink = "0x000000000000000000000000c2d66755e6d0b42f7a54187c020f4c80f779620a"
	// exploiter account, holds the forged balances that never left L1
	attacker = "dd922c841d85e12dac2dc2eef5a5e29cfff6edfa"
	// dex-clamm package (move params clamm_module_address), an ObjectAccount
	clammModuleAddress = "0xd78a3b72c7ef0cfba7286bfb8c618aa4d6011dce05a832871cc9ab323c25f55e"
	// emergency proposals recover the expired IBC clients right after the restart
	emergencyTallyInterval = 5 * time.Minute
)

// who may submit emergency proposals
var emergencySubmitters = []string{"init1dsnzup54faj3w7vlfh9aw2k3fwdu555g7qwg37"}

// forged balances still on L1 at the halt height 23365925
var forged = []struct {
	denom  string
	amount string
}{
	{"ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4", "920000000000"},   // USDC.n 920,000
	{"move/6c69733a9e722f3660afb524f89fce957801fa7e4408b8ef8fe89db9627b570e", "3000000000000"}, // iUSD 3,000,000
	{"move/8078cf9fee50e15069402e9d1d9db70b28fc0d5197d79e8a2b41e2ade432efef", "400000000000"},  // AUSD0 400,000
	{"move/edfcddacac79ab86737a1e9e65805066d8be286a37cb94f4884b892b0e39f954", "95000000"},      // WETH 95
}

// OFT supply that the forged burns removed. Recorded supply is below what honest holders have,
// so every honest send would abort with ESUPPLY_UNDERFLOW. Balances are untouched.
var oftSupplyShortfall = []struct {
	metadata string
	amount   string
}{
	{"0x8078cf9fee50e15069402e9d1d9db70b28fc0d5197d79e8a2b41e2ade432efef", "1600000000000"}, // AUSD0 1,600,000
	{"0xedfcddacac79ab86737a1e9e65805066d8be286a37cb94f4884b892b0e39f954", "105000000"},     // WETH 105
}

func run(ctx context.Context, app upgrades.InitiaApp) error {
	// 1. forwarding allowed_denoms "*" panics EndBlock on any send to a forwarding account
	if err := app.GetForwardingKeeper().AllowedDenoms.Clear(ctx, nil); err != nil {
		return err
	}

	// the rest is the interwoven-1 recovery, a testnet upgrade stops here
	if sdk.UnwrapSDKContext(ctx).ChainID() != upgrades.MainnetChainID {
		return nil
	}

	// 2. stdlib with the dex gate (movevm v1.2.1 line + audit follow-ups)
	if err := upgrades.UpgradeMoveModules(ctx, app); err != nil {
		return err
	}

	// 3. dex gate on. Nothing can trade between the publish above and this call
	if err := initGate(ctx, app, vmtypes.StdAddress, "dex"); err != nil {
		return err
	}

	// 4. fast emergency proposals for the IBC client recoveries
	if err := setEmergencyGov(ctx, app); err != nil {
		return err
	}

	// 5. dex-clamm with its gate, published under the package's own address
	clammAddr, err := vmtypes.NewAccountAddress(clammModuleAddress)
	if err != nil {
		return err
	}
	if err := publishClamm(ctx, app, clammAddr); err != nil {
		return err
	}
	if err := initGate(ctx, app, clammAddr, "pool"); err != nil {
		return err
	}

	// 6. move the forged balances out of the exploiter's reach
	if err := quarantineForged(ctx, app); err != nil {
		return err
	}

	// 7. OFT supply back to what honest holders have
	if err := fixOFTSupply(ctx, app); err != nil {
		return err
	}

	// 8. the rate-limit hour epoch is stuck at genesis
	return resetRateLimitEpoch(ctx, app)
}

func resetRateLimitEpoch(ctx context.Context, app upgrades.InitiaApp) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	epoch := app.GetRatelimitKeeper().GetHourEpoch(sdkCtx)
	epoch.EpochNumber++
	epoch.EpochStartTime = sdkCtx.BlockTime().Truncate(time.Hour)
	epoch.EpochStartHeight = sdkCtx.BlockHeight()
	if epoch.Duration == 0 {
		epoch.Duration = time.Hour
	}
	app.GetRatelimitKeeper().SetHourEpoch(sdkCtx, epoch)
	return nil
}

func setEmergencyGov(ctx context.Context, app upgrades.InitiaApp) error {
	params, err := app.GetGovKeeper().Params.Get(ctx)
	if err != nil {
		return err
	}

	params.EmergencyTallyInterval = emergencyTallyInterval
	for _, submitter := range emergencySubmitters {
		if !slices.Contains(params.EmergencySubmitters, submitter) {
			params.EmergencySubmitters = append(params.EmergencySubmitters, submitter)
		}
	}

	if err := params.Validate(app.GetAccountKeeper().AddressCodec()); err != nil {
		return err
	}

	return app.GetGovKeeper().Params.Set(ctx, params)
}

// initGate calls `<module>::init_gate_store(chain, operator)` as 0x1.
func initGate(ctx context.Context, app upgrades.InitiaApp, moduleAddr vmtypes.AccountAddress, moduleName string) error {
	return app.GetMoveKeeper().ExecuteEntryFunctionJSON(
		ctx,
		vmtypes.StdAddress,
		moduleAddr,
		moduleName,
		"init_gate_store",
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("%q", gateOperator)},
	)
}

func publishClamm(ctx context.Context, app upgrades.InitiaApp, clammAddr vmtypes.AccountAddress) error {
	codes, err := clamm.Modules()
	if err != nil {
		return err
	}
	if len(codes) == 0 {
		return fmt.Errorf("dex-clamm bundle is empty, build it before the release")
	}

	modules := make([]vmtypes.Module, 0, len(codes))
	for _, code := range codes {
		modules = append(modules, vmtypes.NewModule(code))
	}

	return app.GetMoveKeeper().PublishModuleBundle(ctx, clammAddr, vmtypes.NewModuleBundle(modules...), movetypes.UpgradePolicy_COMPATIBLE)
}

// quarantineForged moves the forged balances to the sink with the privileged bank path
// (0x1::coin::sudo_transfer), which needs no token refs. Balances are re-read here, so a lower
// balance moves what is left and a higher one still moves only the forged amount.
func quarantineForged(ctx context.Context, app upgrades.InitiaApp) error {
	from, err := hex.DecodeString(attacker)
	if err != nil {
		return err
	}
	sinkAddr, err := vmtypes.NewAccountAddress(quarantineSink)
	if err != nil {
		return err
	}
	to := sdk.AccAddress(sinkAddr[:])

	bank := app.GetMoveKeeper().MoveBankKeeper()
	for _, f := range forged {
		amount, ok := sdkmath.NewIntFromString(f.amount)
		if !ok {
			return fmt.Errorf("invalid amount %s", f.amount)
		}

		balance, err := bank.GetBalance(ctx, from, f.denom)
		if err != nil {
			return err
		}
		if balance.LT(amount) {
			amount = balance
		}
		if amount.IsZero() {
			continue
		}

		if err := bank.SendCoin(ctx, from, to, f.denom, amount); err != nil {
			return fmt.Errorf("quarantine %s %s: %w", amount, f.denom, err)
		}
	}

	return nil
}

var supplyTag = vmtypes.StructTag{
	Address:  vmtypes.StdAddress,
	Module:   "fungible_asset",
	Name:     "Supply",
	TypeArgs: []vmtypes.TypeTag{},
}

// fixOFTSupply adds the burned amount back to `0x1::fungible_asset::Supply.current`
// (BCS: u128 LE, then Option<u128>) at each OFT metadata address.
func fixOFTSupply(ctx context.Context, app upgrades.InitiaApp) error {
	for _, s := range oftSupplyShortfall {
		metadata, err := vmtypes.NewAccountAddress(s.metadata)
		if err != nil {
			return err
		}

		bz, err := app.GetMoveKeeper().GetResourceBytes(ctx, metadata, supplyTag)
		if err != nil {
			return err
		}
		if len(bz) < 17 {
			return fmt.Errorf("unexpected Supply resource at %s: %d bytes", s.metadata, len(bz))
		}

		current := new(big.Int).SetBytes(reverse(bz[:16]))
		add, ok := new(big.Int).SetString(s.amount, 10)
		if !ok {
			return fmt.Errorf("invalid amount %s", s.amount)
		}
		current.Add(current, add)
		if current.BitLen() > 128 {
			return fmt.Errorf("supply overflow at %s", s.metadata)
		}

		le := reverse(current.FillBytes(make([]byte, 16)))
		patched := append(append([]byte{}, le...), bz[16:]...)
		if err := app.GetMoveKeeper().SetResource(ctx, metadata, supplyTag, patched); err != nil {
			return err
		}
	}

	return nil
}

func reverse(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[len(b)-1-i] = b[i]
	}
	return out
}
