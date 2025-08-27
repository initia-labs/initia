package tendermintattestor

import (
	initiacryptocodec "github.com/initia-labs/initia/crypto/codec"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// RegisterInterfaces registers the tendermint concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*exported.ClientState)(nil),
		&ClientState{},
	)
	registry.RegisterImplementations(
		(*exported.ConsensusState)(nil),
		&ConsensusState{},
	)
	registry.RegisterImplementations(
		(*exported.ClientMessage)(nil),
		&Header{},
	)
	registry.RegisterImplementations(
		(*exported.ClientMessage)(nil),
		&Misbehaviour{},
	)
}

var PubkeyCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

func init() {
	initiacryptocodec.RegisterInterfaces(PubkeyCdc.InterfaceRegistry())
	cryptocodec.RegisterInterfaces(PubkeyCdc.InterfaceRegistry())
}
