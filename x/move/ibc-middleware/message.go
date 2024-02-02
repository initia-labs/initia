package ibc_middleware

import (
	movetypes "github.com/initia-labs/initia/x/move/types"
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
	// at `OnRecvPacket` from receiver chain.
	Message movetypes.MsgExecute `json:"message"`

	// AsyncCallback is a callback message which will be executed
	// at `OnTimeoutPacket` and `OnAcknowledgementPacket` from
	// sender chain.
	AsyncCallback *AsyncCallback `json:"async_callback,omitempty"`
}
