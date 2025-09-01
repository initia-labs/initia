package tendermintattestor_test

import (
	"time"

	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
)

func (suite *TMAttestorTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		msg            string
		consensusState *ibctmattestor.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			&ibctmattestor.ConsensusState{
				ConsensusState: &ibctm.ConsensusState{
					Timestamp:          suite.chainA.Coordinator.CurrentTime,
					Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
					NextValidatorsHash: suite.chainA.Vals.Hash(),
				},
			},
			true,
		},
		{
			"success with sentinel",
			&ibctmattestor.ConsensusState{
				ConsensusState: &ibctm.ConsensusState{
					Timestamp:          suite.chainA.Coordinator.CurrentTime,
					Root:               commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)),
					NextValidatorsHash: suite.chainA.Vals.Hash(),
				},
			},
			true,
		},
		{
			"root is nil",
			&ibctmattestor.ConsensusState{
				ConsensusState: &ibctm.ConsensusState{
					Timestamp:          suite.chainA.Coordinator.CurrentTime,
					Root:               commitmenttypes.MerkleRoot{},
					NextValidatorsHash: suite.chainA.Vals.Hash(),
				},
			},
			false,
		},
		{
			"root is empty",
			&ibctmattestor.ConsensusState{
				ConsensusState: &ibctm.ConsensusState{
					Timestamp:          suite.chainA.Coordinator.CurrentTime,
					Root:               commitmenttypes.MerkleRoot{},
					NextValidatorsHash: suite.chainA.Vals.Hash(),
				},
			},
			false,
		},
		{
			"nextvalshash is invalid",
			&ibctmattestor.ConsensusState{
				ConsensusState: &ibctm.ConsensusState{
					Timestamp:          suite.chainA.Coordinator.CurrentTime,
					Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
					NextValidatorsHash: []byte("hi"),
				},
			},
			false,
		},

		{
			"timestamp is zero",
			&ibctmattestor.ConsensusState{
				ConsensusState: &ibctm.ConsensusState{
					Timestamp:          time.Time{},
					Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
					NextValidatorsHash: suite.chainA.Vals.Hash(),
				},
			},
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		suite.Run(tc.msg, func() {
			// check just to increase coverage
			suite.Require().Equal(ibctmattestor.TendermintAttestor, tc.consensusState.ClientType())
			suite.Require().Equal(tc.consensusState.GetRoot(), tc.consensusState.Root)

			err := tc.consensusState.ValidateBasic()
			if tc.expectPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.msg)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.msg)
			}
		})
	}
}
