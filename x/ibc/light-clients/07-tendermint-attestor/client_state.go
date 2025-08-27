package tendermintattestor

import (
	"slices"
	"time"

	ics23 "github.com/cosmos/ics23/go"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	tmlightclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance
func NewClientState(
	chainID string, trustLevel tmlightclient.Fraction,
	trustingPeriod, ubdPeriod, maxClockDrift time.Duration,
	latestHeight clienttypes.Height, specs []*ics23.ProofSpec,
	upgradePath []string, attestorPubkeys []*codectypes.Any, threshold uint32,
) *ClientState {
	return &ClientState{
		ClientState:     tmlightclient.NewClientState(chainID, trustLevel, trustingPeriod, ubdPeriod, maxClockDrift, latestHeight, specs, upgradePath),
		AttestorPubkeys: attestorPubkeys,
		Threshold:       threshold,
	}
}

func (cs ClientState) Validate() error {
	if err := cs.ClientState.Validate(); err != nil {
		return err
	}

	if len(cs.AttestorPubkeys) < int(cs.Threshold) {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "not enough attestor pubkeys are provided")
	}

	// duplication check for attestor pubkeys
	seenPubKeys := make([]*cryptotypes.PubKey, 0, len(cs.AttestorPubkeys))
	attestorPubKeys := cs.GetAttestorPubkeys()
	for _, attestorPubkey := range attestorPubKeys {
		if slices.Contains(seenPubKeys, &attestorPubkey) {
			return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "duplicate attestor pubkey: %s", attestorPubkey.String())
		}
		seenPubKeys = append(seenPubKeys, &attestorPubkey)
	}
	return nil
}

// ClientType is tendermint attestor.
func (ClientState) ClientType() string {
	return TendermintAttestor
}

// GetTimestampAtHeight returns the timestamp in nanoseconds of the consensus state at the given height.
func (ClientState) GetTimestampAtHeight(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	// get consensus state at height from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return 0, errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "height (%s)", height)
	}
	return consState.GetTimestamp(), nil
}

// Status returns the status of the tendermint client.
// The client may be:
// - Active: FrozenHeight is zero and client is not expired
// - Frozen: Frozen Height is not zero
// - Expired: the latest consensus state timestamp + trusting period <= current time
//
// A frozen client will become expired, so the Frozen status
// has higher precedence.
func (cs ClientState) Status(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
) exported.Status {
	if !cs.FrozenHeight.IsZero() {
		return exported.Frozen
	}

	// get latest consensus state from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, cs.GetLatestHeight())
	if !found {
		// if the client state does not have an associated consensus state for its latest height
		// then it must be expired
		return exported.Expired
	}

	if cs.IsExpired(consState.Timestamp, ctx.BlockTime()) {
		return exported.Expired
	}

	return exported.Active
}

// ZeroCustomFields returns a ClientState that is a copy of the current ClientState
// with all client customizable fields zeroed out
func (cs ClientState) ZeroCustomFields() exported.ClientState {
	// copy over all chain-specified fields
	// and leave custom fields empty
	return &ClientState{
		ClientState:     cs.ClientState.ZeroCustomFields().(*tmlightclient.ClientState),
		AttestorPubkeys: []*codectypes.Any{},
		Threshold:       0,
	}
}

// Initialize checks that the initial consensus state is an 07-tendermint consensus state and
// sets the client state, consensus state and associated metadata in the provided client store.
func (cs ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, consState exported.ConsensusState) error {
	consensusState, ok := consState.(*ConsensusState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T",
			&ConsensusState{}, consState)
	}

	setClientState(clientStore, cdc, &cs)
	setConsensusState(clientStore, cdc, consensusState, cs.GetLatestHeight())
	setConsensusMetadata(ctx, clientStore, cs.GetLatestHeight())

	return nil
}

// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// If a zero proof height is passed in, it will fail to retrieve the associated consensus state.
func (cs ClientState) VerifyMembership(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proofWithAttestations []byte,
	path exported.Path,
	value []byte,
) error {
	if cs.GetLatestHeight().LT(height) {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	if err := verifyDelayPeriodPassed(ctx, clientStore, height, delayTimePeriod, delayBlockPeriod); err != nil {
		return err
	}

	var merkleProofBytesWithAttestations MerkleProofBytesWithAttestations
	if err := cdc.Unmarshal(proofWithAttestations, &merkleProofBytesWithAttestations); err != nil {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into ICS 23 commitment merkle proof")
	}

	if err := cs.VerifySignatures(ctx, merkleProofBytesWithAttestations.ProofBytes, merkleProofBytesWithAttestations.Attestations); err != nil {
		return err
	}

	var merkleProof commitmenttypes.MerkleProof
	if err := cdc.Unmarshal(merkleProofBytesWithAttestations.ProofBytes, &merkleProof); err != nil {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into ICS 23 commitment merkle proof")
	}

	merklePath, ok := path.(commitmenttypes.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypes.MerklePath{}, path)
	}

	consensusState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "please ensure the proof was constructed against a height that exists on the client")
	}

	return merkleProof.VerifyMembership(cs.ProofSpecs, consensusState.GetRoot(), merklePath, value)
}

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// If a zero proof height is passed in, it will fail to retrieve the associated consensus state.
func (cs ClientState) VerifyNonMembership(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proofWithAttestations []byte,
	path exported.Path,
) error {
	if cs.GetLatestHeight().LT(height) {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	if err := verifyDelayPeriodPassed(ctx, clientStore, height, delayTimePeriod, delayBlockPeriod); err != nil {
		return err
	}

	var merkleProofBytesWithAttestations MerkleProofBytesWithAttestations
	if err := cdc.Unmarshal(proofWithAttestations, &merkleProofBytesWithAttestations); err != nil {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into ICS 23 commitment merkle proof")
	}

	if err := cs.VerifySignatures(ctx, merkleProofBytesWithAttestations.ProofBytes, merkleProofBytesWithAttestations.Attestations); err != nil {
		return err
	}

	var merkleProof commitmenttypes.MerkleProof
	if err := cdc.Unmarshal(merkleProofBytesWithAttestations.ProofBytes, &merkleProof); err != nil {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into ICS 23 commitment merkle proof")
	}

	merklePath, ok := path.(commitmenttypes.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypes.MerklePath{}, path)
	}

	consensusState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "please ensure the proof was constructed against a height that exists on the client")
	}

	return merkleProof.VerifyNonMembership(cs.ProofSpecs, consensusState.GetRoot(), merklePath)
}

// verifyDelayPeriodPassed will ensure that at least delayTimePeriod amount of time and delayBlockPeriod number of blocks have passed
// since consensus state was submitted before allowing verification to continue.
func verifyDelayPeriodPassed(ctx sdk.Context, store storetypes.KVStore, proofHeight exported.Height, delayTimePeriod, delayBlockPeriod uint64) error {
	if delayTimePeriod != 0 {
		// check that executing chain's timestamp has passed consensusState's processed time + delay time period
		processedTime, ok := GetProcessedTime(store, proofHeight)
		if !ok {
			return errorsmod.Wrapf(tmlightclient.ErrProcessedTimeNotFound, "processed time not found for height: %s", proofHeight)
		}

		currentTimestamp := uint64(ctx.BlockTime().UnixNano())
		validTime := processedTime + delayTimePeriod

		// NOTE: delay time period is inclusive, so if currentTimestamp is validTime, then we return no error
		if currentTimestamp < validTime {
			return errorsmod.Wrapf(tmlightclient.ErrDelayPeriodNotPassed, "cannot verify packet until time: %d, current time: %d",
				validTime, currentTimestamp)
		}

	}

	if delayBlockPeriod != 0 {
		// check that executing chain's height has passed consensusState's processed height + delay block period
		processedHeight, ok := GetProcessedHeight(store, proofHeight)
		if !ok {
			return errorsmod.Wrapf(tmlightclient.ErrProcessedHeightNotFound, "processed height not found for height: %s", proofHeight)
		}

		currentHeight := clienttypes.GetSelfHeight(ctx)
		validHeight := clienttypes.NewHeight(processedHeight.GetRevisionNumber(), processedHeight.GetRevisionHeight()+delayBlockPeriod)

		// NOTE: delay block period is inclusive, so if currentHeight is validHeight, then we return no error
		if currentHeight.LT(validHeight) {
			return errorsmod.Wrapf(tmlightclient.ErrDelayPeriodNotPassed, "cannot verify packet until height: %s, current height: %s",
				validHeight, currentHeight)
		}
	}

	return nil
}

// GetAttestorPubkeys returns the attestor pubkeys for the client state.
func (cs ClientState) GetAttestorPubkeys() []cryptotypes.PubKey {
	pubKeys := make([]cryptotypes.PubKey, 0, len(cs.AttestorPubkeys))
	for _, attestorPubkey := range cs.AttestorPubkeys {
		pubKeys = append(pubKeys, attestorPubkey.GetCachedValue().(cryptotypes.PubKey))
	}
	return pubKeys
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (cs *ClientState) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for i := range cs.AttestorPubkeys {
		var pubKey cryptotypes.PubKey
		err := unpacker.UnpackAny(cs.AttestorPubkeys[i], &pubKey)
		if err != nil {
			return err
		}
	}
	return nil
}

// hasSameAttestorsAndThreshold returns true if the attestors and threshold are the same between the two client states
func (cs ClientState) hasSameAttestorsAndThreshold(cs2 ClientState) bool {
	if cs.Threshold != cs2.Threshold {
		return false
	}

	pubkeys1 := cs.GetAttestorPubkeys()
	pubkeys2 := cs2.GetAttestorPubkeys()
	if len(pubkeys1) != len(pubkeys2) {
		return false
	}

	for _, pubkey := range pubkeys1 {
		if !slices.ContainsFunc(pubkeys2, func(pubkey2 cryptotypes.PubKey) bool {
			return pubkey.Equals(pubkey2)
		}) {
			return false
		}
	}

	return true
}
