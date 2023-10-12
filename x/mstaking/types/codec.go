package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	govcodec "github.com/cosmos/cosmos-sdk/x/gov/codec"
	groupcodec "github.com/cosmos/cosmos-sdk/x/group/codec"
)

// RegisterLegacyAminoCodec registers the necessary x/staking interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgCreateValidator{}, "mstaking/MsgCreateValidator")
	legacy.RegisterAminoMsg(cdc, &MsgEditValidator{}, "mstaking/MsgEditValidator")
	legacy.RegisterAminoMsg(cdc, &MsgDelegate{}, "mstaking/MsgDelegate")
	legacy.RegisterAminoMsg(cdc, &MsgUndelegate{}, "mstaking/MsgUndelegate")
	legacy.RegisterAminoMsg(cdc, &MsgBeginRedelegate{}, "mstaking/MsgBeginRedelegate")
	legacy.RegisterAminoMsg(cdc, &MsgCancelUnbondingDelegation{}, "mstaking/MsgCancelUnbondingDelegation")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "mstaking/MsgUpdateParams")

	cdc.RegisterInterface((*isStakeAuthorization_Validators)(nil), nil)
	cdc.RegisterConcrete(&StakeAuthorization_AllowList{}, "mstaking/StakeAuthorization/AllowList", nil)
	cdc.RegisterConcrete(&StakeAuthorization_DenyList{}, "mstaking/StakeAuthorization/DenyList", nil)
	cdc.RegisterConcrete(&StakeAuthorization{}, "mstaking/StakeAuthorization", nil)
	cdc.RegisterConcrete(Params{}, "mstaking/Params", nil)
}

// RegisterInterfaces registers the x/staking interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateValidator{},
		&MsgEditValidator{},
		&MsgDelegate{},
		&MsgUndelegate{},
		&MsgBeginRedelegate{},
		&MsgCancelUnbondingDelegation{},
		&MsgUpdateParams{},
	)
	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&StakeAuthorization{},
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
