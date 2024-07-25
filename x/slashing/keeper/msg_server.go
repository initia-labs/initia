package keeper

import (
	"context"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the slashing MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

var _ types.MsgServer = msgServer{}

// UpdateParams implements MsgServer.UpdateParams method.
// It defines a method to update the x/slashing module parameters.
func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	if err := req.Params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// Unjail implements MsgServer.Unjail method.
// Validators must submit a transaction to unjail itself after
// having been jailed (and thus unbonded) for downtime
func (k msgServer) Unjail(ctx context.Context, msg *types.MsgUnjail) (*types.MsgUnjailResponse, error) {
	valAddr, valErr := sdk.ValAddressFromBech32(msg.ValidatorAddr)
	if valErr != nil {
		return nil, valErr
	}
	err := k.Keeper.Unjail(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.MsgUnjailResponse{}, nil
}
