package cli

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"cosmossdk.io/core/address"
	sdkmath "cosmossdk.io/math"
	"github.com/aptos-labs/serde-reflection/serde-generate/runtime/golang/bcs"
	"github.com/aptos-labs/serde-reflection/serde-generate/runtime/golang/serde"
	"github.com/initia-labs/initia/x/move/types"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var NewSerializer = bcs.NewSerializer
var NewDeserializer = bcs.NewDeserializer

type argumentDecoder struct {
	// dec is the default decoder
	dec                func(string) ([]byte, error)
	asciiF, hexF, b64F bool
}

func newArgDecoder(def func(string) ([]byte, error)) *argumentDecoder {
	return &argumentDecoder{dec: def}
}

func (a *argumentDecoder) RegisterFlags(f *flag.FlagSet, argName string) {
	f.BoolVar(&a.asciiF, "ascii", false, "ascii encoded "+argName)
	f.BoolVar(&a.hexF, "hex", false, "hex encoded  "+argName)
	f.BoolVar(&a.b64F, "b64", false, "base64 encoded "+argName)
}

func (a *argumentDecoder) DecodeString(s string) ([]byte, error) {
	found := -1
	for i, v := range []*bool{&a.asciiF, &a.hexF, &a.b64F} {
		if !*v {
			continue
		}
		if found != -1 {
			return nil, errors.New("multiple decoding flags used")
		}
		found = i
	}
	switch found {
	case 0:
		return asciiDecodeString(s)
	case 1:
		return hex.DecodeString(s)
	case 2:
		return base64.StdEncoding.DecodeString(s)
	default:
		return a.dec(s)
	}
}

func asciiDecodeString(s string) ([]byte, error) {
	return []byte(s), nil
}

func BCSEncode(ac address.Codec, flagArgs []string) ([][]byte, error) {
	bcsArgs := [][]byte{}
	for _, flagArg := range flagArgs {
		ss := strings.Split(flagArg, ":")
		if len(ss) != 2 {
			return nil, fmt.Errorf(`expect "type:value" format but got "%s"`, flagArg)
		}

		moveType := ss[0]
		moveValue := ss[1]

		serializer := NewSerializer()
		bcsArg, err := bcsSerializeArg(moveType, moveValue, serializer, ac)
		if err != nil {
			return nil, err
		}

		bcsArgs = append(bcsArgs, bcsArg)
	}

	return bcsArgs, nil
}

func bcsSerializeArg(argType string, arg string, s serde.Serializer, ac address.Codec) ([]byte, error) {
	if arg == "" {
		err := s.SerializeBytes([]byte(arg))
		return s.GetBytes(), err
	}
	switch argType {
	case "raw_hex":
		decoded, err := hex.DecodeString(arg)
		if err != nil {
			return nil, err
		}

		err = s.SerializeBytes(decoded)
		return s.GetBytes(), err
	case "raw_base64":
		decoded, err := base64.StdEncoding.DecodeString(arg)
		if err != nil {
			return nil, err
		}

		err = s.SerializeBytes(decoded)
		return s.GetBytes(), err
	case "address", "object":
		accAddr, err := types.AccAddressFromString(ac, arg)
		if err != nil {
			return nil, err
		}

		err = s.IncreaseContainerDepth()
		if err != nil {
			return nil, err
		}
		for _, item := range accAddr {
			if err := s.SerializeU8(item); err != nil {
				return nil, err
			}
		}
		s.DecreaseContainerDepth()

		return s.GetBytes(), nil
	case "string":
		err := s.SerializeBytes([]byte(arg))
		return s.GetBytes(), err

	case "bool":
		if arg == "true" || arg == "True" {
			err := s.SerializeBool(true)
			return s.GetBytes(), err
		} else if arg == "false" || arg == "False" {
			err := s.SerializeBool(false)
			return s.GetBytes(), err
		} else {
			return nil, errors.New("unsupported bool value")
		}

	case "u8", "u16", "u32", "u64":
		bitSize, _ := strconv.Atoi(strings.TrimPrefix(argType, "u"))

		var num uint64
		var err error
		if strings.HasPrefix(arg, "0x") {
			num, err = strconv.ParseUint(strings.TrimPrefix(arg, "0x"), 16, bitSize)
			if err != nil {
				return nil, err
			}
		} else {
			num, err = strconv.ParseUint(arg, 10, bitSize)
			if err != nil {
				return nil, err
			}
		}

		switch argType {
		case "u8":
			_ = s.SerializeU8(uint8(num))
		case "u16":
			_ = s.SerializeU16(uint16(num))
		case "u32":
			_ = s.SerializeU32(uint32(num))
		case "u64":
			_ = s.SerializeU64(num)
		}
		return s.GetBytes(), nil

	case "u128":
		high, low, err := DivideUint128String(arg)
		if err != nil {
			return nil, err
		}
		_ = s.SerializeU128(serde.Uint128{
			Low:  low,
			High: high,
		})
		return s.GetBytes(), err
	case "u256":
		highHigh, highLow, high, low, err := DivideUint256String(arg)
		if err != nil {
			return nil, err
		}
		_ = s.SerializeU128(serde.Uint128{
			Low:  low,
			High: high,
		})
		_ = s.SerializeU128(serde.Uint128{
			Low:  highLow,
			High: highHigh,
		})
		return s.GetBytes(), nil
	case "biguint":
		n, err := sdkmath.ParseUint(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q as biguint", err)
		}

		// convert to little endian (bytes are cloned)
		bz := n.BigInt().Bytes()
		slices.Reverse(bz)

		err = s.SerializeBytes(bz)
		return s.GetBytes(), err
	case "bigdecimal":
		dec, err := sdkmath.LegacyNewDecFromStr(arg)
		if err != nil {
			return nil, err
		}

		// convert to little endian (bytes are cloned)
		bz := dec.BigInt().Bytes()
		slices.Reverse(bz)

		err = s.SerializeBytes(bz)
		return s.GetBytes(), err
	case "fixed_point32":
		dec, err := sdkmath.LegacyNewDecFromStr(arg)
		if err != nil {
			return nil, err
		}
		decstr := dec.MulInt64(4294967296).TruncateInt().String()
		return bcsSerializeArg("u64", decstr, s, ac)
	case "fixed_point64":
		dec, err := sdkmath.LegacyNewDecFromStr(arg)
		if err != nil {
			return nil, err
		}
		denominator := new(big.Int)
		denominator.SetString("18446744073709551616", 10)
		decstr := dec.MulInt(sdkmath.NewIntFromBigInt(denominator)).TruncateInt().String()
		return bcsSerializeArg("u128", decstr, s, ac)
	default:
		if vectorRegex.MatchString(argType) {
			vecType := getInnerType(argType)
			items := strings.Split(arg, ",")

			if err := s.SerializeLen(uint64(len(items))); err != nil {
				return nil, err
			}
			for _, item := range items {
				_, err := bcsSerializeArg(vecType, item, s, ac)
				if err != nil {
					return nil, err
				}
			}
			return s.GetBytes(), nil
		} else if optionRegex.MatchString(argType) {
			optionType := getInnerType(argType)
			if arg == "null" {
				if err := s.SerializeLen(0); err != nil {
					return nil, err
				}
				return s.GetBytes(), nil
			}
			if err := s.SerializeLen(1); err != nil {
				return nil, err
			}
			_, err := bcsSerializeArg(optionType, arg, s, ac)
			if err != nil {
				return nil, err
			}

			return s.GetBytes(), nil
		} else {
			return nil, errors.New("unsupported type arg")
		}
	}
}

var vectorRegex = regexp.MustCompile(`^vector<(.*)>$`)
var optionRegex = regexp.MustCompile(`^option<(.*)>$`)

func getInnerType(arg string) string {
	re := regexp.MustCompile(`<(.*)>`)
	return re.FindStringSubmatch(arg)[1]
}

func DivideUint128String(s string) (uint64, uint64, error) {
	n := new(big.Int)

	var ok bool
	if strings.HasPrefix(s, "0x") {
		_, ok = n.SetString(strings.TrimPrefix(s, "0x"), 16)
	} else {
		_, ok = n.SetString(s, 10)
	}
	if !ok {
		return 0, 0, fmt.Errorf("failed to parse %q as uint128", s)
	}

	if n.Sign() < 0 {
		return 0, 0, errors.New("value cannot be negative")
	} else if n.BitLen() > 128 {
		return 0, 0, errors.New("value overflows Uint128")
	}
	low := n.Uint64()
	high := n.Rsh(n, 64).Uint64()
	return high, low, nil
}

func DivideUint256String(s string) (uint64, uint64, uint64, uint64, error) {
	n := new(big.Int)

	var ok bool
	if strings.HasPrefix(s, "0x") {
		_, ok = n.SetString(strings.TrimPrefix(s, "0x"), 16)
	} else {
		_, ok = n.SetString(s, 10)
	}
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse %q as uint256", s)
	}

	if n.Sign() < 0 {
		return 0, 0, 0, 0, errors.New("value cannot be negative")
	} else if n.BitLen() > 256 {
		return 0, 0, 0, 0, errors.New("value overflows Uint128")
	}
	low := n.Uint64()
	high := n.Rsh(n, 64).Uint64()
	highLow := n.Rsh(n, 64).Uint64()
	highHigh := n.Rsh(n, 64).Uint64()
	return highHigh, highLow, high, low, nil
}

// Read json string array and decode it into the array of T
func ReadAndDecodeJSONStringArray[T any](cmd *cobra.Command, flagName string) (res []T, err error) {
	s, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return res, err
	}

	jsonStringArray, err := readJSONStringArray(s)
	if err != nil {
		return res, nil
	}

	return decodeJSONStringArray[T](jsonStringArray)
}

// Decode json string array into the array of T
func decodeJSONStringArray[T any](ss []string) ([]T, error) {
	if len(ss) == 0 {
		return []T{}, nil
	}

	res := make([]T, len(ss))

	for i, s := range ss {
		err := json.Unmarshal([]byte(s), &res[i])
		if err != nil {
			return res, err
		}
	}

	return res, nil
}

// Read json string array.
func ReadJSONStringArray(cmd *cobra.Command, flagName string) ([]string, error) {
	s, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return nil, err
	}

	return readJSONStringArray(s)
}

func readJSONStringArray(s string) ([]string, error) {
	if len(s) == 0 {
		return []string{}, nil
	}

	var args []any
	err := json.Unmarshal([]byte(s), &args)
	if err != nil {
		return nil, err
	}

	res := make([]string, len(args))
	for i, arg := range args {
		buf := new(strings.Builder)
		err := createNonEscapeJSONEncoder(buf).Encode(arg)
		if err != nil {
			return nil, err
		}

		res[i] = strings.TrimSpace(buf.String())
	}

	return res, err
}

func createNonEscapeJSONEncoder(buf *strings.Builder) *json.Encoder {
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "")
	return encoder
}
