package types

const (
	// SubModuleName defines the IBC fetchprice consumer name
	SubModuleName = "fetchpriceconsumer"

	StoreKey = SubModuleName
)

var (
	PortKey = []byte{0x01}

	// CurrencyPairPrefix defines the prefix key to store the currency pair in store
	CurrencyPairPrefix = []byte{0x02}
)
