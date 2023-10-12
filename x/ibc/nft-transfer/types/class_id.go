package types

import (
	"fmt"
	"strings"
)

// SenderChainIsSource returns false if the class id originally came
// from the receiving chain and true otherwise.
func SenderChainIsSource(sourcePort, sourceChannel, classId string) bool {
	// This is the prefix that would have been prefixed to the class id
	// on sender chain IF and only if the token originally came from the
	// receiving chain.

	return !ReceiverChainIsSource(sourcePort, sourceChannel, classId)
}

// ReceiverChainIsSource returns true if the class id originally came
// from the receiving chain and false otherwise.
func ReceiverChainIsSource(sourcePort, sourceChannel, classId string) bool {
	// The prefix passed in should contain the SourcePort and SourceChannel.
	// If  the receiver chain originally sent the token to the sender chain
	// the classId will have the sender's SourcePort and SourceChannel as the
	// prefix.

	voucherPrefix := GetClassIdPrefix(sourcePort, sourceChannel)
	return strings.HasPrefix(classId, voucherPrefix)
}

// GetClassIdPrefix returns the receiving class id prefix
func GetClassIdPrefix(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/", portID, channelID)
}

// GetPrefixedClassId returns the class id with the portID and channelID prefixed
func GetPrefixedClassId(portID, channelID, baseClassId string) string {
	return fmt.Sprintf("%s/%s/%s", portID, channelID, baseClassId)
}

// GetNftTransferClassId creates a nft transfer class id with the port ID and channel ID
// prefixed to the base classId.
func GetNftTransferClassId(portID, channelID, baseClassId string) string {
	classTrace := ParseClassTrace(GetPrefixedClassId(portID, channelID, baseClassId))
	return classTrace.IBCClassId()
}
