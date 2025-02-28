package keeper

import (
	"context"

	"cosmossdk.io/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

type customMsgServer struct {
	*Keeper
}

// NewCustomMsgServerImpl returns an implementation of the gov CustomMsgServer interface
// for the provided Keeper.
func NewCustomMsgServerImpl(keeper *Keeper) customtypes.MsgServer {
	return &customMsgServer{Keeper: keeper}
}

var _ customtypes.MsgServer = customMsgServer{}

func (k customMsgServer) UpdateParams(ctx context.Context, req *customtypes.MsgUpdateParams) (*customtypes.MsgUpdateParamsResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	if err := req.Params.Validate(k.authKeeper.AddressCodec()); err != nil {
		return nil, err
	}

	if err := k.Params.Set(ctx, req.Params); err != nil {
		return nil, err
	}

	return &customtypes.MsgUpdateParamsResponse{}, nil
}
