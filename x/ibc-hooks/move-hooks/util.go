package move_hooks

import (
	"encoding/json"
	"fmt"
	"strings"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

const senderPrefix = "ibc-move-hook-intermediary"

// deriveIntermediateSender compute intermediate sender address
// Bech32(Hash(Hash("ibc-hook-intermediary") + channelID/sender))
func deriveIntermediateSender(channel, originalSender string) string {
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
	var data nfttransfertypes.NonFungibleTokenPacketData
	decoder := json.NewDecoder(strings.NewReader(string(packetData)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return false, data
	}
	return true, data
}

func validateAndParseMemo(memo string) (
	isMoveRouted bool,
	hookData HookData,
	err error,
) {
	isMoveRouted, metadata := jsonStringHasKey(memo, moveHookMemoKey)
	if !isMoveRouted {
		return
	}

	moveHookRaw := metadata[moveHookMemoKey]

	// parse move raw bytes to execute message
	bz, err := json.Marshal(moveHookRaw)
	if err != nil {
		err = errors.Wrap(channeltypes.ErrInvalidPacket, err.Error())
		return
	}

	err = json.Unmarshal(bz, &hookData)
	if err != nil {
		err = errors.Wrap(channeltypes.ErrInvalidPacket, err.Error())
		return
	}

	return
}

func validateReceiver(msg *movetypes.MsgExecute, receiver string) error {
	functionIdentifier := fmt.Sprintf("%s::%s::%s", msg.ModuleAddress, msg.ModuleName, msg.FunctionName)
	if receiver != functionIdentifier {
		return errors.Wrap(channeltypes.ErrInvalidPacket, "receiver is not properly set")
	}

	return nil
}

// jsonStringHasKey parses the memo as a json object and checks if it contains the key.
func jsonStringHasKey(memo, key string) (found bool, jsonObject map[string]interface{}) {
	jsonObject = make(map[string]interface{})

	// If there is no memo, the packet was either sent with an earlier version of IBC, or the memo was
	// intentionally left blank. Nothing to do here. Ignore the packet and pass it down the stack.
	if len(memo) == 0 {
		return false, jsonObject
	}

	// the jsonObject must be a valid JSON object
	err := json.Unmarshal([]byte(memo), &jsonObject)
	if err != nil {
		return false, jsonObject
	}

	// If the key doesn't exist, there's nothing to do on this hook. Continue by passing the packet
	// down the stack
	_, ok := jsonObject[key]
	if !ok {
		return false, jsonObject
	}

	return true, jsonObject
}

// newEmitErrorAcknowledgement creates a new error acknowledgement after having emitted an event with the
// details of the error.
func newEmitErrorAcknowledgement(ctx sdk.Context, err error) channeltypes.Acknowledgement {
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: fmt.Sprintf("ibc move hook error: %s", err.Error()),
		},
	}
}

// isAckError checks an IBC acknowledgement to see if it's an error.
// This is a replacement for ack.Success() which is currently not working on some circumstances
func isAckError(acknowledgement []byte) bool {
	var ackErr channeltypes.Acknowledgement_Error
	if err := json.Unmarshal(acknowledgement, &ackErr); err == nil && len(ackErr.Error) > 0 {
		return true
	}
	return false
}
