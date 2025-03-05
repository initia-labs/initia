package move_hooks

import (
	"encoding/json"
	"fmt"
	"strings"

	movetypes "github.com/initia-labs/initia/v1/x/move/types"
)

// A contract that sends an IBC transfer, may need to listen for the ACK from that packet.
// To allow contracts to listen on the ack of specific packets, we provide Ack callbacks.
//
// The contract, which wants to receive ack callback, have to implement two functions
// - ibc_ack
// - ibc_timeout
//
// public entry fun ibc_ack(
//   callback_id: u64,
//   success:     bool,
// )
//
// public entry fun ibc_timeout(
//   callback_id: u64,
// )
//

const (
	// The memo key is used to parse ics-20 or ics-712 memo fields.
	moveHookMemoKey = "move"

	functionNameAck     = "ibc_ack"
	functionNameTimeout = "ibc_timeout"
)

// AsyncCallback is data wrapper which is required
// when we implement async callback.
type AsyncCallback struct {
	// callback id should be issued form the executor contract
	Id            uint64 `json:"id"`
	ModuleAddress string `json:"module_address"`
	ModuleName    string `json:"module_name"`
}

// HookData defines a wrapper for move execute message
// and async callback.
type HookData struct {
	// Message is a move execute message which will be executed
	// at `OnRecvPacket` of receiver chain.
	Message *movetypes.MsgExecute `json:"message,omitempty"`

	// AsyncCallback is a callback message which will be executed
	// at `OnTimeoutPacket` and `OnAcknowledgementPacket` of
	// sender chain.
	AsyncCallback *AsyncCallback `json:"async_callback,omitempty"`
}

// intermediateCallback is used internally for JSON unmarshaling
type intermediateCallback struct {
	Id            interface{} `json:"id"`
	ModuleAddress string      `json:"module_address"`
	ModuleName    string      `json:"module_name"`
}

// UnmarshalJSON implements the json unmarshaler interface.
// It handles both string and numeric id formats and validates the module address.
func (a *AsyncCallback) UnmarshalJSON(bz []byte) error {
	var ic intermediateCallback
	if err := json.Unmarshal(bz, &ic); err != nil {
		return fmt.Errorf("failed to unmarshal AsyncCallback: %w", err)
	}

	// Validate required fields
	if ic.ModuleAddress == "" {
		return fmt.Errorf("module_address cannot be empty")
	}
	if ic.ModuleName == "" {
		return fmt.Errorf("module_name cannot be empty")
	}

	// Validate module address format
	if !strings.HasPrefix(ic.ModuleAddress, "0x") {
		return fmt.Errorf("invalid module_address format: must start with '0x'")
	}

	// Handle ID based on type with overflow checking
	switch v := ic.Id.(type) {
	case float64:
		if v < 0 || v >= float64(^uint64(0)) || v != float64(uint64(v)) {
			return fmt.Errorf("id value out of range or contains decimals")
		}
		a.Id = uint64(v)
	case string:
		var parsed float64
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return fmt.Errorf("invalid id format: %w", err)
		}
		if parsed < 0 || parsed >= float64(^uint64(0)) || parsed != float64(uint64(parsed)) {
			return fmt.Errorf("id value out of range or contains decimals")
		}
		a.Id = uint64(parsed)
	default:
		return fmt.Errorf("invalid id type: expected string or number")
	}

	a.ModuleAddress = ic.ModuleAddress
	a.ModuleName = ic.ModuleName
	return nil
}
