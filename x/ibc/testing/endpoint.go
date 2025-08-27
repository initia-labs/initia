package ibctesting

import (
	"fmt"
	"strings"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctmattestor "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint-attestor"
)

// Endpoint is a which represents a channel endpoint and its associated
// client and connections. It contains client, connection, and channel
// configuration parameters. Endpoint functions will utilize the parameters
// set in the configuration structs when executing IBC messages.
type Endpoint struct {
	Chain        *TestChain
	Counterparty *Endpoint
	ClientID     string
	ConnectionID string
	ChannelID    string

	ClientConfig     ClientConfig
	ConnectionConfig *ConnectionConfig
	ChannelConfig    *ChannelConfig
}

// NewEndpoint constructs a new endpoint without the counterparty.
// CONTRACT: the counterparty endpoint must be set by the caller.
func NewEndpoint(
	chain *TestChain, clientConfig ClientConfig,
	connectionConfig *ConnectionConfig, channelConfig *ChannelConfig,
) *Endpoint {
	return &Endpoint{
		Chain:            chain,
		ClientConfig:     clientConfig,
		ConnectionConfig: connectionConfig,
		ChannelConfig:    channelConfig,
	}
}

// NewDefaultEndpoint constructs a new endpoint using default values.
// CONTRACT: the counterparty endpoitn must be set by the caller.
func NewDefaultEndpoint(chain *TestChain) *Endpoint {
	return &Endpoint{
		Chain:            chain,
		ClientConfig:     NewTendermintConfig(),
		ConnectionConfig: NewConnectionConfig(),
		ChannelConfig:    NewChannelConfig(),
	}
}

func NewEndpointWithTendermintAttestor(chain *TestChain, numAttestors, threshold int) *Endpoint {
	return &Endpoint{
		Chain:            chain,
		ClientConfig:     NewTendermintAttestorConfig(numAttestors, threshold),
		ConnectionConfig: NewConnectionConfig(),
		ChannelConfig:    NewChannelConfig(),
	}
}

// QueryProof queries proof associated with this endpoint using the latest client state
// height on the counterparty chain.
func (endpoint *Endpoint) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	// obtain the counterparty client representing the chain associated with the endpoint
	clientState := endpoint.Counterparty.Chain.GetClientState(endpoint.Counterparty.ClientID)

	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.QueryProofAtHeight(key, clientState.GetLatestHeight().GetRevisionHeight())
}

// QueryProofAtHeight queries proof associated with this endpoint using the proof height
// provided
func (endpoint *Endpoint) QueryProofAtHeight(key []byte, height uint64) ([]byte, clienttypes.Height) {
	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.Chain.QueryProofAtHeight(key, int64(height))
}

// CreateClient creates an IBC client on the endpoint. It will update the
// clientID for the endpoint if the message is successfully executed.
// NOTE: a solo machine client will be created with an empty diversifier.
func (endpoint *Endpoint) CreateClient() {
	// ensure counterparty has committed state
	endpoint.Counterparty.Chain.NextBlock()

	var (
		clientState    exported.ClientState
		consensusState exported.ConsensusState
	)

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		tmConfig, ok := endpoint.ClientConfig.(*TendermintConfig)
		require.True(endpoint.Chain.T, ok)

		height := endpoint.Counterparty.Chain.LastHeader.GetHeight().(clienttypes.Height)
		clientState = ibctm.NewClientState(
			endpoint.Counterparty.Chain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), UpgradePath)
		consensusState = endpoint.Counterparty.Chain.LastHeader.ConsensusState()
	case exported.TendermintAttestor:
		tmAttestorConfig, ok := endpoint.ClientConfig.(*TendermintAttestorConfig)
		require.True(endpoint.Chain.T, ok)

		height := endpoint.Counterparty.Chain.LastHeader.GetHeight().(clienttypes.Height)

		attestors := make([]ibctmattestor.PubKey, 0, len(tmAttestorConfig.AttestorPrivkeys))
		for _, privKey := range tmAttestorConfig.AttestorPrivkeys {
			attestors = append(attestors, ibctmattestor.PubKey{
				Type: ibctmattestor.PubKeyType(ibctmattestor.PubKeyType_value[strings.ToUpper(privKey.Type())]),
				Key:  privKey.PubKey().Bytes(),
			})
		}

		clientState = ibctmattestor.NewClientState(
			endpoint.Counterparty.Chain.ChainID, tmAttestorConfig.TrustLevel, tmAttestorConfig.TrustingPeriod, tmAttestorConfig.UnbondingPeriod, tmAttestorConfig.MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), UpgradePath, attestors, tmAttestorConfig.Threshold)

		consensusState = ibctmattestor.FromTendermintConsensusState(endpoint.Counterparty.Chain.LastHeader.ConsensusState())
	case exported.Solomachine:
		// TODO
		//		solo := NewSolomachine(endpoint.Chain.T, endpoint.Chain.Codec, clientID, "", 1)
		//		clientState = solo.ClientState()
		//		consensusState = solo.ConsensusState()

	default:
		require.Fail(endpoint.Chain.T, "client type %s is not supported", endpoint.ClientConfig.GetClientType())
	}

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.T, err)

	res, err := endpoint.Chain.SendMsgs(msg)
	require.NoError(endpoint.Chain.T, err)

	endpoint.ClientID, err = ParseClientIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.T, err)

}

// UpdateClient updates the IBC client associated with the endpoint.
func (endpoint *Endpoint) UpdateClient() (err error) {
	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	var header exported.ClientMessage

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		header, err = endpoint.Chain.ConstructUpdateTMClientHeader(endpoint.Counterparty.Chain, endpoint.ClientID)
	case exported.TendermintAttestor:
		header, err = endpoint.Chain.ConstructUpdateTMAttestorClientHeader(endpoint.Counterparty.Chain, endpoint.ClientID)

	default:
		err = fmt.Errorf("client type %s is not supported", endpoint.ClientConfig.GetClientType())
	}

	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.T, err)

	return endpoint.Chain.sendMsgs(msg)
}

func (endpoint *Endpoint) UpdateClientWithClientID(clientID string) (err error) {
	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	var header exported.ClientMessage

	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}
	switch clientType {
	case exported.Tendermint:
		header, err = endpoint.Chain.ConstructUpdateTMClientHeader(endpoint.Counterparty.Chain, clientID)
	case exported.TendermintAttestor:
		header, err = endpoint.Chain.ConstructUpdateTMAttestorClientHeader(endpoint.Counterparty.Chain, clientID)

	default:
		err = fmt.Errorf("client type %s is not supported", clientType)
	}

	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		clientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.T, err)

	return endpoint.Chain.sendMsgs(msg)
}

// UpgradeChain will upgrade a chain's chainID to the next revision number.
// It will also update the counterparty client.
// TODO: implement actual upgrade chain functionality via scheduling an upgrade
// and upgrading the client via MsgUpgradeClient
// see reference https://github.com/cosmos/ibc-go/pull/1169
func (endpoint *Endpoint) UpgradeChain() error {
	if strings.TrimSpace(endpoint.Counterparty.ClientID) == "" {
		return fmt.Errorf("cannot upgrade chain if there is no counterparty client")
	}

	clientState := endpoint.Counterparty.GetClientState().(*ibctm.ClientState)

	// increment revision number in chainID

	oldChainID := clientState.ChainId
	if !clienttypes.IsRevisionFormat(oldChainID) {
		return fmt.Errorf("cannot upgrade chain which is not of revision format: %s", oldChainID)
	}

	revisionNumber := clienttypes.ParseChainID(oldChainID)
	newChainID, err := clienttypes.SetRevisionNumber(oldChainID, revisionNumber+1)
	if err != nil {
		return err
	}

	// update chain
	baseapp.SetChainID(newChainID)(endpoint.Chain.GetInitiaApp().GetBaseApp())
	endpoint.Chain.ChainID = newChainID
	endpoint.Chain.CurrentHeader.ChainID = newChainID
	endpoint.Chain.NextBlock() // commit changes

	// update counterparty client manually
	clientState.ChainId = newChainID
	clientState.LatestHeight = clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1)
	endpoint.Counterparty.SetClientState(clientState)

	consensusState := &ibctm.ConsensusState{
		Timestamp:          endpoint.Chain.LastHeader.GetTime(),
		Root:               commitmenttypes.NewMerkleRoot(endpoint.Chain.LastHeader.Header.GetAppHash()),
		NextValidatorsHash: endpoint.Chain.LastHeader.Header.NextValidatorsHash,
	}
	endpoint.Counterparty.SetConsensusState(consensusState, clientState.GetLatestHeight())

	// ensure the next update isn't identical to the one set in state
	endpoint.Chain.Coordinator.IncrementTime()
	endpoint.Chain.NextBlock()

	return endpoint.Counterparty.UpdateClient()
}

func (endpoint *Endpoint) GetProofWithAttestations(proof []byte) ([]byte, error) {
	if endpoint.ClientConfig.GetClientType() != exported.TendermintAttestor {
		return proof, nil
	}

	proofWithAttestations := ibctmattestor.MerkleProofBytesWithAttestations{
		ProofBytes:   proof,
		Attestations: make([]*ibctmattestor.Attestation, 0, len(endpoint.ClientConfig.(*TendermintAttestorConfig).AttestorPrivkeys)),
	}
	for _, privKey := range endpoint.ClientConfig.(*TendermintAttestorConfig).AttestorPrivkeys {
		signature, err := privKey.Sign(proof)
		if err != nil {
			return nil, err
		}

		proofWithAttestations.Attestations = append(proofWithAttestations.Attestations, &ibctmattestor.Attestation{
			PubKey:    privKey.PubKey().Bytes(),
			Signature: signature,
		})
	}

	return proofWithAttestations.Marshal()
}

// ConnOpenInit will construct and execute a MsgConnectionOpenInit on the associated endpoint.
func (endpoint *Endpoint) ConnOpenInit() error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		endpoint.ClientID,
		endpoint.Counterparty.ClientID,
		endpoint.Counterparty.Chain.GetPrefix(), DefaultOpenInitVersion, endpoint.ConnectionConfig.DelayPeriod,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ConnectionID, err = ParseConnectionIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.T, err)

	return nil
}

// ConnOpenTry will construct and execute a MsgConnectionOpenTry on the associated endpoint.
func (endpoint *Endpoint) ConnOpenTry() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	counterpartyClient, proofClient, proofConsensus, consensusHeight, proofInit, proofHeight := endpoint.QueryConnectionHandshakeProof()

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		proofInit, err = endpoint.GetProofWithAttestations(proofInit)
		if err != nil {
			return err
		}
	}

	msg := connectiontypes.NewMsgConnectionOpenTry(
		endpoint.ClientID, endpoint.Counterparty.ConnectionID, endpoint.Counterparty.ClientID,
		counterpartyClient, endpoint.Counterparty.Chain.GetPrefix(), []*connectiontypes.Version{ConnectionVersion}, endpoint.ConnectionConfig.DelayPeriod,
		proofInit, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ConnectionID, err = ParseConnectionIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.T, err)
	return nil
}

// ConnOpenAck will construct and execute a MsgConnectionOpenAck on the associated endpoint.
func (endpoint *Endpoint) ConnOpenAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	counterpartyClient, proofClient, proofConsensus, consensusHeight, proofTry, proofHeight := endpoint.QueryConnectionHandshakeProof()
	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		proofTry, err = endpoint.GetProofWithAttestations(proofTry)
		if err != nil {
			return err
		}
	}

	msg := connectiontypes.NewMsgConnectionOpenAck(
		endpoint.ConnectionID, endpoint.Counterparty.ConnectionID, counterpartyClient, // testing doesn't use flexible selection
		proofTry, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		ConnectionVersion,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ConnOpenConfirm will construct and execute a MsgConnectionOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ConnOpenConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(connectionKey)

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		proof, err = endpoint.GetProofWithAttestations(proof)
		if err != nil {
			return err
		}
	}
	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		endpoint.ConnectionID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// QueryConnectionHandshakeProof returns all the proofs necessary to execute OpenTry or Open Ack of
// the connection handshakes. It returns the counterparty client state, proof of the counterparty
// client state, proof of the counterparty consensus state, the consensus state height, proof of
// the counterparty connection, and the proof height for all the proofs returned.
func (endpoint *Endpoint) QueryConnectionHandshakeProof() (
	clientState exported.ClientState, proofClient,
	proofConsensus []byte, consensusHeight clienttypes.Height,
	proofConnection []byte, proofHeight clienttypes.Height,
) {
	// obtain the client state on the counterparty chain
	clientState = endpoint.Counterparty.Chain.GetClientState(endpoint.Counterparty.ClientID)

	// query proof for the client state on the counterparty
	clientKey := host.FullClientStateKey(endpoint.Counterparty.ClientID)
	proofClient, proofHeight = endpoint.Counterparty.QueryProof(clientKey)

	consensusHeight = clientState.GetLatestHeight().(clienttypes.Height)

	// query proof for the consensus state on the counterparty
	consensusKey := host.FullConsensusStateKey(endpoint.Counterparty.ClientID, consensusHeight)
	proofConsensus, _ = endpoint.Counterparty.QueryProofAtHeight(consensusKey, proofHeight.GetRevisionHeight())

	// query proof for the connection on the counterparty
	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proofConnection, _ = endpoint.Counterparty.QueryProofAtHeight(connectionKey, proofHeight.GetRevisionHeight())

	return clientState, proofClient, proofConsensus, consensusHeight, proofConnection, proofHeight
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit on the associated endpoint.
func (endpoint *Endpoint) ChanOpenInit() error {
	msg := channeltypes.NewMsgChannelOpenInit(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order, []string{endpoint.ConnectionID},
		endpoint.Counterparty.ChannelConfig.PortID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ChannelID, err = ParseChannelIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.T, err)

	// update version to selected app version
	// NOTE: this update must be performed after SendMsgs()
	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry on the associated endpoint.
func (endpoint *Endpoint) ChanOpenTry() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		proof, err = endpoint.GetProofWithAttestations(proof)
		if err != nil {
			return err
		}
	}

	msg := channeltypes.NewMsgChannelOpenTry(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order, []string{endpoint.ConnectionID},
		endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelConfig.Version,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ChannelID, err = ParseChannelIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.T, err)

	// update version to selected app version
	// NOTE: this update must be performed after the endpoint channelID is set
	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version
	endpoint.Counterparty.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck on the associated endpoint.
func (endpoint *Endpoint) ChanOpenAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		proof, err = endpoint.GetProofWithAttestations(proof)
		if err != nil {
			return err
		}
	}

	msg := channeltypes.NewMsgChannelOpenAck(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelConfig.Version, // testing doesn't use flexible selection
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	if err = endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version
	endpoint.Counterparty.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ChanOpenConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		proof, err = endpoint.GetProofWithAttestations(proof)
		if err != nil {
			return err
		}
	}

	msg := channeltypes.NewMsgChannelOpenConfirm(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ChanCloseInit will construct and execute a MsgChannelCloseInit on the associated endpoint.
//
// NOTE: does not work with ibc-transfer module
func (endpoint *Endpoint) ChanCloseInit() error {
	msg := channeltypes.NewMsgChannelCloseInit(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

func (endpoint *Endpoint) UpgradeChannel(connectionHops []string, counterpartyConnectionHops []string) {
	upgradeFields := channeltypes.UpgradeFields{
		Ordering:       endpoint.ChannelConfig.Order,
		Version:        endpoint.ChannelConfig.Version,
		ConnectionHops: connectionHops,
	}

	err := endpoint.ChanUpgradeInit(upgradeFields)
	require.NoError(endpoint.Chain.T, err)

	err = endpoint.Counterparty.ChanUpgradeTry(counterpartyConnectionHops)
	require.NoError(endpoint.Counterparty.Chain.T, err)

	err = endpoint.ChanUpgradeAck()
	require.NoError(endpoint.Chain.T, err)

	err = endpoint.Counterparty.ChanUpgradeConfirm()
	require.NoError(endpoint.Counterparty.Chain.T, err)

	err = endpoint.ChanUpgradeOpen()
	require.NoError(endpoint.Chain.T, err)

	connResponse, err := endpoint.Chain.QueryServer.Connection(endpoint.Chain.GetContext(), &connectiontypes.QueryConnectionRequest{
		ConnectionId: connectionHops[0],
	})
	require.NoError(endpoint.Chain.T, err)
	endpoint.ClientID = connResponse.Connection.ClientId
	endpoint.ConnectionID = connectionHops[0]

	connResponse, err = endpoint.Counterparty.Chain.QueryServer.Connection(endpoint.Counterparty.Chain.GetContext(), &connectiontypes.QueryConnectionRequest{
		ConnectionId: counterpartyConnectionHops[0],
	})
	require.NoError(endpoint.Counterparty.Chain.T, err)
	endpoint.Counterparty.ClientID = connResponse.Connection.ClientId
	endpoint.Counterparty.ConnectionID = counterpartyConnectionHops[0]

	endpoint.ChannelConfig.Version = upgradeFields.Version
	endpoint.Counterparty.ChannelConfig.Version = upgradeFields.Version
}

func (endpoint *Endpoint) ChanUpgradeInit(upgradeFields channeltypes.UpgradeFields) error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	msg := channeltypes.NewMsgChannelUpgradeInit(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		upgradeFields,
		endpoint.Chain.App.GetIBCKeeper().GetAuthority(),
	)
	_, err = endpoint.Chain.App.GetIBCKeeper().ChannelUpgradeInit(
		endpoint.Chain.GetContext(),
		msg,
	)
	if err != nil {
		return err
	}

	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)
	return nil
}

func (endpoint *Endpoint) ChanUpgradeTry(connectionHops []string) error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	channelProof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	upgradeKey := host.ChannelUpgradeKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	upgradeProof, upgradeHeight := endpoint.Counterparty.Chain.QueryProof(upgradeKey)

	if height.Compare(upgradeHeight) != 0 {
		return fmt.Errorf("height mismatch: %s != %s", height.String(), upgradeHeight.String())
	}

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		channelProof, err = endpoint.GetProofWithAttestations(channelProof)
		if err != nil {
			return err
		}

		upgradeProof, err = endpoint.GetProofWithAttestations(upgradeProof)
		if err != nil {
			return err
		}
	}

	channelResponse, err := endpoint.Counterparty.Chain.QueryServer.Channel(endpoint.Counterparty.Chain.GetContext(), &channeltypes.QueryChannelRequest{
		PortId:    endpoint.Counterparty.ChannelConfig.PortID,
		ChannelId: endpoint.Counterparty.ChannelID,
	})
	if err != nil {
		return err
	}
	counterpartyUpgradeSequence := channelResponse.Channel.UpgradeSequence

	upgradeResponse, err := endpoint.Counterparty.Chain.QueryServer.Upgrade(endpoint.Counterparty.Chain.GetContext(), &channeltypes.QueryUpgradeRequest{
		PortId:    endpoint.Counterparty.ChannelConfig.PortID,
		ChannelId: endpoint.Counterparty.ChannelID,
	})
	if err != nil {
		return err
	}

	msg := channeltypes.NewMsgChannelUpgradeTry(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		connectionHops,
		upgradeResponse.Upgrade.Fields,
		counterpartyUpgradeSequence,
		channelProof, upgradeProof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}
	return nil
}

func (endpoint *Endpoint) ChanUpgradeAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	channelProof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	upgradeKey := host.ChannelUpgradeKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	upgradeProof, upgradeHeight := endpoint.Counterparty.Chain.QueryProof(upgradeKey)

	if height.Compare(upgradeHeight) != 0 {
		return fmt.Errorf("height mismatch: %s != %s", height.String(), upgradeHeight.String())
	}

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		channelProof, err = endpoint.GetProofWithAttestations(channelProof)
		if err != nil {
			return err
		}

		upgradeProof, err = endpoint.GetProofWithAttestations(upgradeProof)
		if err != nil {
			return err
		}
	}

	upgradeResponse, err := endpoint.Counterparty.Chain.QueryServer.Upgrade(endpoint.Counterparty.Chain.GetContext(), &channeltypes.QueryUpgradeRequest{
		PortId:    endpoint.Counterparty.ChannelConfig.PortID,
		ChannelId: endpoint.Counterparty.ChannelID,
	})
	if err != nil {
		return err
	}

	msg := channeltypes.NewMsgChannelUpgradeAck(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		upgradeResponse.Upgrade,
		channelProof, upgradeProof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	upgradeAckResponse := channeltypes.MsgChannelUpgradeAckResponse{}
	values, err := GetMarshaledValue(res.Data)
	if err != nil {
		return err
	}
	err = upgradeAckResponse.Unmarshal(values[0])
	if err != nil {
		return err
	}

	if upgradeAckResponse.Result == channeltypes.FAILURE {
		return fmt.Errorf("upgrade result: %s", upgradeAckResponse.Result)
	}
	return nil
}

func (endpoint *Endpoint) ChanUpgradeConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	channelProof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	upgradeKey := host.ChannelUpgradeKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	upgradeProof, upgradeHeight := endpoint.Counterparty.Chain.QueryProof(upgradeKey)

	if height.Compare(upgradeHeight) != 0 {
		return fmt.Errorf("height mismatch: %s != %s", height.String(), upgradeHeight.String())
	}

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		channelProof, err = endpoint.GetProofWithAttestations(channelProof)
		if err != nil {
			return err
		}

		upgradeProof, err = endpoint.GetProofWithAttestations(upgradeProof)
		if err != nil {
			return err
		}
	}

	channelResponse, err := endpoint.Counterparty.Chain.QueryServer.Channel(endpoint.Counterparty.Chain.GetContext(), &channeltypes.QueryChannelRequest{
		PortId:    endpoint.Counterparty.ChannelConfig.PortID,
		ChannelId: endpoint.Counterparty.ChannelID,
	})
	if err != nil {
		return err
	}
	upgradeResponse, err := endpoint.Counterparty.Chain.QueryServer.Upgrade(endpoint.Counterparty.Chain.GetContext(), &channeltypes.QueryUpgradeRequest{
		PortId:    endpoint.Counterparty.ChannelConfig.PortID,
		ChannelId: endpoint.Counterparty.ChannelID,
	})
	if err != nil {
		return err
	}

	msg := channeltypes.NewMsgChannelUpgradeConfirm(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		channelResponse.Channel.State,
		upgradeResponse.Upgrade,
		channelProof, upgradeProof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}
	return nil
}

func (endpoint *Endpoint) ChanUpgradeOpen() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.T, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	channelProof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	if endpoint.ClientConfig.GetClientType() == exported.TendermintAttestor {
		channelProof, err = endpoint.GetProofWithAttestations(channelProof)
		if err != nil {
			return err
		}
	}

	channelResponse, err := endpoint.Counterparty.Chain.QueryServer.Channel(endpoint.Counterparty.Chain.GetContext(), &channeltypes.QueryChannelRequest{
		PortId:    endpoint.Counterparty.ChannelConfig.PortID,
		ChannelId: endpoint.Counterparty.ChannelID,
	})
	if err != nil {
		return err
	}

	msg := channeltypes.NewMsgChannelUpgradeOpen(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		channelResponse.Channel.State,
		channelResponse.Channel.UpgradeSequence,
		channelProof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}
	return nil
}

// SendPacket sends a packet through the channel keeper using the associated endpoint
// The counterparty client is updated so proofs can be sent to the counterparty chain.
// The packet sequence generated for the packet to be sent is returned. An error
// is returned if one occurs.
func (endpoint *Endpoint) SendPacket(
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	channelCap := endpoint.Chain.GetChannelCapability(endpoint.ChannelConfig.PortID, endpoint.ChannelID)

	// no need to send message, acting as a module
	sequence, err := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SendPacket(endpoint.Chain.GetContext(), channelCap, endpoint.ChannelConfig.PortID, endpoint.ChannelID, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

	// commit changes since no message was sent
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	err = endpoint.Counterparty.UpdateClient()
	if err != nil {
		return 0, err
	}

	return sequence, nil
}

// RecvPacket receives a packet on the associated endpoint.
// The counterparty client is updated.
func (endpoint *Endpoint) RecvPacket(packet channeltypes.Packet) error {
	_, err := endpoint.RecvPacketWithResult(packet)
	if err != nil {
		return err
	}

	return nil
}

// RecvPacketWithResult receives a packet on the associated endpoint and the result
// of the transaction is returned. The counterparty client is updated.
func (endpoint *Endpoint) RecvPacketWithResult(packet channeltypes.Packet) (*abci.ExecTxResult, error) {
	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := endpoint.Counterparty.Chain.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	// receive on counterparty and update source client
	res, err := endpoint.Chain.SendMsgs(recvMsg)
	if err != nil {
		return nil, err
	}

	if err := endpoint.Counterparty.UpdateClient(); err != nil {
		return nil, err
	}

	return res, nil
}

// WriteAcknowledgement writes an acknowledgement on the channel associated with the endpoint.
// The counterparty client is updated.
func (endpoint *Endpoint) WriteAcknowledgement(ack exported.Acknowledgement, packet exported.PacketI) error {
	channelCap := endpoint.Chain.GetChannelCapability(packet.GetDestPort(), packet.GetDestChannel())

	// no need to send message, acting as a handler
	err := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.WriteAcknowledgement(endpoint.Chain.GetContext(), channelCap, packet, ack)
	if err != nil {
		return err
	}

	// commit changes since no message was sent
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	return endpoint.Counterparty.UpdateClient()
}

// AcknowledgePacket sends a MsgAcknowledgement to the channel associated with the endpoint.
func (endpoint *Endpoint) AcknowledgePacket(packet channeltypes.Packet, ack []byte) error {
	// get proof of acknowledgement on counterparty
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	return endpoint.Chain.sendMsgs(ackMsg)
}

// TimeoutPacket sends a MsgTimeout to the channel associated with the endpoint.
func (endpoint *Endpoint) TimeoutPacket(packet channeltypes.Packet) error {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch endpoint.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return fmt.Errorf("unsupported order type %s", endpoint.ChannelConfig.Order)
	}

	counterparty := endpoint.Counterparty
	proof, proofHeight := counterparty.QueryProof(packetKey)
	nextSeqRecv, found := counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(counterparty.Chain.GetContext(), counterparty.ChannelConfig.PortID, counterparty.ChannelID)
	require.True(endpoint.Chain.T, found)

	timeoutMsg := channeltypes.NewMsgTimeout(
		packet, nextSeqRecv,
		proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(timeoutMsg)
}

// TimeoutOnClose sends a MsgTimeoutOnClose to the channel associated with the endpoint.
func (endpoint *Endpoint) TimeoutOnClose(packet channeltypes.Packet) error {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch endpoint.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return fmt.Errorf("unsupported order type %s", endpoint.ChannelConfig.Order)
	}

	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
	proofClosed, _ := endpoint.Counterparty.QueryProof(channelKey)

	nextSeqRecv, found := endpoint.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(endpoint.Counterparty.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	require.True(endpoint.Chain.T, found)

	timeoutOnCloseMsg := channeltypes.NewMsgTimeoutOnClose(
		packet, nextSeqRecv,
		proof, proofClosed, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(timeoutOnCloseMsg)
}

// SetChannelState sets a channel state
func (endpoint *Endpoint) SetChannelState(state channeltypes.State) error {
	channel := endpoint.GetChannel()

	channel.State = state
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, channel)

	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	return endpoint.Counterparty.UpdateClient()
}

// GetClientState retrieves the Client State for this endpoint. The
// client state is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetClientState() exported.ClientState {
	return endpoint.Chain.GetClientState(endpoint.ClientID)
}

// SetClientState sets the client state for this endpoint.
func (endpoint *Endpoint) SetClientState(clientState exported.ClientState) {
	endpoint.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(endpoint.Chain.GetContext(), endpoint.ClientID, clientState)
}

// GetConsensusState retrieves the Consensus State for this endpoint at the provided height.
// The consensus state is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetConsensusState(height exported.Height) exported.ConsensusState {
	consensusState, found := endpoint.Chain.GetConsensusState(endpoint.ClientID, height)
	require.True(endpoint.Chain.T, found)

	return consensusState
}

// SetConsensusState sets the consensus state for this endpoint.
func (endpoint *Endpoint) SetConsensusState(consensusState exported.ConsensusState, height exported.Height) {
	endpoint.Chain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(endpoint.Chain.GetContext(), endpoint.ClientID, height, consensusState)
}

// GetConnection retrieves an IBC Connection for the endpoint. The
// connection is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetConnection() connectiontypes.ConnectionEnd {
	connection, found := endpoint.Chain.App.GetIBCKeeper().ConnectionKeeper.GetConnection(endpoint.Chain.GetContext(), endpoint.ConnectionID)
	require.True(endpoint.Chain.T, found)

	return connection
}

// SetConnection sets the connection for this endpoint.
func (endpoint *Endpoint) SetConnection(connection connectiontypes.ConnectionEnd) {
	endpoint.Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(endpoint.Chain.GetContext(), endpoint.ConnectionID, connection)
}

// GetChannel retrieves an IBC Channel for the endpoint. The channel
// is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetChannel() channeltypes.Channel {
	channel, found := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	require.True(endpoint.Chain.T, found)

	return channel
}

// SetChannel sets the channel for this endpoint.
func (endpoint *Endpoint) SetChannel(channel channeltypes.Channel) {
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, channel)
}

// QueryClientStateProof performs and abci query for a client stat associated
// with this endpoint and returns the ClientState along with the proof.
func (endpoint *Endpoint) QueryClientStateProof() (exported.ClientState, []byte) {
	// retrieve client state to provide proof for
	clientState := endpoint.GetClientState()

	clientKey := host.FullClientStateKey(endpoint.ClientID)
	proofClient, _ := endpoint.QueryProof(clientKey)

	return clientState, proofClient
}
