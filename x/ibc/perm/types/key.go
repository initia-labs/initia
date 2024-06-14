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
	// PermissionedRelayerPrefixKey defines the key to store the channel relayer in store
	PermissionedRelayersPrefixKey = []byte{0x01}

	RelayersPrefixKey = []byte{0x02}
)

// GetPermissionedRelayerKey return channel relayer key of the channel.
func GetPermissionedRelayerKey(channel string) []byte {
	return append(PermissionedRelayersPrefixKey, []byte(channel)...)
}
