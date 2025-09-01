package tendermintattestor_test

import (
	collections "cosmossdk.io/collections"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"

	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v8/testing/mock"
	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"
)

func (suite *TMAttestorTestSuite) TestGetSelfConsensusState() {
	var height clienttypes.Height

	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "zero height",
			malleate: func() {},
			expError: clienttypes.ErrInvalidHeight,
		},
		{
			name: "height > latest height",
			malleate: func() {
				height = clienttypes.NewHeight(1, uint64(suite.chainA.GetContext().BlockHeight())+1)
			},
			expError: collections.ErrNotFound,
		},
		{
			name: "pruned historical info",
			malleate: func() {
				height = clienttypes.NewHeight(1, uint64(suite.chainA.GetContext().BlockHeight())-1)

				err := suite.chainA.App.GetStakingKeeper().DeleteHistoricalInfo(suite.chainA.GetContext(), int64(height.GetRevisionHeight()))
				suite.Require().NoError(err)
			},
			expError: collections.ErrNotFound,
		},
		{
			name: "custom consensus host: failure",
			malleate: func() {
				consensusHost := &mock.ConsensusHost{
					GetSelfConsensusStateFn: func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
						return nil, mock.MockApplicationCallbackError
					},
				}
				suite.chainA.App.GetIBCKeeper().SetConsensusHost(consensusHost)
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "custom consensus host: success",
			malleate: func() {
				consensusHost := &mock.ConsensusHost{
					GetSelfConsensusStateFn: func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
						return &solomachine.ConsensusState{}, nil
					},
				}
				suite.chainA.App.GetIBCKeeper().SetConsensusHost(consensusHost)
			},
			expError: nil,
		},
		{
			name: "latest height - 1",
			malleate: func() {
				height = clienttypes.NewHeight(1, uint64(suite.chainA.GetContext().BlockHeight())-1)
			},
			expError: nil,
		},
		{
			name: "latest height",
			malleate: func() {
				// historical info is set on BeginBlock in x/staking, which is now encapsulated within the FinalizeBlock abci method,
				// thus, we do not have historical info for current height due to how the ibctesting library operates.
				// ibctesting calls app.Commit() as a final step on NextBlock and we invoke test code before FinalizeBlock is called at the current height once again.
				err := suite.chainA.App.GetStakingKeeper().TrackHistoricalInfo(suite.chainA.GetContext())
				suite.Require().NoError(err)

				height = clienttypes.GetSelfHeight(suite.chainA.GetContext())
			},
			expError: nil,
		},
	}

	for i, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			height = clienttypes.ZeroHeight()

			tc.malleate()

			cs, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetSelfConsensusState(suite.chainA.GetContext(), height)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err, "Case %d should have passed: %s", i, tc.name)
				suite.Require().NotNil(cs, "Case %d should have passed: %s", i, tc.name)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "Case %d should have failed: %s", i, tc.name)
				suite.Require().Nil(cs, "Case %d should have failed: %s", i, tc.name)
			}
		})
	}
}

func (suite *TMAttestorTestSuite) TestValidateSelfClient() {
	testClientHeight := clienttypes.GetSelfHeight(suite.chainA.GetContext())
	testClientHeight.RevisionHeight--

	var clientState exported.ClientState

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name: "success",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: nil,
		},
		{
			name: "success with nil UpgradePath",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), nil, []*codectypes.Any{}, 0)
			},
			expError: nil,
		},
		{
			name: "success with custom self validator: solomachine",
			malleate: func() {
				clientState = solomachine.NewClientState(1, &solomachine.ConsensusState{})

				smConsensusHost := &mock.ConsensusHost{
					ValidateSelfClientFn: func(ctx sdk.Context, clientState exported.ClientState) error {
						smClientState, ok := clientState.(*solomachine.ClientState)
						suite.Require().True(ok)
						suite.Require().Equal(uint64(1), smClientState.Sequence)

						return nil
					},
				}

				// add mock validation logic
				suite.chainA.App.GetIBCKeeper().SetConsensusHost(smConsensusHost)
			},
			expError: nil,
		},
		{
			name: "frozen client",
			malleate: func() {
				clientState = &ibctmattestor.ClientState{
					ClientState: &ibctm.ClientState{
						ChainId: suite.chainA.ChainID, TrustLevel: ibctesting.DefaultTrustLevel, TrustingPeriod: ibctesting.TrustingPeriod, UnbondingPeriod: ibctesting.UnbondingPeriod, MaxClockDrift: ibctesting.MaxClockDrift, FrozenHeight: testClientHeight, LatestHeight: testClientHeight, ProofSpecs: commitmenttypes.GetSDKSpecs(), UpgradePath: ibctesting.UpgradePath,
					},
					AttestorPubkeys: []*codectypes.Any{},
					Threshold:       0,
				}
			},
			expError: clienttypes.ErrClientFrozen,
		},
		{
			name: "incorrect chainID",
			malleate: func() {
				clientState = ibctmattestor.NewClientState("gaiatestnet", ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid client height",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clienttypes.GetSelfHeight(suite.chainA.GetContext()).Increment().(clienttypes.Height), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid client type",
			malleate: func() {
				clientState = solomachine.NewClientState(0, &solomachine.ConsensusState{})
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid client revision",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clienttypes.NewHeight(1, 5), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid proof specs",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, testClientHeight, nil, ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid trust level",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctm.Fraction{Numerator: 0, Denominator: 1}, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid unbonding period",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod+10, ibctesting.MaxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid trusting period",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.UnbondingPeriod+10, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "invalid upgrade path",
			malleate: func() {
				clientState = ibctmattestor.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), []string{"bad", "upgrade", "path"}, []*codectypes.Any{}, 0)
			},
			expError: clienttypes.ErrInvalidClient,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			err := suite.chainA.App.GetIBCKeeper().ClientKeeper.ValidateSelfClient(suite.chainA.GetContext(), clientState)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err, "expected valid client for case: %s", tc.name)
			} else {
				suite.Require().Error(err, "expected invalid client for case: %s", tc.name)
			}
		})
	}
}
