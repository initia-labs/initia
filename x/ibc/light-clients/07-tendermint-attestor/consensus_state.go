package tendermintattestor

import (
	"time"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"

	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	tmlightclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

var _ exported.ConsensusState = (*ConsensusState)(nil)

// NewConsensusState creates a new ConsensusState instance.
func NewConsensusState(
	timestamp time.Time, root commitmenttypes.MerkleRoot, nextValsHash tmbytes.HexBytes,
) *ConsensusState {
	return &ConsensusState{
		tmlightclient.NewConsensusState(timestamp, root, nextValsHash),
	}
}

// FromTendermintConsensusState creates a new ConsensusState from a tendermint consensus state.
func FromTendermintConsensusState(cs *tmlightclient.ConsensusState) *ConsensusState {
	return &ConsensusState{
		ConsensusState: cs,
	}
}

// ClientType returns Tendermint attestor client type.
func (ConsensusState) ClientType() string {
	return TendermintAttestor
}
