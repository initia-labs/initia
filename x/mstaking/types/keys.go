package types

const (
	// ModuleName is the name of the staking module
	ModuleName = "mstaking"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// QuerierRoute is the querier route for the staking module
	QuerierRoute = ModuleName

	// RouterKey is the msg router key for the staking module
	RouterKey = ModuleName
)

var (
	// Keys for store prefixes
	// Last* values are constant during a block.
	LastValidatorConsPowersPrefix = []byte{0x11} // prefix for each key to a validator index, for bonded validators
	WhitelistedValidatorsPrefix   = []byte{0x12} // prefix for each key to a validator index, sorted by power

	ValidatorsPrefix                 = []byte{0x21} // prefix for each key to a validator
	ValidatorsByConsAddrPrefix       = []byte{0x22} // prefix for each key to a validator index, by pubkey
	ValidatorsByConsPowerIndexPrefix = []byte{0x23} // prefix for each key to a validator index, sorted by power

	DelegationsPrefix                    = []byte{0x31} // key for a delegation
	DelegationsByValIndexPrefix          = []byte{0x32}
	UnbondingDelegationsPrefix           = []byte{0x33} // key for an unbonding-delegation
	UnbondingDelegationsByValIndexPrefix = []byte{0x34} // prefix for each key for an unbonding-delegation, by validator operator
	RedelegationsPrefix                  = []byte{0x35} // key for a redelegation
	RedelegationsByValSrcIndexPrefix     = []byte{0x36} // prefix for each key for an redelegation, by source validator operator
	RedelegationsByValDstIndexPrefix     = []byte{0x37} // prefix for each key for an redelegation, by destination validator operator

	NextUnbondingIdKey    = []byte{0x41} // key for the counter for the incrementing id for UnbondingOperations
	UnbondingsIndexPrefix = []byte{0x42} // prefix for an index for looking up unbonding operations by their IDs
	UnbondingsTypePrefix  = []byte{0x43} // prefix for an index containing the type of unbonding operations

	UnbondingQueuePrefix    = []byte{0x51} // prefix for the timestamps in unbonding queue
	RedelegationQueuePrefix = []byte{0x52} // prefix for the timestamps in redelegations queue
	ValidatorQueuePrefix    = []byte{0x53} // prefix for the timestamps in validator queue

	HistoricalInfosPrefix = []byte{0x61} // prefix for the historical info

	ParamsKey = []byte{0x71} // prefix for parameters for module x/staking
)

// UnbondingType defines the type of unbonding operation
type UnbondingType uint32

const (
	UnbondingType_Undefined UnbondingType = iota
	UnbondingType_UnbondingDelegation
	UnbondingType_Redelegation
	UnbondingType_ValidatorUnbonding
)
