package keeper

import (
	"github.com/cosmos/gogoproto/proto"
	govtypes "github.com/initia-labs/initia/x/gov/types"
)

type ProtoSet struct {
	Request  proto.Message
	Response proto.Message
}

type StargateQueryWhiteList map[string]ProtoSet

func DefaultStargateQueryWhiteList() StargateQueryWhiteList {
	res := make(StargateQueryWhiteList)
	res["/initia.gov.v1.Query/Proposal"] = ProtoSet{
		Request:  &govtypes.QueryProposalRequest{},
		Response: &govtypes.QueryProposalResponse{},
	}
	return res
}

func EmptyStargateQueryWhitelist() StargateQueryWhiteList {
	return make(StargateQueryWhiteList)
}
