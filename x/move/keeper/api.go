package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
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
func (api GoApi) GetAccountInfo(addr vmtypes.AccountAddress) (bool /* found */, uint64 /* account number */, uint64 /* sequence */, uint8 /* account_type */) {
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
		case *authtypes.ModuleAccount:
			accType = vmtypes.AccountType_Module
		default:
			// TODO - panic to error
			panic("unknown account type")
		}

		return true, acc.GetAccountNumber(), acc.GetSequence(), accType
	}

	return false, 0, 0, 0
}

// AmountToShare convert amount to share
func (api GoApi) AmountToShare(valBz []byte, metadata vmtypes.AccountAddress, amount uint64) (uint64, error) {
	valAddr, err := sdk.ValAddressFromBech32(string(valBz))
	if err != nil {
		return 0, err
	}

	denom, err := types.DenomFromMetadataAddress(api.ctx, NewMoveBankKeeper(&api.Keeper), metadata)
	if err != nil {
		return 0, err
	}

	share, err := api.Keeper.AmountToShare(api.ctx, valAddr, sdk.NewCoin(denom, math.NewIntFromUint64(amount)))
	return share.Uint64(), err
}

// ShareToAmount convert share to amount
func (api GoApi) ShareToAmount(valBz []byte, metadata vmtypes.AccountAddress, share uint64) (uint64, error) {
	valAddr, err := sdk.ValAddressFromBech32(string(valBz))
	if err != nil {
		return 0, err
	}

	denom, err := types.DenomFromMetadataAddress(api.ctx, NewMoveBankKeeper(&api.Keeper), metadata)
	if err != nil {
		return 0, err
	}

	amount, err := api.Keeper.ShareToAmount(api.ctx, valAddr, sdk.NewDecCoin(denom, math.NewIntFromUint64(share)))
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
	return uint64(sdkCtx.BlockTime().Unix()) + uint64(unbondingTime)
}
