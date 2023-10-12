package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	classId            = "transfer/initiachannel/0x123::nft_store::Extension"
	amount             = "100"
	largeAmount        = "18446744073709551616"                                                           // one greater than largest uint64 (^uint64(0))
	invalidLargeAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
)

// TestNonFungibleTokenPacketDataValidateBasic tests ValidateBasic for NonFungibleTokenPacketData
func TestNonFungibleTokenPacketDataValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData NonFungibleTokenPacketData
		expPass    bool
	}{
		{"valid packet", NewNonFungibleTokenPacketData(classId, "", "", []string{"1", "2", "3"}, []string{"", "", ""}, []string{"", "", ""}, addr1, addr2, ""), true},
		{"invalid classId", NewNonFungibleTokenPacketData("", "", "", []string{"1", "2", "3"}, []string{"", "", ""}, []string{"", "", ""}, addr1, addr2, ""), false},
		{"invalid empty token ids", NewNonFungibleTokenPacketData(classId, "", "", []string{}, []string{}, []string{}, addr1, addr2, ""), false},
		{"invalid token data", NewNonFungibleTokenPacketData(classId, "", "", []string{"1", "2"}, []string{"", ""}, []string{""}, addr1, addr2, ""), false},
		{"invalid token uris", NewNonFungibleTokenPacketData(classId, "", "", []string{"1", "2"}, []string{""}, []string{"", ""}, addr1, addr2, ""), false},
		{"missing sender address", NewNonFungibleTokenPacketData(classId, "", "", []string{"1", "2", "3"}, []string{"", "", ""}, []string{"", "", ""}, emptyAddr, addr2, ""), false},
		{"missing recipient address", NewNonFungibleTokenPacketData(classId, "", "", []string{"1", "2", "3"}, []string{"", "", ""}, []string{"", "", ""}, addr1, emptyAddr, ""), false},
	}

	for i, tc := range testCases {
		err := tc.packetData.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %v", i, err)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
