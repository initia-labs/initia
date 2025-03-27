package keeper

import (
	"context"
	"time"

	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

type MsgServer struct {
	*Keeper
}

var _ types.MsgServer = MsgServer{}

// NewMsgServerImpl return MsgServer instance
func NewMsgServerImpl(k *Keeper) MsgServer {
	return MsgServer{k}
}

func (ms MsgServer) UpdateParams(context context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "update-params")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	if err := ms.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
