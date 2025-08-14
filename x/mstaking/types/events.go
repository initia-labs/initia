package types

// staking module event types
const (
	EventTypeCompleteUnbonding         = "complete_unbonding"
	EventTypeCompleteRedelegation      = "complete_redelegation"
	EventTypeCreateValidator           = "create_validator"
	EventTypeEditValidator             = "edit_validator"
	EventTypeDelegate                  = "delegate"
	EventTypeUnbond                    = "unbond"
	EventTypeCancelUnbondingDelegation = "cancel_unbonding_delegation"
	EventTypeRedelegate                = "redelegate"
	EventTypeRegisterMigration         = "register_migration"
	EventTypeMigrateDelegation         = "migrate_delegation"

	AttributeKeyValidator           = "validator"
	AttributeKeyCommissionRate      = "commission_rate"
	AttributeKeySrcValidator        = "source_validator"
	AttributeKeyDstValidator        = "destination_validator"
	AttributeKeyDelegator           = "delegator"
	AttributeKeyCompletionTime      = "completion_time"
	AttributeKeyCreationHeight      = "creation_height"
	AttributeKeyNewShares           = "new_shares"
	AttributeKeyOriginShares        = "origin_shares"
	AttributeKeyLpDenomIn           = "lp_denom_in"
	AttributeKeyLpDenomOut          = "lp_denom_out"
	AttributeKeySwapContractAddress = "swap_contract_address"
	AttributeKeyDenomIn             = "denom_in"
	AttributeKeyDenomOut            = "denom_out"
	AttributeValueCategory          = ModuleName
)
