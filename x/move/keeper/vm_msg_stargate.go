package keeper

import (
	"context"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/gogoproto/proto"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

type StargateMsgWhiteList map[string]proto.Message

func DefaultStargateMsgWhiteList() StargateMsgWhiteList {
	res := make(StargateMsgWhiteList)
	res["/cosmos.gov.v1.Msg/Vote"] = &govtypes.MsgVote{}
	return res
}

func EmptyStargateMsgWhitelist() StargateMsgWhiteList {
	return make(StargateMsgWhiteList)
}

func (k Keeper) HandleVMStargateMsg(ctx context.Context, req *vmtypes.StargateMessage) (proto.Message, error) {
	protoReq, exists := k.vmStargateMsgWhiteList[req.Path]
	if !exists {
		return nil, types.ErrNotSupportedStargateQuery
	}
	protoReq.Reset()
	err := k.cdc.UnmarshalJSON(req.Data, protoReq)
	if err != nil {
		return nil, err
	}

	signer, err := k.ac.BytesToString(types.ConvertVMAddressToSDKAddress(req.Sender))
	if err != nil {
		return nil, err
	}

	if msg, ok := protoReq.(*types.MsgExecute); ok && msg.Sender != signer {
		return nil, types.ErrMalformedSenderCosmosMessage
	}

	if msg, ok := protoReq.(*types.MsgScript); ok && msg.Sender != signer {
		return nil, types.ErrMalformedSenderCosmosMessage
	}

	return protoReq, nil
}
