package tendermintattestor

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	tmlightclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

var _ clienttypes.ConsensusHost = (*ConsensusHost)(nil)

// ConsensusHost implements the 02-client clienttypes.ConsensusHost interface.
type ConsensusHost struct {
	*tmlightclient.ConsensusHost
}

// NewConsensusHost creates and returns a new ConsensusHost for tendermint consensus.
func NewConsensusHost(stakingKeeper clienttypes.StakingKeeper) clienttypes.ConsensusHost {
	return &ConsensusHost{
		ConsensusHost: tmlightclient.NewConsensusHost(stakingKeeper).(*tmlightclient.ConsensusHost),
	}
}

// GetSelfConsensusState implements the 02-client clienttypes.ConsensusHost interface.
func (c *ConsensusHost) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	return c.ConsensusHost.GetSelfConsensusState(ctx, height)
}

// ValidateSelfClient implements the 02-client clienttypes.ConsensusHost interface.
func (c *ConsensusHost) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	tmAttestorClient, ok := clientState.(*ClientState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "client must be a Tendermint client, expected: %T, got: %T", &ClientState{}, tmAttestorClient)
	}
	return c.ConsensusHost.ValidateSelfClient(ctx, tmAttestorClient.ClientState)
}
