// noalias
package types

// Move module event types
const (
	EventTypePublish               = "publish"
	EventTypeExecute               = "execute"
	EventTypeScript                = "script"
	EventTypeMove                  = "move"
	EventTypeUpgradePolicy         = "upgrade_policy"
	EventTypeContractSharedRevenue = "contract_shared_revenue"

	AttributeKeySender       = "sender"
	AttributeKeyModuleAddr   = "module_addr"
	AttributeKeyModuleName   = "module_name"
	AttributeKeyFunctionName = "function_name"
	AttributeKeyCreator      = "creator"
	AttributeKeyRevenue      = "revenue"

	// move type event attributes
	AttributeKeyTypeTag = "type_tag"
	AttributeKeyData    = "data"

	// upgrade policy attributes
	AttributeKeyOriginPolicy = "origin_policy"
	AttributeKeyNewPolicy    = "new_policy"
)
