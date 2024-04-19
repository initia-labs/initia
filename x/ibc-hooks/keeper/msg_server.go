package keeper

import (
	"context"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/initia-labs/initia/x/ibc-hooks/types"
)

type msgServer struct {
	*Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the hook MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(k *Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

// UpdateACL implements types.MsgServer.
func (ms msgServer) UpdateACL(ctx context.Context, msg *types.MsgUpdateACL) (*types.MsgUpdateACLResponse, error) {
	if msg.Authority != ms.authority {
		return nil, sdkerrors.ErrUnauthorized.Wrapf("expected `%s` but got `%s`", ms.authority, msg.Authority)
	}

	addr, err := ms.ac.StringToBytes(msg.Address)
	if err != nil {
		return nil, err
	}

	err = ms.SetAllowed(ctx, addr, msg.Allowed)
	if err != nil {
		return nil, err
	}

	return &types.MsgUpdateACLResponse{}, nil
}

// UpdateParams implements types.MsgServer.
func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != ms.authority {
		return nil, sdkerrors.ErrUnauthorized.Wrapf("expected `%s` but got `%s`", ms.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	// no validate; only bool types exists
	err := ms.Params.Set(ctx, msg.Params)
	if err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
