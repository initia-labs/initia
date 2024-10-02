package keeper

import (
	"context"
	"time"

	"github.com/cosmos/gogoproto/proto"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/intertx/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
	icaControllerMsgServer icacontrollertypes.MsgServer
}

// NewMsgServerImpl creates and returns a new types.MsgServer, fulfilling the intertx Msg service interface
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k, icaControllerMsgServer: icacontrollerkeeper.NewMsgServerImpl(&k.icaControllerKeeper)}
}

// RegisterAccount implements the Msg/RegisterAccount interface
func (k msgServer) RegisterAccount(goCtx context.Context, msg *types.MsgRegisterAccount) (*types.MsgRegisterAccountResponse, error) {
	if err := msg.Validate(k.ac); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	icaMsg := icacontrollertypes.NewMsgRegisterInterchainAccount(msg.ConnectionId, msg.Owner, msg.Version)
	if _, err := k.icaControllerMsgServer.RegisterInterchainAccount(ctx, icaMsg); err != nil {
		return nil, err
	}

	return &types.MsgRegisterAccountResponse{}, nil
}

// SubmitTx implements the Msg/SubmitTx interface
func (k msgServer) SubmitTx(goCtx context.Context, msg *types.MsgSubmitTx) (*types.MsgSubmitTxResponse, error) {
	if err := msg.Validate(k.ac); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	data, err := icatypes.SerializeCosmosTx(k.cdc, []proto.Message{msg.GetTxMsg()}, icatypes.EncodingProtobuf)
	if err != nil {
		return nil, err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	// timeoutTimestamp set to max value with the unsigned bit shifted to satisfy hermes timestamp conversion
	// it is the responsibility of the auth module developer to ensure an appropriate timeout timestamp
	timeoutTimestamp := ctx.BlockTime().Add(time.Minute).UnixNano()

	icaMsg := icacontrollertypes.NewMsgSendTx(msg.Owner, msg.ConnectionId, uint64(timeoutTimestamp), packetData)
	_, err = k.icaControllerMsgServer.SendTx(ctx, icaMsg)
	if err != nil {
		return nil, err
	}

	return &types.MsgSubmitTxResponse{}, nil
}
