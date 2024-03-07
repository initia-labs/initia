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

func Test_decodePacketData(t *testing.T) {
	data := NonFungibleTokenPacketData{
		ClassId:   "class_id",
		ClassUri:  "class_uri",
		ClassData: "class_data",
		TokenIds:  []string{"token_id_1", "token_id_2"},
		TokenUris: []string{"token_uri_1", "token_uri_2"},
		TokenData: []string{"token_data_1", "token_data_2"},
		Sender:    "sender",
		Receiver:  "receiver",
		Memo:      "memo",
	}

	// snake case
	snakeJsonStr := `{
		"class_id": "class_id",
		"class_uri": "class_uri",
		"class_data": "class_data",
		"token_ids": ["token_id_1", "token_id_2"],
		"token_uris": ["token_uri_1", "token_uri_2"],
		"token_data": ["token_data_1", "token_data_2"],
		"sender": "sender",
		"receiver": "receiver",
		"memo": "memo"
	}`

	res, err := DecodePacketData([]byte(snakeJsonStr), "ics721")
	require.NoError(t, err)
	require.Equal(t, data, res)

	// camel case
	camelJsonStr := `{
		"classId": "class_id",
		"classUri": "class_uri",
		"classData": "class_data",
		"tokenIds": ["token_id_1", "token_id_2"],
		"tokenUris": ["token_uri_1", "token_uri_2"],
		"tokenData": ["token_data_1", "token_data_2"],
		"sender": "sender",
		"receiver": "receiver",
		"memo": "memo"
	}`

	camelRes, err := DecodePacketData([]byte(camelJsonStr), "wasm.contract")
	require.NoError(t, err)
	require.Equal(t, data, camelRes)
}

func Test_GetBytes(t *testing.T) {
	data := NonFungibleTokenPacketData{
		ClassId:   "class_id",
		ClassUri:  "class_uri",
		ClassData: "class_data",
		TokenIds:  []string{"token_id_1", "token_id_2"},
		TokenUris: []string{"token_uri_1", "token_uri_2"},
		TokenData: []string{"token_data_1", "token_data_2"},
		Sender:    "sender",
		Receiver:  "receiver",
		Memo:      "memo",
	}

	// case wasm
	wasmPortID := wasmPortPrefix + "contract"
	_data, err := DecodePacketData(data.GetBytes(wasmPortID), wasmPortID)
	require.NoError(t, err)
	require.Equal(t, data, _data)

	// case normal
	portID := "ics721"
	_data, err = DecodePacketData(data.GetBytes(portID), portID)
	require.NoError(t, err)
	require.Equal(t, data, _data)

	// case mixed
	_, err = DecodePacketData(data.GetBytes(wasmPortID), portID)
	require.Error(t, err)
	_, err = DecodePacketData(data.GetBytes(portID), wasmPortID)
	require.Error(t, err)
}
