package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/ibc fetchprice interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgActivate{}, "fetchprice/MsgActivate")
	legacy.RegisterAminoMsg(cdc, &MsgDeactivate{}, "fetchprice/MsgDeactivate")
}

// RegisterInterfaces register the ibc fetchprice module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgActivate{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgDeactivate{})

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
