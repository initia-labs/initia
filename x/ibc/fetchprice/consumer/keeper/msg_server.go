package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	consumertypes "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"
	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

type MsgServer struct {
	*Keeper
}

var _ consumertypes.MsgServer = MsgServer{}

// NewMsgServerImpl return MsgServer instance
func NewMsgServerImpl(k *Keeper) MsgServer {
	return MsgServer{k}
}

// FetchPrice implements types.MsgServer.
func (ms MsgServer) FetchPrice(ctx context.Context, msg *consumertypes.MsgFetchPrice) (*consumertypes.MsgFetchPriceResponse, error) {
	ac := ms.ac
	if err := msg.Validate(ac); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sequence, err := ms.sendFetchPrice(
		sdkCtx,
		msg.SourcePort,
		msg.SourceChannel,
		msg.CurrencyIds,
		msg.TimeoutHeight,
		msg.TimeoutTimestamp,
		msg.Memo,
	)
	if err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeFetchPrice,
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyCurrencyIds, strings.Join(msg.CurrencyIds, ",")),
		),
	})

	return &consumertypes.MsgFetchPriceResponse{Sequence: sequence}, nil
}
