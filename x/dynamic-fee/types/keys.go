package types

const (
	// ModuleName is the name of the dynamic fee module
	ModuleName = "dynamicfee"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// TStoreKey is the string transient store representation
	TStoreKey = "transient_" + ModuleName

	// QuerierRoute is the querier route for the dynamic fee module
	QuerierRoute = ModuleName

	// RouterKey is the msg router key for the dynamic fee module
	RouterKey = ModuleName
)

// Keys for dynamic fee store
// Items are stored with the following key: values
var (
	ParamsKey = []byte{0x11} // key for parameters for module x/dynamicfee
)

// Transient store keys
var (
	AccumulatedGasKey = []byte{0x21} // key for accumulated gas for module x/dynamicfee
)
