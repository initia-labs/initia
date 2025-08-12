package move_hooks

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/x/move/types"

	coreaddress "cosmossdk.io/core/address"

	ibchookskeeper "github.com/initia-labs/initia/x/ibc-hooks/keeper"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
)

// DeriveIntermediateSender compute intermediate sender address
// Bech32(Hash(Hash("ibc-hook-intermediary") + channelID/sender))
func DeriveIntermediateSender(channel, originalSender string) string {
	senderStr := fmt.Sprintf("%s/%s", channel, originalSender)
	senderAddr := sdk.AccAddress(address.Hash(SenderPrefix, []byte(senderStr)))
	return senderAddr.String()
}

// IsIcs20Packet checks if the packet is ICS20 for v1
func IsIcs20Packet(packetData []byte, ics20Version, encoding string) (isIcs20 bool, ics20data transfertypes.InternalTransferRepresentation) {
	if ics20Version != transfertypes.V1 {
		return false, ics20data
	}

	ics20data, err := transfertypes.UnmarshalPacketData(packetData, ics20Version, encoding)
	if err != nil {
		return false, ics20data
	}

	return true, ics20data
}

// IsIcs721Packet checks if the packet is ICS721 for v1
func IsIcs721Packet(packetData []byte, ics721Version, encoding string) (isIcs721 bool, ics721data nfttransfertypes.NonFungibleTokenPacketData) {
	if ics721Version != nfttransfertypes.V1 {
		return false, ics721data
	}

	ics721data, err := nfttransfertypes.UnmarshalPacketData(packetData, ics721Version, encoding)
	if err != nil {
		return false, ics721data
	}

	return true, ics721data
}

func ValidateAndParseMemo(memo string) (
	isMoveRouted bool,
	hookData HookData,
	err error,
) {
	isMoveRouted, metadata := jsonStringHasKey(memo, MoveHookMemoKey)
	if !isMoveRouted {
		return
	}

	moveHookRaw := metadata[MoveHookMemoKey]

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

func ValidateReceiver(msg *movetypes.MsgExecute, receiver string, ac coreaddress.Codec) error {
	functionIdentifier := fmt.Sprintf("%s::%s::%s", msg.ModuleAddress, msg.ModuleName, msg.FunctionName)
	if receiver == functionIdentifier {
		return nil
	}

	hashedFunctionIdentifier := sha256.Sum256([]byte(functionIdentifier))
	hashedFunctionIdentifierString, err := ac.BytesToString(hashedFunctionIdentifier[:])
	if err != nil || receiver != hashedFunctionIdentifierString {
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

func IsAckError(appCodec codec.Codec, acknowledgement []byte) bool {
	var ack channeltypes.Acknowledgement
	if err := appCodec.UnmarshalJSON(acknowledgement, &ack); err == nil && !ack.Success() {
		return true
	}
	return false
}

func ExecMsg(ctx sdk.Context, msg *movetypes.MsgExecute, mk *movekeeper.Keeper, cdc coreaddress.Codec) (*movetypes.MsgExecuteResponse, error) {
	if err := msg.Validate(cdc); err != nil {
		return nil, err
	}
	moveMsgServer := movekeeper.NewMsgServerImpl(mk)
	res, err := moveMsgServer.Execute(ctx, msg)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// CheckACL checks if the given address is allowed to use IBC hooks
func CheckACL(ctx sdk.Context, ac coreaddress.Codec, hooksKeeper *ibchookskeeper.Keeper, addrStr string) (bool, error) {
	vmAddr, err := movetypes.AccAddressFromString(ac, addrStr)
	if err != nil {
		return false, err
	}

	sdkAddr := movetypes.ConvertVMAddressToSDKAddress(vmAddr)
	return hooksKeeper.GetAllowed(ctx, sdkAddr)
}

