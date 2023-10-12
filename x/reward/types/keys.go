package types

const (
	// module name
	ModuleName = "reward"

	// StoreKey is the default store key for reward
	StoreKey = ModuleName

	// RouterKey is the msg router key for the reward module
	RouterKey = ModuleName
)

var (
	ParamsKey                = []byte{0x00} // Prefix for params key
	LastReleaseTimestampKey  = []byte{0x01} // Key to store last release timestamp
	LastDilutionTimestampKey = []byte{0x02} // Key to store last dilution timestamp
)
