package types

const (
	EventTypeSetPermissionedRelayers = "update_channel_relayers"
	EventTypeHaltChannel             = "halt_channel"
	EventTypeResumeChannel           = "resume_channel"

	AttributeKeyPortId    = "port_id"
	AttributeKeyChannelId = "channel_id"
	AttributeKeyRelayers  = "relayers"
	AttributeKeyHaltedBy  = "halted_by"
	AttributeKeyResumedBy = "resumed_by"
)
