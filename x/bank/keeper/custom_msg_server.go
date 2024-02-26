package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	cosmosbank "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	customtypes "github.com/initia-labs/initia/x/bank/types"
)

var _ customtypes.MsgServer = customMsgServer{}

type customMsgServer struct {
	cosmosbank.Keeper
}

// NewCustomMsgServerImpl returns an implementation of the custom bank MsgServer interface
// for the provided Keeper.
func NewCustomMsgServerImpl(keeper cosmosbank.Keeper) customtypes.MsgServer {
	return &customMsgServer{Keeper: keeper}
}

// SetDenomMetadata implements customtypes.MsgServer.
func (ms customMsgServer) SetDenomMetadata(ctx context.Context, req *customtypes.MsgSetDenomMetadata) (*customtypes.MsgSetDenomMetadataResponse, error) {
	if ms.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.GetAuthority(), req.Authority)
	}

	if err := req.Metadata.Validate(); err != nil {
		return nil, err
	}

	ms.Keeper.SetDenomMetaData(ctx, req.Metadata)

	return &customtypes.MsgSetDenomMetadataResponse{}, nil
}
