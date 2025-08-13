package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/x/authz"
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
	legacy.RegisterAminoMsg(cdc, &MsgRegisterMigration{}, "mstaking/MsgRegisterMigration")
	legacy.RegisterAminoMsg(cdc, &MsgMigrateDelegation{}, "mstaking/MsgMigrateDelegation")

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
		&MsgRegisterMigration{},
		&MsgMigrateDelegation{},
	)
	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&StakeAuthorization{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
