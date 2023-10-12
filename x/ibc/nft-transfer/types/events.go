package types

// IBC transfer events
const (
	EventTypeTimeout      = "timeout"
	EventTypePacket       = "non_fungible_token_packet"
	EventTypeNftTransfer  = "ibc_nft_transfer"
	EventTypeChannelClose = "channel_closed"
	EventTypeClassTrace   = "class_trace"

	AttributeKeyReceiver       = "receiver"
	AttributeKeyExtension      = "extension"
	AttributeKeyClassId        = "class_id"
	AttributeKeyTokenIds       = "token_ids"
	AttributeKeyRefundReceiver = "refund_receiver"
	AttributeKeyRefundClassId  = "refund_class_id"
	AttributeKeyRefundTokenIds = "refund_token_ids"
	AttributeKeyAckSuccess     = "success"
	AttributeKeyAck            = "acknowledgement"
	AttributeKeyAckError       = "error"
	AttributeKeyMemo           = "memo"
	AttributeKeyTraceHash      = "trace_hash"
)
