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
	// ChannelRelayerPrefixKey defines the key to store the channel relayer in store
	ChannelRelayerPrefixKey = []byte{0x01}
)

// GetChannelRelayerKey return channel relayer key of the channel.
func GetChannelRelayerKey(channel string) []byte {
	return append(ChannelRelayerPrefixKey, []byte(channel)...)
}
