package types

import (
	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/gogoproto/proto"
	govtypes "github.com/initia-labs/initia/x/gov/types"
)

type VMQueryWhiteList struct {
	Custom   CustomQueryWhiteList
	Stargate StargateQueryWhiteList
}

func DefaultVMQueryWhiteList(ac address.Codec) VMQueryWhiteList {
	return VMQueryWhiteList{
		Custom:   DefaultCustomQueryWhiteList(ac),
		Stargate: DefaultStargateQueryWhiteList(),
	}
}

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

// ConvertProtoToJSONMarshal unmarshals the given bytes into a proto message and then marshals it to json.
func ConvertProtoToJSONMarshal(cdc codec.Codec, protoResponse proto.Message, bz []byte) ([]byte, error) {
	// unmarshal binary into stargate response data structure
	err := cdc.Unmarshal(bz, protoResponse)
	if err != nil {
		return nil, err
	}

	bz, err = cdc.MarshalJSON(protoResponse)
	if err != nil {
		return nil, err
	}

	protoResponse.Reset()
	return bz, nil
}

func ConvertJSONMarshalToProto(cdc codec.Codec, protoRequest proto.Message, bz []byte) ([]byte, error) {
	// unmarshal binary into stargate response data structure
	err := cdc.UnmarshalJSON(bz, protoRequest)
	if err != nil {
		return nil, err
	}

	bz, err = cdc.Marshal(protoRequest)
	if err != nil {
		return nil, err
	}

	protoRequest.Reset()
	return bz, nil
}
