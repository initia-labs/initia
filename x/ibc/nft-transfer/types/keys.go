package types

import (
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the IBC nft-transfer name
	ModuleName = "nonfungibletokentransfer"

	// Version defines the current version the IBC nft-transfer
	// module supports
	V1 = "ics721-1"

	// PortID is the default port id that nft-transfer module binds to
	PortID = "nft-transfer"

	// StoreKey is the store key string for IBC nft-transfer
	StoreKey = ModuleName

	// RouterKey is the message route for IBC nft-transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC nft-transfer
	QuerierRoute = ModuleName

	// ClassIdPrefix is the prefix used for internal class id representation.
	ClassIdPrefix = "ibc"
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
	// ClassTraceKey defines the key to store the class id trace info in store
	ClassTraceKey = []byte{0x02}
	// ClassDataPrefix defines the prefix for class data
	ClassDataPrefix = []byte{0x03}
	// TokenDataPrefix defines the prefix for token data
	TokenDataPrefix = []byte{0x04}

	ParamsKey = []byte{0x11}

	SupportedVersions = []string{V1}
)

// GetEscrowAddress returns the escrow address for the specified channel.
// The escrow address follows the format as outlined in ADR 028:
// https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-028-public-key-addresses.md
func GetEscrowAddress(portID, channelID string) sdk.AccAddress {
	// a slash is used to create domain separation between port and channel identifiers to
	// prevent address collisions between escrow addresses created for different channels
	contents := fmt.Sprintf("%s/%s", portID, channelID)

	// ADR 028 AddressHash construction
	preImage := []byte(V1)
	preImage = append(preImage, 0)
	preImage = append(preImage, contents...)
	hash := sha256.Sum256(preImage)
	return hash[:20]
}
