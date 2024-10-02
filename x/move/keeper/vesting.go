package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/json"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

var _ types.VestingKeeper = VestingKeeper{}

// VestingKeeper implements move wrapper for types.VestingKeeper interface
type VestingKeeper struct {
	*Keeper
}

// NewVestingKeeper creates a new instance of VestingKeeper
func NewVestingKeeper(k *Keeper) VestingKeeper {
	return VestingKeeper{k}
}

// GetVestingHandle returns the vesting table handle for the given module and creator
// If the vesting token is not the base denom, it returns nil to skip the vesting voting power calculation.
func (vk VestingKeeper) GetVestingHandle(ctx context.Context, moduleAccAddr sdk.AccAddress, moduleName string, creatorAccAddr sdk.AccAddress) (*sdk.AccAddress, error) {
	denom, err := vk.getVestingTokenDenom(ctx, moduleAccAddr, moduleName, creatorAccAddr)
	if err != nil {
		return nil, err
	}

	params, err := vk.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// check if the vesting token is the base denom
	if params.BaseDenom != denom {
		return nil, nil
	}

	handle, err := vk.getVestingTableHandler(ctx, moduleAccAddr, moduleName, creatorAccAddr)
	if err != nil {
		return nil, err
	}

	return &handle, nil
}

func (vk VestingKeeper) getVestingTokenDenom(ctx context.Context, moduleAccAddr sdk.AccAddress, moduleName string, creatorAccAddr sdk.AccAddress) (string, error) {
	moduleAddr, err := vmtypes.NewAccountAddressFromBytes(moduleAccAddr.Bytes())
	if err != nil {
		return "", err
	}

	creatorAddr, err := vmtypes.NewAccountAddressFromBytes(creatorAccAddr.Bytes())
	if err != nil {
		return "", err
	}

	output, _, err := vk.executeViewFunction(
		ctx,
		moduleAddr,
		moduleName,
		types.FunctionNameVestingTokenMetadata,
		[]vmtypes.TypeTag{},
		[][]byte{[]byte(fmt.Sprintf("\"%s\"", creatorAddr))},
		true,
	)
	if err != nil {
		return "", err
	}

	// output should be a json encoded 32-byte hex string
	var vestingTokenMetadataHex string
	err = json.Unmarshal([]byte(output.Ret), &vestingTokenMetadataHex)
	if err != nil {
		return "", err
	}

	vestingTokenMetadata, err := hex.DecodeString(vestingTokenMetadataHex[2:])
	if err != nil {
		return "", err
	}

	return types.DenomFromMetadataAddress(ctx, NewMoveBankKeeper(vk.Keeper), vmtypes.AccountAddress(vestingTokenMetadata))
}

// getVestingTableHandler returns the vesting table handle for the given module and creator
func (vk VestingKeeper) getVestingTableHandler(ctx context.Context, moduleAccAddr sdk.AccAddress, moduleName string, creatorAccAddr sdk.AccAddress) (sdk.AccAddress, error) {
	moduleAddr, err := vmtypes.NewAccountAddressFromBytes(moduleAccAddr.Bytes())
	if err != nil {
		return nil, err
	}

	creatorAddr, err := vmtypes.NewAccountAddressFromBytes(creatorAccAddr.Bytes())
	if err != nil {
		return nil, err
	}

	output, _, err := vk.executeViewFunction(
		ctx,
		moduleAddr,
		moduleName,
		types.FunctionNameVestingTableHandle,
		[]vmtypes.TypeTag{},
		[][]byte{[]byte(fmt.Sprintf("\"%s\"", creatorAddr))},
		true,
	)
	if err != nil {
		return nil, err
	}

	// output should be a json encoded 32-byte hex string
	var tableHandleHexAddr string
	err = json.Unmarshal([]byte(output.Ret), &tableHandleHexAddr)
	if err != nil {
		return nil, err
	}

	return hex.DecodeString(tableHandleHexAddr[2:])
}

// GetUnclaimedVestedAmount returns the vested amount if the vesting is linear.
// In Initia vesting, there is a cliff period where the claimable vested amount is 0.
// However, we want to use this unclaimed vested amount to calculate the gov voting power.
func (vk VestingKeeper) GetUnclaimedVestedAmount(ctx context.Context, tableHandle, recipientAccAddr sdk.AccAddress) (math.Int, error) {
	recipientAddr, err := vmtypes.NewAccountAddressFromBytes(recipientAccAddr.Bytes())
	if err != nil {
		return math.ZeroInt(), err
	}

	// table handle retrieved from the view function, so it should be a valid address
	entry, err := vk.GetTableEntryBytes(ctx, vmtypes.AccountAddress(tableHandle), recipientAddr.Bytes())
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return math.ZeroInt(), nil
	}

	// decode value
	allocation, claimedAmount, startTime, vestingPeriod, err := types.ReadVesting(entry.ValueBytes)
	if err != nil {
		return math.ZeroInt(), err
	}

	curTime := uint64(sdk.UnwrapSDKContext(ctx).BlockTime().Unix())
	if curTime < startTime {
		return math.ZeroInt(), nil
	}
	if curTime >= startTime+vestingPeriod {
		return math.NewIntFromUint64(allocation).Sub(math.NewIntFromUint64(claimedAmount)), nil
	}

	vestedAmountInLinearVesting := math.NewIntFromUint64(allocation).
		Mul(math.NewIntFromUint64(curTime - startTime)).
		Quo(math.NewIntFromUint64(vestingPeriod))
	return vestedAmountInLinearVesting.Sub(math.NewIntFromUint64(claimedAmount)), nil
}
