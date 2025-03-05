package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/initia-labs/initia/v1/x/ibc/nft-transfer/types"
)

type MsgServer struct {
	*Keeper
}

var _ types.MsgServer = MsgServer{}

// NewMsgServerImpl return MsgServer instance
func NewMsgServerImpl(k *Keeper) MsgServer {
	return MsgServer{k}
}

// Transfer defines a rpc handler method for MsgTransfer.
func (ms MsgServer) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ac := ms.authKeeper.AddressCodec()
	if err := msg.Validate(ac); err != nil {
		return nil, err
	}

	sender, err := ac.StringToBytes(msg.Sender)
	if err != nil {
		return nil, err
	}

	sequence, err := ms.sendNftTransfer(
		ctx,
		msg.SourcePort,
		msg.SourceChannel,
		msg.ClassId,
		msg.TokenIds,
		sender,
		msg.Receiver,
		msg.TimeoutHeight,
		msg.TimeoutTimestamp,
		msg.Memo,
	)
	if err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info("IBC non-fungible token transfer", "class id", msg.ClassId, "token ids", strings.Join(msg.TokenIds, ","), "sender", msg.Sender, "receiver", msg.Receiver)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeNftTransfer,
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
			sdk.NewAttribute(types.AttributeKeyClassId, msg.ClassId),
			sdk.NewAttribute(types.AttributeKeyTokenIds, strings.Join(msg.TokenIds, ",")),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	return &types.MsgTransferResponse{Sequence: sequence}, nil
}

func (ms MsgServer) UpdateParams(context context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ac := ms.authKeeper.AddressCodec()
	if err := msg.Validate(ac); err != nil {
		return nil, err
	}

	if ms.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := ms.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
