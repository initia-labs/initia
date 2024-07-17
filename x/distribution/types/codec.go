package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/distribution interfaces
// and concrete types on the provided LegacyAmino codec. These types are used
// for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// amino is rejecting long type names, so we register under `distr`
	legacy.RegisterAminoMsg(cdc, &MsgDepositValidatorRewardsPool{}, "distr/MsgDepositValidatorRewardsPool")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "distribution/MsgUpdateParams")
	cdc.RegisterConcrete(Params{}, "distribution/Params", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgDepositValidatorRewardsPool{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
