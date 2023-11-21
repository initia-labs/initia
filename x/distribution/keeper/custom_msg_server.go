package keeper

import (
	"context"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
)

type customMsgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the distribution MsgServer interface
// for the provided Keeper.
func NewCustomMsgServerImpl(keeper Keeper) customtypes.MsgServer {
	return &customMsgServer{Keeper: keeper}
}

var _ customtypes.MsgServer = customMsgServer{}

func (k customMsgServer) UpdateParams(goCtx context.Context, req *customtypes.MsgUpdateParams) (*customtypes.MsgUpdateParamsResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &customtypes.MsgUpdateParamsResponse{}, nil
}
