package types

import (
	"context"

	"cosmossdk.io/core/address"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// AccountKeeper defines the contract required for account APIs.
type AccountKeeper interface {
	AddressCodec() address.Codec
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI

	SetAccount(ctx context.Context, acc sdk.AccountI)
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

type NftKeeper interface {
	CreateOrUpdateClass(ctx context.Context, classId, classUri, classData string) error
	Transfers(ctx context.Context, sender, escrowAddress sdk.AccAddress, classId string, tokenIds []string) error
	Burns(ctx context.Context, owner sdk.AccAddress, classId string, tokenIds []string) error
	Mints(ctx context.Context, receiver sdk.AccAddress, classId string, tokenIds, tokenUris []string, tokenData []string) error
	GetClassInfo(ctx context.Context, classId string) (className string, classUri string, classData string, err error)
	GetTokenInfos(ctx context.Context, classId string, tokenIds []string) (tokenUris []string, tokenData []string, err error)
}

// ICS4Wrapper defines the expected ICS4Wrapper for middleware
type ICS4Wrapper interface {
	SendPacket(ctx sdk.Context, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error)
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
	HasChannel(ctx sdk.Context, portID, channelID string) bool
}

// ClientKeeper defines the expected IBC client keeper
type ClientKeeper interface {
	GetClientConsensusState(ctx sdk.Context, clientID string) (connection ibcexported.ConsensusState, found bool)
}

// ConnectionKeeper defines the expected IBC connection keeper
type ConnectionKeeper interface {
	GetConnection(ctx sdk.Context, connectionID string) (connection connectiontypes.ConnectionEnd, found bool)
}
