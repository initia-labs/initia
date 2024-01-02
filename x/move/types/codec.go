package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// RegisterLegacyAminoCodec registers the move types and interface
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgPublish{}, "move/MsgPublish")
	legacy.RegisterAminoMsg(cdc, &MsgExecute{}, "move/MsgExecute")
	legacy.RegisterAminoMsg(cdc, &MsgScript{}, "move/MsgScript")
	legacy.RegisterAminoMsg(cdc, &MsgGovPublish{}, "move/MsgGovPublish")
	legacy.RegisterAminoMsg(cdc, &MsgGovExecute{}, "move/MsgGovExecute")
	legacy.RegisterAminoMsg(cdc, &MsgGovScript{}, "move/MsgGovScript")
	legacy.RegisterAminoMsg(cdc, &MsgWhitelist{}, "move/MsgWhitelist")
	legacy.RegisterAminoMsg(cdc, &MsgDelist{}, "move/MsgDelist")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "move/MsgUpdateParams")

	cdc.RegisterConcrete(&ObjectAccount{}, "move/ObjectAccount", nil)
	cdc.RegisterConcrete(&TableAccount{}, "move/TableAccount", nil)
	cdc.RegisterConcrete(&ExecuteAuthorization{}, "move/ExecuteAuthorization", nil)
	cdc.RegisterConcrete(&PublishAuthorization{}, "move/PublishAuthorization", nil)
	cdc.RegisterConcrete(Params{}, "move/Params", nil)
}

// RegisterInterfaces registers the x/market interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgPublish{},
		&MsgExecute{},
		&MsgScript{},
		&MsgGovPublish{},
		&MsgGovExecute{},
		&MsgGovScript{},
		&MsgWhitelist{},
		&MsgDelist{},
		&MsgUpdateParams{},
	)
	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&ExecuteAuthorization{},
		&PublishAuthorization{},
	)

	// auth account registration
	registry.RegisterImplementations(
		(*sdk.AccountI)(nil),
		&ObjectAccount{},
		&TableAccount{},
	)
	registry.RegisterImplementations(
		(*authtypes.GenesisAccount)(nil),
		&ObjectAccount{},
		&TableAccount{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
