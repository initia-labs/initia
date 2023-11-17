package types

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var (
	EmergencyProposalsPrefix               = []byte{0x90}
	LastEmergencyProposalTallyTimestampKey = []byte{0x91}
)

// GetEmergencyProposalKey gets a specific proposal from the store
func GetEmergencyProposalKey(proposalID uint64) []byte {
	return append(EmergencyProposalsPrefix, govtypes.GetProposalIDBytes(proposalID)...)
}
