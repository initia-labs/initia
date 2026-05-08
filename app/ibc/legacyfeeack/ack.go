package legacyfeeack

import (
	"encoding/json"
)

// feeMetadata mirrors ibc-go v8 `29-fee/types.Metadata`.
type feeMetadata struct {
	FeeVersion string `json:"fee_version"`
	AppVersion string `json:"app_version"`
}

// incentivizedAck mirrors ibc-go v8's `29-fee/types.IncentivizedAcknowledgement`.
type incentivizedAck struct {
	AppAcknowledgement    []byte `json:"app_acknowledgement"`
	ForwardRelayerAddress string `json:"forward_relayer_address"`
	UnderlyingAppSuccess  bool   `json:"underlying_app_success,omitempty"`
}

// IsFeeWrappedVersion determines whether the channel version string is the
// 29-fee Metadata envelope, e.g. `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
// Plain version strings like "ics20-1" / "ics721-1" return false.
func IsFeeWrappedVersion(version string) bool {
	_, ok := UnwrapAppVersion(version)

	return ok
}

// UnwrapAppVersion returns the inner `app_version` of a fee-wrapped channel
// version (e.g. "ics20-1" from the 29-fee envelope) and reports whether the
// input was actually fee-wrapped. Plain versions return ("", false).
func UnwrapAppVersion(version string) (string, bool) {
	var m feeMetadata
	if err := json.Unmarshal([]byte(version), &m); err != nil {
		return "", false
	}

	if m.FeeVersion == "" || m.AppVersion == "" {
		return "", false
	}

	return m.AppVersion, true
}

// TryUnwrapFeeAck peels the fee envelope off an incoming acknowledgement.
// Returns (inner, true) when the bytes are a valid IncentivizedAcknowledgement,
// otherwise (nil, false) so the caller can pass the original ack through unchanged.
func TryUnwrapFeeAck(ack []byte) ([]byte, bool) {
	var w incentivizedAck
	if err := json.Unmarshal(ack, &w); err != nil {
		return nil, false
	}

	if len(w.AppAcknowledgement) == 0 {
		return nil, false
	}

	return w.AppAcknowledgement, true
}

// WrapFeeAck encodes an inner app ack inside the 29-fee envelope so a v8
// counterparty's fee middleware can decode it. ForwardRelayerAddress is left
// empty because v10 has no fee module to register relayers.
func WrapFeeAck(inner []byte, success bool) []byte {
	out, _ := json.Marshal(incentivizedAck{
		AppAcknowledgement:   inner,
		UnderlyingAppSuccess: success,
	})

	return out
}
