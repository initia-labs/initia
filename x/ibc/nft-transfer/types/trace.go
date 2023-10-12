package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"

	"cosmossdk.io/errors"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

// ParseClassTrace parses a string with the ibc prefix (class id trace) and the base class id
// into a ClassTrace type.
//
// Examples:
//
// - "portidone/channel-0/0x123::nft_store::Extension" => ClassTrace{Path: "portidone/channel-0", BaseClassId: "0x123::nft_store::Extension"}
// - "portidone/channel-0/portidtwo/channel-1/0x123::nft_store::Extension" => ClassTrace{Path: "portidone/channel-0/portidtwo/channel-1", BaseClassId: "0x123::nft_store::Extension"}
// - "0x123::nft_store::Extension" => ClassTrace{Path: "", BaseClassId: "0x123::nft_store::Extension"}
func ParseClassTrace(rawClassId string) ClassTrace {
	denomSplit := strings.Split(rawClassId, "/")

	if denomSplit[0] == rawClassId {
		return ClassTrace{
			Path:        "",
			BaseClassId: rawClassId,
		}
	}

	path, baseClassId := extractPathAndBaseFromFullClassId(denomSplit)
	return ClassTrace{
		Path:        path,
		BaseClassId: baseClassId,
	}
}

// Hash returns the hex bytes of the SHA256 hash of the ClassTrace fields using the following formula:
//
// hash = sha256(tracePath + "/" + baseClassId)
func (dt ClassTrace) Hash() tmbytes.HexBytes {
	hash := sha256.Sum256([]byte(dt.GetFullClassIdPath()))
	return hash[:]
}

// GetPrefix returns the receiving denomination prefix composed by the trace info and a separator.
func (dt ClassTrace) GetPrefix() string {
	return dt.Path + "/"
}

// IBCClassId a nft class id for an ICS721 non fungible token in the format
// 'ibc/{hash(tracePath + baseClassId)}'. If the trace is empty, it will return the base class id.
func (dt ClassTrace) IBCClassId() string {
	if dt.Path != "" {
		return fmt.Sprintf("%s/%s", ClassIdPrefix, dt.Hash())
	}
	return dt.BaseClassId
}

// GetFullClassIdPath returns the full class id according to the ICS721 specification:
// tracePath + "/" + baseClassId
// If there exists no trace then the base class id is returned.
func (dt ClassTrace) GetFullClassIdPath() string {
	if dt.Path == "" {
		return dt.BaseClassId
	}
	return dt.GetPrefix() + dt.BaseClassId
}

// extractPathAndBaseFromFullClassId returns the trace path and the base class id from
// the elements that constitute the complete denom.
func extractPathAndBaseFromFullClassId(fullClassIdItems []string) (string, string) {
	var (
		path        []string
		baseClassId []string
	)

	length := len(fullClassIdItems)
	for i := 0; i < length; i = i + 2 {
		// The IBC specification does not guarentee the expected format of the
		// destination port or destination channel identifier. A short term solution
		// to determine base class id is to expect the channel identifier to be the
		// one ibc-go specifies. A longer term solution is to separate the path and base
		// class id in the ICS20 packet. If an intermediate hop prefixes the full denom
		// with a channel identifier format different from our own, the base class id
		// will be incorrectly parsed, but the token will continue to be treated correctly
		// as an IBC class id. The hash used to store the token internally on our chain
		// will be the same value as the base class id being correctly parsed.
		if i < length-1 && length > 2 && channeltypes.IsValidChannelID(fullClassIdItems[i+1]) {
			path = append(path, fullClassIdItems[i], fullClassIdItems[i+1])
		} else {
			baseClassId = fullClassIdItems[i:]
			break
		}
	}

	return strings.Join(path, "/"), strings.Join(baseClassId, "/")
}

func validateTraceIdentifiers(identifiers []string) error {
	if len(identifiers) == 0 || len(identifiers)%2 != 0 {
		return fmt.Errorf("trace info must come in pairs of port and channel identifiers '{portID}/{channelID}', got the identifiers: %s", identifiers)
	}

	// validate correctness of port and channel identifiers
	for i := 0; i < len(identifiers); i += 2 {
		if err := host.PortIdentifierValidator(identifiers[i]); err != nil {
			return errors.Wrapf(err, "invalid port ID at position %d", i)
		}
		if err := host.ChannelIdentifierValidator(identifiers[i+1]); err != nil {
			return errors.Wrapf(err, "invalid channel ID at position %d", i)
		}
	}
	return nil
}

// Validate performs a basic validation of the ClassTrace fields.
func (dt ClassTrace) Validate() error {
	// empty trace is accepted when token lives on the original chain
	switch {
	case dt.Path == "" && dt.BaseClassId != "":
		return nil
	case strings.TrimSpace(dt.BaseClassId) == "":
		return fmt.Errorf("base class id cannot be blank")
	}

	// NOTE: no base class id validation

	identifiers := strings.Split(dt.Path, "/")
	return validateTraceIdentifiers(identifiers)
}

// Traces defines a wrapper type for a slice of ClassTrace.
type Traces []ClassTrace

// Validate performs a basic validation of each denomination trace info.
func (t Traces) Validate() error {
	seenTraces := make(map[string]bool)
	for i, trace := range t {
		hash := trace.Hash().String()
		if seenTraces[hash] {
			return fmt.Errorf("duplicated denomination trace with hash %s", trace.Hash())
		}

		if err := trace.Validate(); err != nil {
			return errors.Wrapf(err, "failed denom trace %d validation", i)
		}
		seenTraces[hash] = true
	}
	return nil
}

var _ sort.Interface = Traces{}

// Len implements sort.Interface for Traces
func (t Traces) Len() int { return len(t) }

// Less implements sort.Interface for Traces
func (t Traces) Less(i, j int) bool { return t[i].GetFullClassIdPath() < t[j].GetFullClassIdPath() }

// Swap implements sort.Interface for Traces
func (t Traces) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// Sort is a helper function to sort the set of class id traces in-place
func (t Traces) Sort() Traces {
	sort.Sort(t)
	return t
}

// ValidatePrefixedClassId checks that the class id for an IBC non fungible token packet class id is correctly prefixed.
// The function will return no error if the given string follows one of the two formats:
//
//   - Prefixed class id: '{portIDN}/{channelIDN}/.../{portID0}/{channelID0}/baseClassId'
//   - Unprefixed class id: 'baseClassId'
//
// 'baseClassId' may or may not contain '/'s
func ValidatePrefixedClassId(classId string) error {
	classIdSplit := strings.Split(classId, "/")
	if classIdSplit[0] == classId && strings.TrimSpace(classId) != "" {
		// NOTE: no base class id validation
		return nil
	}

	if strings.TrimSpace(classIdSplit[len(classIdSplit)-1]) == "" {
		return errors.Wrap(ErrInvalidClassIdForNftTransfer, "base class id cannot be blank")
	}

	path, _ := extractPathAndBaseFromFullClassId(classIdSplit)
	if path == "" {
		// NOTE: base class id contains slashes, so no base class id validation
		return nil
	}

	identifiers := strings.Split(path, "/")
	return validateTraceIdentifiers(identifiers)
}

// ValidateIBCClassId validates that the given class id is either:
//
//   - A valid base class id (eg: 'uatom' or 'gamm/pool/1' as in https://github.com/cosmos/ibc-go/issues/894)
//   - A valid non fungible token representation (i.e 'ibc/{hash}') per ADR 001 https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-001-coin-source-tracing.md
func ValidateIBCClassId(classId string) error {
	if strings.TrimSpace(classId) == "" {
		return errors.Wrapf(ErrInvalidClassIdForNftTransfer, "class id should not be empty")
	}

	classIdSplit := strings.SplitN(classId, "/", 2)

	switch {
	case classId == ClassIdPrefix:
		return errors.Wrapf(ErrInvalidClassIdForNftTransfer, "class id should be prefixed with the format 'ibc/{hash(trace + \"/\" + %s)}'", classId)

	case len(classIdSplit) == 2 && classIdSplit[0] == ClassIdPrefix:
		if strings.TrimSpace(classIdSplit[1]) == "" {
			return errors.Wrapf(ErrInvalidClassIdForNftTransfer, "class id should be prefixed with the format 'ibc/{hash(trace + \"/\" + %s)}'", classId)
		}

		if _, err := ParseHexHash(classIdSplit[1]); err != nil {
			return errors.Wrapf(err, "invalid class id trace hash %s", classIdSplit[1])
		}
	}

	return nil
}

// ParseHexHash parses a hex hash in string format to bytes and validates its correctness.
func ParseHexHash(hexHash string) (tmbytes.HexBytes, error) {
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		return nil, err
	}

	if err := tmtypes.ValidateHash(hash); err != nil {
		return nil, err
	}

	return hash, nil
}

// RemoveClassPrefix returns the unprefixed classID.
// After the receiving chain receives the packet,if isAwayFromOrigin=false, it means that nft is moving
// in the direction of the original chain, and the portID/channelID prefix of the sending chain
// in trace.path needs to be removed
func RemoveClassPrefix(portID, channelID, classID string) (string, error) {
	classPrefix := GetClassIdPrefix(portID, channelID)
	if strings.HasPrefix(classID, classPrefix) {
		return strings.TrimPrefix(classID, classPrefix), nil
	}
	return "", fmt.Errorf("invalid class:%s, no class prefix: %s", classID, classPrefix)
}
