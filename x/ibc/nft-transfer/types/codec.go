package types

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/cosmos/gogoproto/proto"
)

// RegisterLegacyAminoCodec registers the necessary x/ibc transfer interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgTransfer{}, "nft-transfer/MsgTransfer")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "nft-transfer/MsgUpdateParams")

	cdc.RegisterConcrete(Params{}, "nft-transfer/Params", nil)
}

// RegisterInterfaces register the ibc transfer module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgTransfer{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgUpdateParams{})

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	anyResolver = codectypes.NewInterfaceRegistry()
	protoCodec  = codec.NewProtoCodec(anyResolver)
)

// mustProtoMarshalJSON marshals a protobuf message to JSON and panics if there is an error.
//
// It takes a protobuf message as input and returns the JSON-encoded byte array.
// The function uses the provided anyResolver to resolve any protobuf Any types in the message.
// If there is an error during the marshaling process, the function panics.
//
// Parameters:
// - msg: The protobuf message to be marshaled.
//
// Returns:
// - []byte: The JSON-encoded byte array.
func mustProtoMarshalJSON(msg proto.Message) []byte {
	bz, err := protoMarshalJSON(msg, anyResolver)
	if err != nil {
		panic(err)
	}
	return bz
}

// protoMarshalJSON provides an auxiliary function to return Proto3 JSON encoded
// bytes of a message.
func protoMarshalJSON(msg proto.Message, resolver jsonpb.AnyResolver) ([]byte, error) {
	jm := &jsonpb.Marshaler{OrigName: false, EmitDefaults: false, AnyResolver: resolver}
	err := codectypes.UnpackInterfaces(msg, codectypes.ProtoJSONPacker{JSONPBMarshaler: jm})
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := jm.Marshal(buf, msg); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// unmarshalProtoJSON unmarshals JSON-encoded bytes into a protobuf message.
func unmarshalProtoJSON(bz []byte, ptr proto.Message) error {
	return protoCodec.UnmarshalJSON(bz, ptr)
}
