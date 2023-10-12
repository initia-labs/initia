package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseClassTrace(t *testing.T) {
	testCases := []struct {
		name     string
		classId  string
		expTrace ClassTrace
	}{
		{"empty classId", "", ClassTrace{}},
		{"base classId", "0x123::nft_store::Extension", ClassTrace{BaseClassId: "0x123::nft_store::Extension"}},
		{"base classId ending with '/'", "0x123::nft_store::Extension/", ClassTrace{BaseClassId: "0x123::nft_store::Extension/"}},
		{"trace info", "nft-transfer/channel-1/0x123::nft_store::Extension", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1"}},
		{"trace info with custom port", "custom-nft-transfer/channel-1/0x123::nft_store::Extension", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "custom-nft-transfer/channel-1"}},
		{"trace info with base classId ending in '/'", "nft-transfer/channel-1/0x123::nft_store::Extension/", ClassTrace{BaseClassId: "0x123::nft_store::Extension/", Path: "nft-transfer/channel-1"}},
		{"trace info with multiple port/channel pairs", "nft-transfer/channel-1/nft-transfer/channel-2/0x123::nft_store::Extension", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1/nft-transfer/channel-2"}},
		{"trace info with multiple custom ports", "custom-nft-transfer/channel-1/alternative-nft-transfer/channel-2/0x123::nft_store::Extension", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "custom-nft-transfer/channel-1/alternative-nft-transfer/channel-2"}},
		{"incomplete path", "nft-transfer/0x123::nft_store::Extension", ClassTrace{BaseClassId: "nft-transfer/0x123::nft_store::Extension"}},
		{"invalid path (1)", "nft-transfer//0x123::nft_store::Extension", ClassTrace{BaseClassId: "nft-transfer//0x123::nft_store::Extension", Path: ""}},
		{"invalid path (2)", "channel-1/nft-transfer/0x123::nft_store::Extension", ClassTrace{BaseClassId: "channel-1/nft-transfer/0x123::nft_store::Extension"}},
		{"invalid path (3)", "0x123::nft_store::Extension/transfer", ClassTrace{BaseClassId: "0x123::nft_store::Extension/transfer"}},
		{"invalid path (4)", "nft-transfer/channel-1", ClassTrace{BaseClassId: "nft-transfer/channel-1"}},
		{"invalid path (5)", "nft-transfer/channel-1/", ClassTrace{Path: "nft-transfer/channel-1"}},
		{"invalid path (6)", "nft-transfer/channel-1/transfer", ClassTrace{BaseClassId: "transfer", Path: "nft-transfer/channel-1"}},
		{"invalid path (7)", "nft-transfer/channel-1/nft-transfer/channel-2", ClassTrace{Path: "nft-transfer/channel-1/nft-transfer/channel-2"}},
		{"invalid path (8)", "nft-transfer/channelToA/0x123::nft_store::Extension", ClassTrace{BaseClassId: "nft-transfer/channelToA/0x123::nft_store::Extension", Path: ""}},
	}

	for _, tc := range testCases {
		trace := ParseClassTrace(tc.classId)
		require.Equal(t, tc.expTrace, trace, tc.name)
	}
}

func TestClassTrace_IBCClassId(t *testing.T) {
	testCases := []struct {
		name     string
		trace    ClassTrace
		expDenom string
	}{
		{"base classId", ClassTrace{BaseClassId: "0x123::nft_store::Extension"}, "0x123::nft_store::Extension"},
		{"trace info", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1"}, "ibc/78D9F145EB4146FAB632BC6274BCA8805A1C44FC2B57FF3333A25F0F80F3799B"},
	}

	for _, tc := range testCases {
		classId := tc.trace.IBCClassId()
		require.Equal(t, tc.expDenom, classId, tc.name)
	}
}

func TestClassTrace_Validate(t *testing.T) {
	testCases := []struct {
		name     string
		trace    ClassTrace
		expError bool
	}{
		{"base classId only", ClassTrace{BaseClassId: "0x123::nft_store::Extension"}, false},
		{"base classId only with single '/'", ClassTrace{BaseClassId: "erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA"}, false},
		{"base classId only with multiple '/'s", ClassTrace{BaseClassId: "gamm/pool/1"}, false},
		{"empty ClassTrace", ClassTrace{}, true},
		{"valid single trace info", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1"}, false},
		{"valid multiple trace info", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1/nft-transfer/channel-2"}, false},
		{"single trace identifier", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "transfer"}, true},
		{"invalid port ID", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "(transfer)/channel-1"}, true},
		{"invalid channel ID", ClassTrace{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/(channel-1)"}, true},
		{"empty base classId with trace", ClassTrace{BaseClassId: "", Path: "nft-transfer/channel-1"}, true},
	}

	for _, tc := range testCases {
		err := tc.trace.Validate()
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}

func TestTraces_Validate(t *testing.T) {
	testCases := []struct {
		name     string
		traces   Traces
		expError bool
	}{
		{"empty Traces", Traces{}, false},
		{"valid multiple trace info", Traces{{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1/nft-transfer/channel-2"}}, false},
		{
			"valid multiple trace info",
			Traces{
				{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1/nft-transfer/channel-2"},
				{BaseClassId: "0x123::nft_store::Extension", Path: "nft-transfer/channel-1/nft-transfer/channel-2"},
			},
			true,
		},
		{"empty base classId with trace", Traces{{BaseClassId: "", Path: "nft-transfer/channel-1"}}, true},
	}

	for _, tc := range testCases {
		err := tc.traces.Validate()
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}

func TestValidatePrefixedClassId(t *testing.T) {
	testCases := []struct {
		name     string
		classId  string
		expError bool
	}{
		{"prefixed classId", "nft-transfer/channel-1/0x123::nft_store::Extension", false},
		{"prefixed classId with '/'", "nft-transfer/channel-1/gamm/pool/1", false},
		{"empty prefix", "/0x123::nft_store::Extension", false},
		{"empty identifiers", "//0x123::nft_store::Extension", false},
		{"base classId", "0x123::nft_store::Extension", false},
		{"base classId with single '/'", "erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", false},
		{"base classId with multiple '/'s", "gamm/pool/1", false},
		{"invalid port ID", "(transfer)/channel-1/0x123::nft_store::Extension", true},
		{"empty classId", "", true},
		{"single trace identifier", "nft-transfer/", true},
	}

	for _, tc := range testCases {
		err := ValidatePrefixedClassId(tc.classId)
		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
		}
	}
}

func TestValidateIBCClassId(t *testing.T) {
	testCases := []struct {
		name     string
		classId  string
		expError bool
	}{
		{"classId with trace hash", "ibc/F54C67869D6548E0078EA5AD443B858272B04939E6AD9108E970D04887694437", false},
		{"base classId", "0x123::nft_store::Extension", false},
		{"base classId ending with '/'", "0x123::nft_store::Extension/", false},
		{"base classId with single '/'s", "gamm/pool/1", false},
		{"base classId with double '/'s", "gamm//pool//1", false},
		{"non-ibc prefix with hash", "notibc/F54C67869D6548E0078EA5AD443B858272B04939E6AD9108E970D04887694437", false},
		{"empty classId", "", true},
		{"classId 'ibc'", "ibc", true},
		{"classId 'ibc/'", "ibc/", true},
		{"invalid hash", "ibc/!@#$!@#", true},
	}

	for _, tc := range testCases {
		err := ValidateIBCClassId(tc.classId)
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}
