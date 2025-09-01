package tendermintattestor_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	ics23 "github.com/cosmos/ics23/go"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"

	"testing"

	"github.com/stretchr/testify/require"
)

const (
	// Do not change the length of these variables
	fiftyCharChainID    = "12345678901234567890123456789012345678901234567890"
	fiftyOneCharChainID = "123456789012345678901234567890123456789012345678901"
)

var invalidProof = []byte("invalid proof")

func (suite *TMAttestorTestSuite) TestStatus() {
	var (
		path        *ibctesting.Path
		clientState *ibctmattestor.ClientState
	)

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{"client is active", func() {}, exported.Active},
		{"client is frozen", func() {
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, exported.Frozen},
		{"client status without consensus state", func() {
			clientState.LatestHeight = clientState.LatestHeight.Increment().(clienttypes.Height)
			path.EndpointA.SetClientState(clientState)
		}, exported.Expired},
		{"client status is expired", func() {
			suite.coordinator.IncrementTimeBy(clientState.TrustingPeriod)
		}, exported.Expired},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPathWithTendermintAttestors(suite.chainA, suite.chainB, 0, 0)
			suite.coordinator.SetupClients(path)

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)
			clientState = path.EndpointA.GetClientState().(*ibctmattestor.ClientState)

			tc.malleate()

			status := clientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.App.AppCodec())
			suite.Require().Equal(tc.expStatus, status)
		})

	}
}

func (suite *TMAttestorTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *ibctmattestor.ClientState
		expPass     bool
	}{
		{
			name:        "valid client",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     true,
		},
		{
			name:        "valid client with nil upgrade path",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), nil, []*codectypes.Any{}, 0),
			expPass:     true,
		},
		{
			name:        "invalid chainID",
			clientState: ibctmattestor.NewClientState("  ", ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "valid chainID - chainID validation failed for chainID of length 50! ",
			clientState: ibctmattestor.NewClientState(fiftyCharChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     true,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "invalid chainID - chainID validation did not fail for chainID of length 51! ",
			clientState: ibctmattestor.NewClientState(fiftyOneCharChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid trust level",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctm.Fraction{Numerator: 0, Denominator: 1}, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid zero trusting period",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, 0, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid negative trusting period",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, -1, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid zero unbonding period",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, 0, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid negative unbonding period",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, -1, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid zero max clock drift",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, 0, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid negative max clock drift",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, -1, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid revision number",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "invalid revision height",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clienttypes.ZeroHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "trusting period not less than unbonding period",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.UnbondingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "proof specs is nil",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.UnbondingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, nil, ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
		{
			name:        "proof specs contains nil",
			clientState: ibctmattestor.NewClientState(ibctesting.ChainID, ibctesting.DefaultTrustLevel, ibctesting.UnbondingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, ibctesting.Height, []*ics23.ProofSpec{ics23.TendermintSpec, nil}, ibctesting.UpgradePath, []*codectypes.Any{}, 0),
			expPass:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *TMAttestorTestSuite) TestInitialize() {
	testCases := []struct {
		name           string
		consensusState exported.ConsensusState
		expPass        bool
	}{
		{
			name:           "valid consensus",
			consensusState: &ibctmattestor.ConsensusState{},
			expPass:        true,
		},
		{
			name:           "invalid consensus: consensus state is solomachine consensus",
			consensusState: ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2).ConsensusState(),
			expPass:        false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPathWithTendermintAttestors(suite.chainA, suite.chainB, 0, 0)

			tmConfig, ok := path.EndpointB.ClientConfig.(*ibctesting.TendermintAttestorConfig)
			suite.Require().True(ok)

			clientState := ibctmattestor.NewClientState(
				path.EndpointB.Chain.ChainID,
				tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
				suite.chainB.LastHeader.GetTrustedHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath,
				[]*codectypes.Any{}, tmConfig.Threshold,
			)

			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)
			err := clientState.Initialize(suite.chainA.GetContext(), suite.chainA.Codec, store, tc.consensusState)

			if tc.expPass {
				suite.Require().NoError(err, "valid case returned an error")
				suite.Require().True(store.Has(host.ClientStateKey()))
				suite.Require().True(store.Has(host.ConsensusStateKey(suite.chainB.LastHeader.GetTrustedHeight())))
			} else {
				suite.Require().Error(err, "invalid case didn't return an error")
				suite.Require().False(store.Has(host.ClientStateKey()))
				suite.Require().False(store.Has(host.ConsensusStateKey(suite.chainB.LastHeader.GetTrustedHeight())))
			}
		})
	}
}

func (suite *TMAttestorTestSuite) TestVerifyMembership() {
	var (
		testingpath      *ibctesting.Path
		delayTimePeriod  uint64
		delayBlockPeriod uint64
		err              error
		proofHeight      exported.Height
		proof            []byte
		path             exported.Path
		value            []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful ClientState verification",
			func() {
				// default proof construction uses ClientState
			},
			true,
		},
		{
			"successful ConsensusState verification", func() {
				key := host.FullConsensusStateKey(testingpath.EndpointB.ClientID, testingpath.EndpointB.GetClientState().GetLatestHeight())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)

				consensusState := testingpath.EndpointB.GetConsensusState(testingpath.EndpointB.GetClientState().GetLatestHeight()).(*ibctmattestor.ConsensusState)
				value, err = suite.chainB.Codec.MarshalInterface(consensusState)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Connection verification", func() {
				key := host.ConnectionKey(testingpath.EndpointB.ConnectionID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)

				connection := testingpath.EndpointB.GetConnection()
				value, err = suite.chainB.Codec.Marshal(&connection)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Channel verification", func() {
				key := host.ChannelKey(testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)

				channel := testingpath.EndpointB.GetChannel()
				value, err = suite.chainB.Codec.Marshal(&channel)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful PacketCommitment verification", func() {
				// send from chainB to chainA since we are proving chainB sent a packet
				res, err := testingpath.EndpointB.Chain.SendMsgs(&transfertypes.MsgTransfer{
					SourcePort:    testingpath.EndpointB.ChannelConfig.PortID,
					SourceChannel: testingpath.EndpointB.ChannelID,
					Token:         sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
					Sender:        testingpath.EndpointB.Chain.SenderAccount.GetAddress().String(),
					Receiver:      testingpath.EndpointA.Chain.SenderAccount.GetAddress().String(),
					TimeoutHeight: clienttypes.NewHeight(1, 100),
				})
				suite.Require().NoError(err)
				packet, err := ibctesting.ParsePacketFromEvents(res.Events)
				suite.Require().NoError(err)

				key := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				err = testingpath.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)

				value = channeltypes.CommitPacket(suite.chainA.App.GetIBCKeeper().Codec(), packet)
			}, true,
		},
		{
			"successful Acknowledgement verification", func() {
				// send from chainA to chainB since we are proving chainB wrote an acknowledgement
				res, err := testingpath.EndpointA.Chain.SendMsgs(&transfertypes.MsgTransfer{
					SourcePort:    testingpath.EndpointA.ChannelConfig.PortID,
					SourceChannel: testingpath.EndpointA.ChannelID,
					Token:         sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
					Sender:        testingpath.EndpointA.Chain.SenderAccount.GetAddress().String(),
					Receiver:      testingpath.EndpointB.Chain.SenderAccount.GetAddress().String(),
					TimeoutHeight: clienttypes.NewHeight(1, 100),
				})
				suite.Require().NoError(err)
				packet, err := ibctesting.ParsePacketFromEvents(res.Events)
				suite.Require().NoError(err)

				err = testingpath.EndpointB.UpdateClient()
				suite.Require().NoError(err)

				res, err = testingpath.EndpointB.RecvPacketWithResult(packet)
				suite.Require().NoError(err)

				key := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				err = testingpath.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)

				value = channeltypes.CommitAcknowledgement(ibctesting.TransferSuccessAcknowledgement.Acknowledgement())
			},
			true,
		},
		{
			"successful verification outside IBC store", func() {
				key := transfertypes.PortKey
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(commitmenttypes.NewMerklePrefix([]byte(transfertypes.StoreKey)), merklePath)
				suite.Require().NoError(err)

				clientState := testingpath.EndpointA.GetClientState()
				proof, proofHeight = suite.chainB.QueryProofForStore(transfertypes.StoreKey, key, int64(clientState.GetLatestHeight().GetRevisionHeight()))
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)

				value = []byte(suite.chainB.App.GetTransferKeeper().GetPort(suite.chainB.GetContext()))
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			true,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			false,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			true,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			false,
		},
		{
			"latest client height < height", func() {
				proofHeight = testingpath.EndpointA.GetClientState().GetLatestHeight().Increment()
			}, false,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			false,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = invalidProof
			}, false,
		},
		{
			"consensus state not found", func() {
				proofHeight = clienttypes.ZeroHeight()
			}, false,
		},
		{
			"proof verification failed", func() {
				// change the value being proved
				value = []byte("invalid value")
			}, false,
		},
		{
			"proof is empty", func() {
				// change the inserted proof
				proof = []byte{}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			testingpath = ibctesting.NewPathWithTendermintAttestors(suite.chainA, suite.chainB, 0, 0)
			suite.coordinator.Setup(testingpath)

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0

			// create default proof, merklePath, and value which passes
			// may be overwritten by malleate()
			key := host.FullClientStateKey(testingpath.EndpointB.ClientID)
			merklePath := commitmenttypes.NewMerklePath(string(key))
			path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
			suite.Require().NoError(err)

			proof, proofHeight = suite.chainB.QueryProof(key)
			proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
			suite.Require().NoError(err)

			clientState := testingpath.EndpointB.GetClientState().(*ibctmattestor.ClientState)
			value, err = suite.chainB.Codec.MarshalInterface(clientState)
			suite.Require().NoError(err)

			tc.malleate() // make changes as necessary

			clientState = testingpath.EndpointA.GetClientState().(*ibctmattestor.ClientState)

			ctx := suite.chainA.GetContext()
			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, testingpath.EndpointA.ClientID)

			err = clientState.VerifyMembership(
				ctx, store, suite.chainA.Codec, proofHeight, delayTimePeriod, delayBlockPeriod,
				proof, path, value,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TMAttestorTestSuite) TestVerifyNonMembership() {
	var (
		testingpath         *ibctesting.Path
		delayTimePeriod     uint64
		delayBlockPeriod    uint64
		err                 error
		proofHeight         exported.Height
		path                exported.Path
		proof               []byte
		invalidClientID     = "09-tendermint"
		invalidConnectionID = "connection-100"
		invalidChannelID    = "channel-800"
		invalidPortID       = "invalid-port"
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful ClientState verification of non membership",
			func() {
				// default proof construction uses ClientState
			},
			true,
		},
		{
			"successful ConsensusState verification of non membership", func() {
				key := host.FullConsensusStateKey(invalidClientID, testingpath.EndpointB.GetClientState().GetLatestHeight())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Connection verification of non membership", func() {
				key := host.ConnectionKey(invalidConnectionID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Channel verification of non membership", func() {
				key := host.ChannelKey(testingpath.EndpointB.ChannelConfig.PortID, invalidChannelID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful PacketCommitment verification of non membership", func() {
				// make packet commitment proof
				key := host.PacketCommitmentKey(invalidPortID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)
			}, true,
		},
		{
			"successful Acknowledgement verification of non membership", func() {
				key := host.PacketAcknowledgementKey(invalidPortID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful NextSequenceRecv verification of non membership", func() {
				key := host.NextSequenceRecvKey(invalidPortID, invalidChannelID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful verification of non membership outside IBC store", func() {
				key := []byte{0x08}
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(commitmenttypes.NewMerklePrefix([]byte(transfertypes.StoreKey)), merklePath)
				suite.Require().NoError(err)

				clientState := testingpath.EndpointA.GetClientState()
				proof, proofHeight = suite.chainB.QueryProofForStore(transfertypes.StoreKey, key, int64(clientState.GetLatestHeight().GetRevisionHeight()))
				proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			true,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			false,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			true,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			false,
		},
		{
			"latest client height < height", func() {
				proofHeight = testingpath.EndpointA.GetClientState().GetLatestHeight().Increment()
			}, false,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			false,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = invalidProof
			}, false,
		},
		{
			"consensus state not found", func() {
				proofHeight = clienttypes.ZeroHeight()
			}, false,
		},
		{
			"verify non membership fails as path exists", func() {
				// change the value being proved
				key := host.FullClientStateKey(testingpath.EndpointB.ClientID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
			}, false,
		},
		{
			"proof is empty", func() {
				// change the inserted proof
				proof = []byte{}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			testingpath = ibctesting.NewPathWithTendermintAttestors(suite.chainA, suite.chainB, 0, 0)
			suite.coordinator.Setup(testingpath)

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0

			// create default proof, merklePath, and value which passes
			// may be overwritten by malleate()
			key := host.FullClientStateKey("invalid-client-id")

			merklePath := commitmenttypes.NewMerklePath(string(key))
			path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
			suite.Require().NoError(err)

			proof, proofHeight = suite.chainB.QueryProof(key)
			proof, err = testingpath.EndpointA.GetProofWithAttestations(proof)
			suite.Require().NoError(err)

			tc.malleate() // make changes as necessary

			clientState := testingpath.EndpointA.GetClientState().(*ibctmattestor.ClientState)

			ctx := suite.chainA.GetContext()
			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, testingpath.EndpointA.ClientID)

			err = clientState.VerifyNonMembership(
				ctx, store, suite.chainA.Codec, proofHeight, delayTimePeriod, delayBlockPeriod,
				proof, path,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func Test_hasSameAttestorsAndThreshold(t *testing.T) {
	// Generate test public keys
	pubkey1 := secp256k1.GenPrivKey().PubKey()
	pubkey2 := secp256k1.GenPrivKey().PubKey()
	pubkey3 := secp256k1.GenPrivKey().PubKey()
	pubkey4 := secp256k1.GenPrivKey().PubKey()

	// Helper function to create attestor pubkeys
	createAttestorPubkeys := func(pubkeys ...cryptotypes.PubKey) []*codectypes.Any {
		result := make([]*codectypes.Any, len(pubkeys))
		for i, pk := range pubkeys {
			any, err := codectypes.NewAnyWithValue(pk)
			require.NoError(t, err)
			result[i] = any
		}
		return result
	}

	// Helper function to create client state
	createClientState := func(attestorPubkeys []*codectypes.Any, threshold uint32) ibctmattestor.ClientState {
		return ibctmattestor.ClientState{
			ClientState: &ibctm.ClientState{
				ChainId:         "test-chain",
				TrustLevel:      ibctm.Fraction{Numerator: 1, Denominator: 3},
				TrustingPeriod:  86400,
				UnbondingPeriod: 172800,
				MaxClockDrift:   1000,
				LatestHeight:    clienttypes.Height{RevisionNumber: 1, RevisionHeight: 100},
				ProofSpecs:      commitmenttypes.GetSDKSpecs(),
				UpgradePath:     []string{"upgrade", "upgradedClient"},
			},
			AttestorPubkeys: attestorPubkeys,
			Threshold:       threshold,
		}
	}

	tests := []struct {
		name     string
		cs1      ibctmattestor.ClientState
		cs2      ibctmattestor.ClientState
		expected bool
	}{
		{
			name: "same attestors and threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			expected: true,
		},
		{
			name: "same attestors in different order and threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey3, pubkey1, pubkey2),
				2,
			),
			expected: true,
		},
		{
			name: "different threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				3,
			),
			expected: false,
		},
		{
			name: "different number of attestors",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2),
				2,
			),
			expected: false,
		},
		{
			name: "different attestor sets",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey4),
				2,
			),
			expected: false,
		},
		{
			name: "duplicate attestors in first set",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey1, pubkey2),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			expected: false,
		},
		{
			name: "duplicate attestors in second set",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey2),
				2,
			),
			expected: false,
		},
		{
			name: "empty attestor sets with same threshold",
			cs1: createClientState(
				[]*codectypes.Any{},
				0,
			),
			cs2: createClientState(
				[]*codectypes.Any{},
				0,
			),
			expected: true,
		},
		{
			name: "empty attestor sets with different threshold",
			cs1: createClientState(
				[]*codectypes.Any{},
				0,
			),
			cs2: createClientState(
				[]*codectypes.Any{},
				1,
			),
			expected: false,
		},
		{
			name: "single attestor with same threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				1,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				1,
			),
			expected: true,
		},
		{
			name: "single attestor with different threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				1,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				2,
			),
			expected: false,
		},
		{
			name: "same attestors but different counts (duplicates)",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3, pubkey3),
				2,
			),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cs1.HasSameAttestorsAndThreshold(tt.cs2)
			require.Equal(t, tt.expected, result, "test case: %s", tt.name)
		})
	}
}

// Test_hasSameAttestorsAndThreshold_EdgeCases tests edge cases and boundary conditions
func Test_hasSameAttestorsAndThreshold_EdgeCases(t *testing.T) {
	// Generate test public keys
	pubkey1 := secp256k1.GenPrivKey().PubKey()
	pubkey2 := secp256k1.GenPrivKey().PubKey()

	// Helper function to create attestor pubkeys
	createAttestorPubkeys := func(pubkeys ...cryptotypes.PubKey) []*codectypes.Any {
		result := make([]*codectypes.Any, len(pubkeys))
		for i, pk := range pubkeys {
			any, err := codectypes.NewAnyWithValue(pk)
			require.NoError(t, err)
			result[i] = any
		}
		return result
	}

	// Helper function to create client state
	createClientState := func(attestorPubkeys []*codectypes.Any, threshold uint32) ibctmattestor.ClientState {
		return ibctmattestor.ClientState{
			ClientState: &ibctm.ClientState{
				ChainId:         "test-chain",
				TrustLevel:      ibctm.Fraction{Numerator: 1, Denominator: 3},
				TrustingPeriod:  86400,
				UnbondingPeriod: 172800,
				MaxClockDrift:   1000,
				LatestHeight:    clienttypes.Height{RevisionNumber: 1, RevisionHeight: 100},
				ProofSpecs:      commitmenttypes.GetSDKSpecs(),
				UpgradePath:     []string{"upgrade", "upgradedClient"},
			},
			AttestorPubkeys: attestorPubkeys,
			Threshold:       threshold,
		}
	}

	tests := []struct {
		name     string
		cs1      ibctmattestor.ClientState
		cs2      ibctmattestor.ClientState
		expected bool
	}{
		{
			name: "zero threshold with empty attestors",
			cs1: createClientState(
				[]*codectypes.Any{},
				0,
			),
			cs2: createClientState(
				[]*codectypes.Any{},
				0,
			),
			expected: true,
		},
		{
			name: "zero threshold with non-empty attestors",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				0,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				0,
			),
			expected: true,
		},
		{
			name: "threshold equals attestor count",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2),
				2,
			),
			expected: true,
		},
		{
			name: "threshold greater than attestor count",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				2,
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cs1.HasSameAttestorsAndThreshold(tt.cs2)
			require.Equal(t, tt.expected, result, "test case: %s", tt.name)
		})
	}
}
