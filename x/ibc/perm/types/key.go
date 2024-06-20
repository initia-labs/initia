package types

const (
	// ModuleName defines the IBC perm name
	ModuleName = "permissionedchannelrelayer"

	// StoreKey is the store key string for IBC sft-transfer
	StoreKey = ModuleName

	// RouterKey is the message route for IBC sft-transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC sft-transfer
	QuerierRoute = ModuleName
)

var (
	// PermissionedRelayersPrefixKey defines the key to store the channel relayers in store
	PermissionedRelayersPrefixKey = []byte{0x01}

	RelayersPrefixKey = []byte{0x02}
)

// GetPermissionedRelayersKey return channel relayers key of the channel.
func GetPermissionedRelayersKey(channel string) []byte {
	return append(PermissionedRelayersPrefixKey, []byte(channel)...)
}
