package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	govcodec "github.com/cosmos/cosmos-sdk/x/gov/codec"
	groupcodec "github.com/cosmos/cosmos-sdk/x/group/codec"
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
		(*authtypes.AccountI)(nil),
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

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)

	// Register all Amino interfaces and concrete types on the authz  and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)
}
