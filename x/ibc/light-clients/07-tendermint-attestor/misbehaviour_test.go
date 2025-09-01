package tendermintattestor_test

import (
	"time"

	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctestingmock "github.com/cosmos/ibc-go/v8/testing/mock"
	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"
)

func (suite *TMAttestorTestSuite) TestMisbehaviour() {
	heightMinus1 := clienttypes.NewHeight(0, ibctesting.Height.RevisionHeight-1)

	misbehaviour := &ibctmattestor.Misbehaviour{
		Header1: &ibctmattestor.Header{Header: suite.chainA.LastHeader},
		Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, int64(suite.chainA.LastHeader.Header.Height), heightMinus1, suite.chainA.LastHeader.Header.Time, suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers),
	}

	suite.Require().Equal(ibctmattestor.TendermintAttestor, misbehaviour.ClientType())
}

func (suite *TMAttestorTestSuite) TestMisbehaviourValidateBasic() {
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	revisionHeight := int64(ibctesting.Height.RevisionHeight)

	altVal := tmtypes.NewValidator(altPubKey, revisionHeight)

	// Create alternative validator set with only altVal
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})

	// Create signer array and ensure it is in same order as bothValSet
	bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)

	altSignerArr := []tmtypes.PrivValidator{altPrivVal}

	heightMinus1 := clienttypes.NewHeight(0, ibctesting.Height.RevisionHeight-1)

	header := suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time, suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers)

	testCases := []struct {
		name                 string
		misbehaviour         *ibctmattestor.Misbehaviour
		malleateMisbehaviour func(misbehaviour *ibctmattestor.Misbehaviour) error
		expPass              bool
	}{
		{
			"valid fork misbehaviour, two headers at same height have different time",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			true,
		},
		{
			"valid time misbehaviour, both headers at different heights are at same time",
			&ibctmattestor.Misbehaviour{
				Header1: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight+5, heightMinus1, suite.chainA.LastHeader.Header.Time, suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers),
				Header2: header,
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			true,
		},
		{
			"misbehaviour Header1 is nil",
			ibctmattestor.NewMisbehaviour(nil, header),
			func(m *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"misbehaviour Header2 is nil",
			ibctmattestor.NewMisbehaviour(header, nil),
			func(m *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"valid misbehaviour with different trusted headers",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, clienttypes.NewHeight(0, uint64(revisionHeight)-3), suite.chainA.LastHeader.Header.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.NextVals, bothValSet, suite.chainA.Signers),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			true,
		},
		{
			"trusted height is 0 in Header1",
			&ibctmattestor.Misbehaviour{
				Header1: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, clienttypes.ZeroHeight(), suite.chainA.LastHeader.Header.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers),
				Header2: header,
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"trusted height is 0 in Header2",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, clienttypes.ZeroHeight(), suite.chainA.LastHeader.Header.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"trusted valset is nil in Header1",
			&ibctmattestor.Misbehaviour{
				Header1: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.NextVals, nil, suite.chainA.Signers),
				Header2: header,
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"trusted valset is nil in Header2",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.NextVals, nil, suite.chainA.Signers),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"chainIDs do not match",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader("ethermint", revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"header2 height is greater",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, 6, clienttypes.NewHeight(0, uint64(ibctesting.Height.RevisionHeight)+1), suite.chainA.LastHeader.Header.Time, suite.chainA.Vals, suite.chainA.NextVals, suite.chainA.Vals, suite.chainA.Signers),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error { return nil },
			false,
		},
		{
			"header 1 doesn't have 2/3 majority",
			&ibctmattestor.Misbehaviour{
				Header1: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time, bothValSet, bothValSet, suite.chainA.Vals, bothSigners),
				Header2: header,
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error {
				// voteSet contains only altVal which is less than 2/3 of total power (height/1height)
				wrongVoteSet := tmtypes.NewVoteSet(suite.chainA.ChainID, int64(misbehaviour.Header1.GetHeight().GetRevisionHeight()), 1, tmproto.PrecommitType, altValSet)
				blockID, err := tmtypes.BlockIDFromProto(&misbehaviour.Header1.Commit.BlockID)
				if err != nil {
					return err
				}

				extCommit, err := tmtypes.MakeExtCommit(*blockID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), misbehaviour.Header1.Commit.Round, wrongVoteSet, altSignerArr, suite.chainA.LastHeader.Header.Time, false)
				misbehaviour.Header1.Commit = extCommit.ToCommit().ToProto()
				return err
			},
			false,
		},
		{
			"header 2 doesn't have 2/3 majority",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time, bothValSet, bothValSet, suite.chainA.Vals, bothSigners),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error {
				// voteSet contains only altVal which is less than 2/3 of total power (height/1height)
				wrongVoteSet := tmtypes.NewVoteSet(suite.chainA.ChainID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), 1, tmproto.PrecommitType, altValSet)
				blockID, err := tmtypes.BlockIDFromProto(&misbehaviour.Header2.Commit.BlockID)
				if err != nil {
					return err
				}

				extCommit, err := tmtypes.MakeExtCommit(*blockID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), misbehaviour.Header2.Commit.Round, wrongVoteSet, altSignerArr, suite.chainA.LastHeader.Header.Time, false)
				misbehaviour.Header2.Commit = extCommit.ToCommit().ToProto()
				return err
			},
			false,
		},
		{
			"validators sign off on wrong commit",
			&ibctmattestor.Misbehaviour{
				Header1: header,
				Header2: suite.chainA.CreateTMAttestorClientHeader(suite.chainA.ChainID, revisionHeight, heightMinus1, suite.chainA.LastHeader.Header.Time, bothValSet, bothValSet, suite.chainA.Vals, bothSigners),
			},
			func(misbehaviour *ibctmattestor.Misbehaviour) error {
				tmBlockID := ibctesting.MakeBlockID(tmhash.Sum([]byte("other_hash")), 3, tmhash.Sum([]byte("other_partset")))
				misbehaviour.Header2.Commit.BlockID = tmBlockID.ToProto()
				return nil
			},
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.malleateMisbehaviour(tc.misbehaviour)
			suite.Require().NoError(err)

			if tc.expPass {
				suite.Require().NoError(tc.misbehaviour.ValidateBasic(), "valid test case %d failed: %s", i, tc.name)
			} else {
				suite.Require().Error(tc.misbehaviour.ValidateBasic(), "invalid test case %d passed: %s", i, tc.name)
			}
		})
	}
}

func getBothSigners(suite *TMAttestorTestSuite, altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) (*tmtypes.ValidatorSet, map[string]tmtypes.PrivValidator) {
	// Create bothValSet with both suite validator and altVal. Would be valid update
	bothValSet := tmtypes.NewValidatorSet(append(suite.chainA.Vals.Validators, altVal))

	bothSigners := map[string]tmtypes.PrivValidator{
		altVal.Address.String(): altPrivVal,
	}

	for k, v := range suite.chainA.Signers {
		bothSigners[k] = v
	}
	return bothValSet, bothSigners
}
