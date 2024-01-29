package types

import (
	"cosmossdk.io/collections"
	icqtypes "github.com/cosmos/ibc-apps/modules/async-icq/v8/types"
)

const (
	// ModuleName is the name of the fetchprice module
	ModuleName = "fetchprice"

	// Version defines the current version the IBC ICQ module supports
	Version = icqtypes.Version

	// PortID is the default port id that fetchprice module binds to
	PortID = "fetchprice"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// TStoreKey is the string transient store representation
	TStoreKey = "transient_" + ModuleName

	// QuerierRoute is the querier route for the fetchprice module
	QuerierRoute = ModuleName

	// RouterKey is the msg router key for the fetchprice module
	RouterKey = ModuleName
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = collections.NewPrefix(0x1)

	// QueryAliveKey defines the key to store there is alive icq query
	QueryAliveKey = collections.NewPrefix(0x2)
	ParamsKey     = collections.NewPrefix(0x11)
)
