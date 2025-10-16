package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/reward/types"

	"cosmossdk.io/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := msg.Validate(ms.accKeeper.AddressCodec()); err != nil {
		return nil, err
	}

	if ms.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
	}

	// store params
	if err := ms.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// FundCommunityPool allows to send a portion of reward module account balance to the community pool
func (ms msgServer) FundCommunityPool(ctx context.Context, msg *types.MsgFundCommunityPool) (*types.MsgFundCommunityPoolResponse, error) {
	if err := msg.Validate(ms.accKeeper.AddressCodec()); err != nil {
		return nil, err
	}

	if ms.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
	}

	moduleAddr := ms.accKeeper.GetModuleAddress(types.ModuleName)
	if err := ms.communityPoolKeeper.FundCommunityPool(ctx, msg.Amount, moduleAddr); err != nil {
		return nil, err
	}

	return &types.MsgFundCommunityPoolResponse{}, nil
}
