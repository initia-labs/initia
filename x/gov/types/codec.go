package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/gov interfaces
// and concrete types on the provided LegacyAmino codec. These types are used
// for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "gov/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgAddEmergencySubmitters{}, "gov/MsgAddEmergencySubmitters")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveEmergencySubmitters{}, "gov/MsgRemoveEmergencySubmitters")
	legacy.RegisterAminoMsg(cdc, &MsgActivateEmergencyProposal{}, "gov/MsgActivateEmergencyProposal")
	cdc.RegisterConcrete(Params{}, "gov/Params", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgAddEmergencySubmitters{},
		&MsgRemoveEmergencySubmitters{},
		&MsgActivateEmergencyProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
