// noalias
package types

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

// Move module event types
const (
	EventTypePublish               = "publish"
	EventTypeExecute               = "execute"
	EventTypeScript                = "script"
	EventTypeMove                  = "move"
	EventTypeUpgradePolicy         = "upgrade_policy"
	EventTypeContractSharedRevenue = "contract_shared_revenue"
	EventTypeSubmsg                = "submsg"

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

	// submessage event attributes
	AttributeKeySuccess = "success"
	AttributeKeyReason  = "reason"
)

// EmitContractEvents processes contract events from execution results and emits them to the context's EventManager.
// It tries to parse the event's JSON data and appends only scalar key-value pairs as event attributes.
// If parsing fails, the raw event data is emitted as the sole data attribute.
func EmitContractEvents(ctx sdk.Context, events []vmtypes.JsonEvent) {
	for _, event := range events {
		typeTag := event.TypeTag

		attributes := []sdk.Attribute{
			sdk.NewAttribute(AttributeKeyTypeTag, typeTag),
			sdk.NewAttribute(AttributeKeyData, event.EventData),
		}

		var dataEvent map[string]any
		if err := json.Unmarshal([]byte(event.EventData), &dataEvent); err == nil {
			keys := make([]string, 0, len(dataEvent))
			for k := range dataEvent {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := dataEvent[k]
				if isJSONContainer(v) {
					continue
				}

				// only add non-container values as separate attributes
				attributes = append(attributes, sdk.NewAttribute(k, stringifyJSONValue(v)))
			}
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(EventTypeMove, attributes...))
	}
}

func isJSONContainer(value any) bool {
	switch value.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func stringifyJSONValue(value any) string {
	switch val := value.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.Itoa(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
