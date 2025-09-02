package types

const (
	// ModuleName is the name of the hook module
	ModuleName = "ibchooks"

	// StoreKey is the string store representation
	// not using the module name because of collisions with key "ibc"
	StoreKey = "hooks-for-ibc"

	MemStoreKey = "mem_" + ModuleName

	// TStoreKey is the string transient store representation
	TStoreKey = "transient_" + ModuleName

	// QuerierRoute is the querier route for the hook module
	QuerierRoute = ModuleName

	// RouterKey is the msg router key for the hook module
	RouterKey = ModuleName
)

// Keys for hook store
// Items are stored with the following key: values
var (
	ACLPrefix        = []byte{0x11} // prefix for allowed
	ParamsKey        = []byte{0x21} // prefix for parameters for module x/hook
	TransferFundsKey = []byte{0x31} // prefix for transfer funds
)
