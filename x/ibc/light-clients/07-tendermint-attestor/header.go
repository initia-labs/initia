package tendermintattestor

import (
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	tmlightclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

var _ exported.ClientMessage = (*Header)(nil)

// ConsensusState returns the updated consensus state associated with the header
func (h Header) ConsensusState() *ConsensusState {
	return &ConsensusState{
		ConsensusState: &tmlightclient.ConsensusState{
			Timestamp:          h.GetTime(),
			Root:               commitmenttypes.NewMerkleRoot(h.Header.Header.GetAppHash()),
			NextValidatorsHash: h.Header.Header.NextValidatorsHash,
		},
	}
}

// ClientType defines that the Header is a Tendermint consensus algorithm
func (Header) ClientType() string {
	return TendermintAttestor
}
