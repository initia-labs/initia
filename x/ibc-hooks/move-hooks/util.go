package move_hooks

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	coreaddress "cosmossdk.io/core/address"
	moderrors "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

const senderPrefix = "ibc-move-hook-intermediary"

// DeriveIntermediateSender compute intermediate sender address
// Bech32(Hash(Hash("ibc-hook-intermediary") + channelID/sender))
func DeriveIntermediateSender(channel, originalSender string) string {
	senderStr := fmt.Sprintf("%s/%s", channel, originalSender)
	senderAddr := sdk.AccAddress(address.Hash(senderPrefix, []byte(senderStr)))
	return senderAddr.String()
}

func isIcs20Packet(packetData []byte) (isIcs20 bool, ics20data transfertypes.FungibleTokenPacketData) {
	var data transfertypes.FungibleTokenPacketData
	decoder := json.NewDecoder(strings.NewReader(string(packetData)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return false, data
	}
	return true, data
}

func isIcs721Packet(packetData []byte) (isIcs721 bool, ics721data nfttransfertypes.NonFungibleTokenPacketData) {
	if data, err := nfttransfertypes.DecodePacketData(packetData); err != nil {
		return false, data
	} else {
		return true, data
	}
}

func parseHookData(memo string) (*HookData, bool, error) {
	if len(memo) == 0 {
		return nil, false, nil
	}

	var memoMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(memo), &memoMap); err != nil {
		return nil, false, nil
	}

	raw, ok := memoMap[moveHookMemoKey]
	if !ok {
		return nil, false, nil
	}

	var hookData HookData
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&hookData); err != nil {
		return nil, true, moderrors.Wrap(channeltypes.ErrInvalidPacket, err.Error())
	}

	return &hookData, true, nil
}

func validateReceiver(functionIdentifier string, receiver string, ac coreaddress.Codec) error {
	if receiver == functionIdentifier {
		return nil
	}

	hashedFunctionIdentifier := sha256.Sum256([]byte(functionIdentifier))
	hashedFunctionIdentifierString, err := ac.BytesToString(hashedFunctionIdentifier[:])
	if err != nil || receiver != hashedFunctionIdentifierString {
		return moderrors.Wrap(channeltypes.ErrInvalidPacket, "receiver is not properly set")
	}
	return nil
}

// newEmitErrorAcknowledgement creates a new error acknowledgement after having emitted an event with the
// details of the error.
func newEmitErrorAcknowledgement(err error) channeltypes.Acknowledgement {
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: fmt.Sprintf("ibc move hook error: %s", err.Error()),
		},
	}
}

// isAckError checks an IBC acknowledgement to see if it's an error.
func isAckError(appCodec codec.Codec, acknowledgement []byte) bool {
	var ack channeltypes.Acknowledgement
	if err := appCodec.UnmarshalJSON(acknowledgement, &ack); err == nil && !ack.Success() {
		return true
	}

	return false
}
