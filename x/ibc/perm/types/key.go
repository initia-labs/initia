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
	// ChannelStatePrefix defines the key to store the channel state in store
	ChannelStatePrefix = []byte{0x01}
)

// GetChannelStateKey return channel relayers key of the channel.
func GetChannelStateKey(channel string) []byte {
	return append(ChannelStatePrefix, []byte(channel)...)
}
