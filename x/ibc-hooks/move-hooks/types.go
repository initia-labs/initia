package move_hooks

import (
	"encoding/json"

	movetypes "github.com/initia-labs/initia/x/move/types"
)

const (
	// The memo key is used to parse ics-20 or ics-712 memo fields.
	MoveHookMemoKey = "move"

	FunctionNameAck     = "ibc_ack"
	FunctionNameTimeout = "ibc_timeout"

	SenderPrefix = "ibc-move-hook-intermediary"
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

	// AsyncCallback is a optional structure which contains the
	// callback id and the contract address. These fields will be used
	// to callback to the contract when the packet is ack or timeout.
	AsyncCallback *AsyncCallback `json:"async_callback,omitempty"`
}

// GetAsyncCallbackAck returns a MsgExecute for ack callback
func (h HookData) GetAsyncCallbackAck(success bool) movetypes.MsgExecute {
	var msg movetypes.MsgExecute

	successBz, _ := json.Marshal(success)
	msg.Sender = h.AsyncCallback.ModuleAddress
	msg.ModuleAddress = h.AsyncCallback.ModuleAddress
	msg.ModuleName = h.AsyncCallback.ModuleName
	msg.FunctionName = FunctionNameAck
	msg.TypeArgs = []string{}
	msg.Args = [][]byte{
		successBz,
	}

	// callback id
	callbackIdBz, _ := json.Marshal(h.AsyncCallback.Id)
	msg.Args = append([][]byte{callbackIdBz}, msg.Args...)

	return msg
}

// GetAsyncCallbackTimeout returns a MsgExecute for timeout callback
func (h HookData) GetAsyncCallbackTimeout() movetypes.MsgExecute {
	var msg movetypes.MsgExecute

	msg.Sender = h.AsyncCallback.ModuleAddress
	msg.ModuleAddress = h.AsyncCallback.ModuleAddress
	msg.ModuleName = h.AsyncCallback.ModuleName
	msg.FunctionName = FunctionNameTimeout
	msg.TypeArgs = []string{}

	// callback id
	callbackIdBz, _ := json.Marshal(h.AsyncCallback.Id)
	msg.Args = [][]byte{callbackIdBz}

	return msg
}
