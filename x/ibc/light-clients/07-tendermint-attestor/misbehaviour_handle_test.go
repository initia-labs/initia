package tendermintattestor_test

import (
	"fmt"
	"strings"
	"time"

	tmtypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"

	// ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctestingmock "github.com/cosmos/ibc-go/v8/testing/mock"
	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"
)

func (suite *TMAttestorTestSuite) TestVerifyMisbehaviour() {
	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	// create modified heights to use for test-cases
	altVal := tmtypes.NewValidator(altPubKey, 100)

	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
	altSigners := ibctesting.GetAltSigners(altVal, altPrivVal)

	var (
		path         *ibctesting.Path
		misbehaviour exported.ClientMessage
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid fork misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour, header 1 time stricly less than header 2 time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Hour), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, true,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals1, found := suite.chainB.GetValsAtHeight(int64(trustedHeight1.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				trustedHeight2 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals2, found := suite.chainB.GetValsAtHeight(int64(trustedHeight2.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight1, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals1, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight2, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals2, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehaviour at a previous revision", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}

				// increment revision number
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"valid misbehaviour at a future revision", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				futureRevision := fmt.Sprintf("%s-%d", strings.TrimSuffix(suite.chainB.ChainID, fmt.Sprintf("-%d", clienttypes.ParseChainID(suite.chainB.ChainID))), height.GetRevisionNumber()+1)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehaviour with trusted heights at a previous revision", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				// increment revision of chainID
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altValSet.Proposer))
				// Create bothValSet with both suite validator and altVal
				bothSigners := make(map[string]tmtypes.PrivValidator, len(suite.chainB.Signers)+1)
				for k, v := range suite.chainB.Signers {
					bothSigners[k] = v
				}
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
				}
			}, true,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight.Increment().(clienttypes.Height), suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"invalid tendermint misbehaviour", func() {
				misbehaviour = &solomachine.Misbehaviour{}
			}, false,
		},
		{
			"trusting period expired", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				suite.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintAttestorConfig).TrustingPeriod)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPathWithTendermintAttestors(suite.chainA, suite.chainB, 0, 0)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, misbehaviour)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// test both fork and time misbehaviour for chainIDs not in the revision format
// this function is separate as it must use a global variable in the testing package
// to initialize chains not in the revision format
func (suite *TMAttestorTestSuite) TestVerifyMisbehaviourNonRevisionChainID() {
	// NOTE: chains set to non revision format
	prevSuffix := ibctesting.ChainIDSuffix
	ibctesting.ChainIDSuffix = ""
	defer func() { ibctesting.ChainIDSuffix = prevSuffix }()

	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	// create modified heights to use for test-cases
	altVal := tmtypes.NewValidator(altPubKey, 100)

	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
	altSigners := ibctesting.GetAltSigners(altVal, altPrivVal)

	var (
		path         *ibctesting.Path
		misbehaviour exported.ClientMessage
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid fork misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour, header 1 time stricly less than header 2 time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Hour), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, true,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals1, found := suite.chainB.GetValsAtHeight(int64(trustedHeight1.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				trustedHeight2 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals2, found := suite.chainB.GetValsAtHeight(int64(trustedHeight2.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight1, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals1, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight2, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals2, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := make(map[string]tmtypes.PrivValidator, len(suite.chainB.Signers)+1)
				for k, v := range suite.chainB.Signers {
					bothSigners[k] = v
				}
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
				}
			}, true,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight.Increment().(clienttypes.Height), suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"invalid tendermint misbehaviour", func() {
				misbehaviour = &solomachine.Misbehaviour{}
			}, false,
		},
		{
			"trusting period expired", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				suite.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintAttestorConfig).TrustingPeriod)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctmattestor.Misbehaviour{
					Header1: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMAttestorClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPathWithTendermintAttestors(suite.chainA, suite.chainB, 0, 0)

			path.EndpointA.CreateClient()

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, misbehaviour)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
