package cli

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
)

func Test_readJSONStringArray(t *testing.T) {
	testCases := []struct {
		name   string
		s      string
		expRes []string
		expErr bool
	}{
		{"empty", "", []string{}, false},
		{"empty array", "[]", []string{}, false},
		{"single element", "[\"hello\"]", []string{"\"hello\""}, false},
		{"multiple elements", "[\"hello\",\"world\"]", []string{"\"hello\"", "\"world\""}, false},
		{"multiple elements with spaces", "[\"hello\", \"world\", true, 234, \"123\"]", []string{"\"hello\"", "\"world\"", "true", "234", "\"123\""}, false},
		{"invalid json", "[\"hello\", \"world\"", nil, true},
	}

	for _, tc := range testCases {
		res, err := readJSONStringArray(tc.s)
		if tc.expErr {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.expRes, res, tc.name)
		}
	}
}

func Test_decodeJSONStringArray(t *testing.T) {
	testCases := []struct {
		name   string
		ss     []string
		expRes []string
		expErr bool
	}{
		{"empty", []string{}, []string{}, false},
		{"empty array", []string{}, []string{}, false},
		{"single element", []string{"\"hello\""}, []string{"hello"}, false},
		{"multiple elements", []string{"\"hello\"", "\"world\""}, []string{"hello", "world"}, false},
		{"mismatched types", []string{"\"hello\"", "\"world\"", "true", "234", "\"123\""}, []string{}, true},
	}

	for _, tc := range testCases {
		res, err := decodeJSONStringArray[string](tc.ss)
		if tc.expErr {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.expRes, res, tc.name)
		}
	}
}

func Test_bcsSerializeArg(t *testing.T) {
	testCases := []struct {
		argType string
		arg     string
		expRes  []byte
		expErr  bool
		name    string
	}{
		{"raw_hex", "014c6f72656d00497073756d02", []byte{0x0d, 0x01, 0x4c, 0x6f, 0x72, 0x65, 0x6d, 0x00, 0x49, 0x70, 0x73, 0x75, 0x6d, 0x02}, false, "raw_hex \\x01Lorem\\x00Ipsum\\x02"},
		{"raw_base64", "AUxvcmVtAElwc3VtAg==", []byte{0x0d, 0x01, 0x4c, 0x6f, 0x72, 0x65, 0x6d, 0x00, 0x49, 0x70, 0x73, 0x75, 0x6d, 0x02}, false, "raw_base64 \\x01Lorem\\x00Ipsum\\x02"},
		{"u8", "1", []byte{1}, false, "u8 1"},
		{"u8", "-1", nil, true, "u8 -1"},
		{"u8", "256", nil, true, "u8 256"},
		{"u16", "65535", []byte{255, 255}, false, "u16 65535"},
		{"u16", "-1", nil, true, "u16 -1"},
		{"u16", "65536", nil, true, "u16 65536"},
		{"u16", "0x0001", []byte{0x01, 0x00}, false, "u32 0x0001"},
		{"u32", "4294967295", []byte{255, 255, 255, 255}, false, "u32 4294967295"},
		{"u32", "0x00010203", []byte{0x03, 0x02, 0x01, 0x00}, false, "u32 0x00010203"},
		{"u32", "-1", nil, true, "u32 -1"},
		{"u32", "4294967296", nil, true, "u32 4294967296"},
		{"u64", "-1", nil, true, "u64 -1"},
		{"u64", "18446744073709551615", []byte{255, 255, 255, 255, 255, 255, 255, 255}, false, "u64 18446744073709551615"}, //2^64-1
		{"u64", "18446744073709551616", nil, true, "u64 18446744073709551616"},                                             //2^64
		{"u128",
			"0x00010203040506070809101112131415",
			[]byte{0x15, 0x14, 0x13, 0x12, 0x11, 0x10, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00},
			false, "u128 0x00010203040506070809101112131415"},
		{"u256",
			"0x0001020304050607080910111213141516171819202122232425262728293031",
			[]byte{0x31, 0x30, 0x29, 0x28, 0x27, 0x26, 0x25, 0x24, 0x23, 0x22, 0x21, 0x20, 0x19, 0x18, 0x17, 0x16,
				0x15, 0x14, 0x13, 0x12, 0x11, 0x10, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00},
			false, "u256 0x0001020304050607080910111213141516171819202122232425262728293031",
		}, //2^64
		{"bool", "true", []byte{1}, false, "bool true"},
		{"bool", "false", []byte{0}, false, "bool false"},
		{"bool", "noboolean", nil, true, "bool noboolean"},
		{"string", "hello", []byte{0x05, 0x68, 0x65, 0x6c, 0x6c, 0x6f}, false, "string hello"},
		{"string", "", []byte{0}, false, "string empty"},
		{"address", "0x1", []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}, false, "address 0x1"},
		{"vector<bool>", "true,false,true", []byte{0x3, 0x1, 0x0, 0x1}, false, "vector<bool> true,false,true"},
		{"vector<u8>", "1,2,3", []byte{0x3, 0x1, 0x2, 0x3}, false, "vector<u8> 1,2,3"},
		{"vector<u8>", "256,256,-1", nil, true, "vector<u8> 256,256,-1"},
		{"vector<u8>", "", []byte{0}, false, "vector<u8> empty"},
		{"vector<u16>", "1,2,3", []byte{0x3, 0x1, 0x0, 0x2, 0x0, 0x3, 0x0}, false, "vector<u16> 1,2,3"},
		{"vector<u16>", "65536,65536,-1", nil, true, "vector<u16> 65536,65536,-1"},
		{"vector<address>", "0x1,0x2,0x3", []byte{0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3}, false, "vector<address> 0x1,0x2,0x3"},
		{"vector<string>", "hello,world", []byte{0x2, 0x5, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x5, 0x77, 0x6f, 0x72, 0x6c, 0x64}, false, "vector<string> hello,world"},
		{"vector<string>", "hello world,hello world", []byte{0x2, 0xb, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0xb, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64}, false, "vector<string> \"hello world,hello world\""},
		{"vector<string>", "", []byte{0}, false, "vector<string> empty"},
		{"vector<u128>", "0", []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, false, "vector<u128> 0"},
		{"vector<u128>", "1", []byte{0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, false, "vector<u128> 1"},
		{"badtype", "1", nil, true, "badtype 1"},
	}

	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	for _, tc := range testCases {
		s := NewSerializer()
		res, err := bcsSerializeArg(tc.argType, tc.arg, s, ac)
		if tc.expErr {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.expRes, res, tc.name)
		}
	}
}

func Test_DivideUint128String(t *testing.T) {
	testCases := []struct {
		uint128 string
		high    uint64
		low     uint64
		err     bool
	}{
		{"340282366920938463463374607431768211455", 0xffffffffffffffff, 0xffffffffffffffff, false},
		{"0", 0, 0, false},
		{"1", 0, 1, false},
	}

	for _, tc := range testCases {
		high, low, err := DivideUint128String(tc.uint128)
		if tc.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.high, high)
			require.Equal(t, tc.low, low)
		}
	}
}
