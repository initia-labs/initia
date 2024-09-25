package keeper

import (
	"bytes"
	"context"
	"errors"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"

	storetypes "cosmossdk.io/store/types"

	connecttypes "github.com/skip-mev/connect/v2/pkg/types"
)

var _ vmapi.GoAPI = &GoApi{}

type GoApi struct {
	Keeper
	ctx context.Context
}

func NewApi(k Keeper, ctx context.Context) GoApi {
	return GoApi{k, ctx}
}

// GetAccountInfo return account info (account number, sequence)
func (api GoApi) GetAccountInfo(addr vmtypes.AccountAddress) (bool /* found */, uint64 /* account number */, uint64 /* sequence */, uint8 /* account_type */, bool /* is_blocked */) {
	sdkAddr := types.ConvertVMAddressToSDKAddress(addr)
	if api.authKeeper.HasAccount(api.ctx, sdkAddr) {
		acc := api.authKeeper.GetAccount(api.ctx, sdkAddr)
		var accType uint8
		switch acc.(type) {
		case *authtypes.BaseAccount:
			accType = vmtypes.AccountType_Base
		case *types.ObjectAccount:
			accType = vmtypes.AccountType_Object
		case *types.TableAccount:
			accType = vmtypes.AccountType_Table
		default:
			// other account types are considered as module account
			accType = vmtypes.AccountType_Module
		}

		isBlocked := api.bankKeeper.BlockedAddr(sdkAddr)
		return true, acc.GetAccountNumber(), acc.GetSequence(), accType, isBlocked
	}

	return false, 0, 0, 0, false
}

// AmountToShare convert amount to share
func (api GoApi) AmountToShare(valBz []byte, metadata vmtypes.AccountAddress, amount uint64) (string, error) {
	valAddr, err := api.vc.StringToBytes(string(valBz))
	if err != nil {
		return "0", err
	}

	denom, err := types.DenomFromMetadataAddress(api.ctx, NewMoveBankKeeper(&api.Keeper), metadata)
	if err != nil {
		return "0", err
	}

	share, err := api.Keeper.AmountToShare(api.ctx, valAddr, sdk.NewCoin(denom, math.NewIntFromUint64(amount)))
	return share.String(), err
}

// ShareToAmount convert share to amount
func (api GoApi) ShareToAmount(valBz []byte, metadata vmtypes.AccountAddress, share string) (uint64, error) {
	valAddr, err := api.vc.StringToBytes(string(valBz))
	if err != nil {
		return 0, err
	}

	denom, err := types.DenomFromMetadataAddress(api.ctx, NewMoveBankKeeper(&api.Keeper), metadata)
	if err != nil {
		return 0, err
	}

	dec, err := math.LegacyNewDecFromStr(share)
	if err != nil {
		return 0, err
	}

	amount, err := api.Keeper.ShareToAmount(api.ctx, valAddr, sdk.NewDecCoinFromDec(denom, dec))
	return amount.Uint64(), err
}

// UnbondTimestamp return staking unbond time
func (api GoApi) UnbondTimestamp() uint64 {
	unbondingTime, err := api.StakingKeeper.UnbondingTime(api.ctx)

	// TODO - panic to error
	if err != nil {
		panic(err)
	}

	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	return uint64(sdkCtx.BlockTime().Unix()) + uint64(unbondingTime.Seconds())
}

func (api GoApi) GetPrice(pairId string) ([]byte, uint64, uint64, error) {
	cp, err := connecttypes.CurrencyPairFromString(pairId)
	if err != nil {
		return nil, 0, 0, err
	}

	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	price, err := api.oracleKeeper.GetPriceForCurrencyPair(sdkCtx, cp)
	if err != nil {
		return nil, 0, 0, err
	}

	priceBz, err := getLittleEndianBytes(price.Price)
	if err != nil {
		return nil, 0, 0, err
	}

	decimal, err := api.oracleKeeper.GetDecimalsForCurrencyPair(sdkCtx, cp)
	if err != nil {
		return nil, 0, 0, err
	}

	return priceBz, uint64(price.BlockTimestamp.Unix()), decimal, nil
}

func (api GoApi) Query(req vmtypes.QueryRequest, gasBalance uint64) ([]byte, uint64, error) {
	// use normal gas meter to meter gas consumption during query with max gas limit
	sdkCtx := sdk.UnwrapSDKContext(api.ctx).WithGasMeter(storetypes.NewGasMeter(gasBalance))

	res, err := api.Keeper.HandleVMQuery(sdkCtx, &req)
	if err != nil {
		return nil, sdkCtx.GasMeter().GasConsumed(), err
	}

	return res, sdkCtx.GasMeter().GasConsumed(), err
}

// convert math.Int to little endian bytes
// with u256 size assertion.
func getLittleEndianBytes(num math.Int) ([]byte, error) {
	b := num.BigInt().Bytes()
	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-i-1] = b[len(b)-i-1], b[i]
	}

	diff := 32 - len(b)
	if diff > 0 {
		b = append(b, bytes.Repeat([]byte{0}, diff)...)
	} else if diff < 0 {
		return nil, errors.New("exceed u256 range")
	}

	return b, nil
}
