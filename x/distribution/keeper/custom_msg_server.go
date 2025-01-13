package keeper

import (
	"context"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
)

type customMsgServer struct {
	Keeper
}

// NewCustomMsgServerImpl returns an implementation of the distribution CustomMsgServer interface
// for the provided Keeper.
func NewCustomMsgServerImpl(k Keeper) customtypes.MsgServer {
	return &customMsgServer{Keeper: k}
}

var _ customtypes.MsgServer = customMsgServer{}

func (k customMsgServer) UpdateParams(ctx context.Context, req *customtypes.MsgUpdateParams) (*customtypes.MsgUpdateParamsResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	if err := req.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := k.Params.Set(ctx, req.Params); err != nil {
		return nil, err
	}

	return &customtypes.MsgUpdateParamsResponse{}, nil
}

func (k customMsgServer) DepositValidatorRewardsPool(ctx context.Context, msg *customtypes.MsgDepositValidatorRewardsPool) (*customtypes.MsgDepositValidatorRewardsPoolResponse, error) {
	depositor, err := k.authKeeper.AddressCodec().StringToBytes(msg.Depositor)
	if err != nil {
		return nil, err
	}

	// deposit coins from depositor's account to the distribution module
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, depositor, types.ModuleName, msg.Amount); err != nil {
		return nil, err
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	validator, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if err := sdk.ValidateDenom(msg.Denom); err != nil {
		return nil, err
	}

	// Allocate tokens from the distribution module to the validator, which are
	// then distributed to the validator's delegators.
	reward := sdk.NewDecCoinsFromCoins(msg.Amount...)
	if err = k.AllocateTokensToValidatorPool(ctx, validator, msg.Denom, reward); err != nil {
		return nil, err
	}

	logger := k.Logger(ctx)
	logger.Info(
		"transferred from rewards to validator rewards pool",
		"depositor", msg.Depositor,
		"amount", msg.Amount.String(),
		"validator", msg.ValidatorAddress,
	)

	return &customtypes.MsgDepositValidatorRewardsPoolResponse{}, nil
}
