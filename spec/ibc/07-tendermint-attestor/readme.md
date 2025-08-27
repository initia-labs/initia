# 07-Tendermint-Attestor Light Client Specification

## Overview

The 07-tendermint-attestor light client is an enhanced implementation of the standard Tendermint light client that incorporates an attestation mechanism for enhanced security and trust. This light client extends the base 07-tendermint light client with additional attestor public keys and threshold-based signature verification.

## Architecture

### Client State Structure

The `ClientState` embeds the standard Tendermint light client state and extends it with attestor-specific fields:

```go
type ClientState struct {
    *tmlightclient.ClientState  // Embedded standard Tendermint client state
    AttestorPubkeys []codectypes.Any  // List of authorized attestor public keys
    Threshold       uint32             // Minimum number of attestations required
}
```

### Consensus State

The consensus state inherits from the standard Tendermint consensus state without modifications:

```go
type ConsensusState struct {
    *tmlightclient.ConsensusState
}
```

## Core Components

### 1. Attestation Mechanism

#### Attestation Structure

```go
type Attestation struct {
    PubKey    codectypes.Any  // Attestor's public key
    Signature []byte          // Signature over proof bytes
}
```

#### Merkle Proof with Attestations

```go
type MerkleProofBytesWithAttestations struct {
    ProofBytes   []byte        // Merkle proof bytes
    Attestations []*Attestation // Collection of attestations
}
```

### 2. Signature Verification

The light client implements a threshold-based signature verification system:

- **Threshold Check**: Verifies that the number of attestations meets or exceeds the configured threshold
- **Attestor Authorization**: Ensures all attestations come from authorized attestor public keys
- **Signature Validation**: Verifies cryptographic signatures over proof bytes

#### Verification Process

1. Check if threshold is met (number of attestations ≥ threshold)
2. Validate each attestation's public key against the authorized list
3. Verify each attestation's signature over the proof bytes
4. Return success only if all verifications pass

### 3. Client State Management

#### Status Determination

The client can be in one of three states:

- **Active**: FrozenHeight is zero and client is not expired
- **Frozen**: FrozenHeight is not zero (client has been frozen due to misbehaviour)
- **Expired**: Latest consensus state timestamp + trusting period ≤ current time

#### Validation Rules

- Inherits all validation rules from the standard Tendermint light client
- Additional validation for attestor public keys and threshold configuration

### 4. Update Mechanisms

#### Header Verification

The light client verifies headers through a multi-step process:

1. **Trusted Consensus State Retrieval**: Fetches the trusted consensus state at the specified height
2. **Header Validation**: Ensures header height is newer than the trusted consensus state
3. **Revision Consistency**: Verifies header revision matches the trusted consensus state revision
4. **Validator Set Verification**: Validates the new validator set against the trusted validator set
5. **Attestation Verification**: If attestations are present, verifies them using the threshold mechanism

#### Misbehaviour Handling

- Detects conflicting headers at the same height
- Freezes the client upon detection of misbehaviour
- Prevents further updates until manual intervention

### 5. Consensus State Updates

#### Update Process

1. **Height Validation**: Ensures new height is greater than current height
2. **Timestamp Validation**: Verifies timestamp is within acceptable bounds
3. **Validator Set Update**: Updates the validator set if changes are detected
4. **Attestation Processing**: Processes and verifies any accompanying attestations

## Security Model

### Trust Assumptions

- **Attestor Trust**: Assumes attestor public keys are correctly configured and trusted
- **Threshold Security**: Relies on the threshold mechanism to prevent single-point-of-failure attacks
- **Cryptographic Security**: Depends on the security of the underlying signature schemes

## Configuration Parameters

### Required Parameters

- `chainID`: The chain identifier
- `trustLevel`: Fraction of validators required for trust
- `trustingPeriod`: Duration for which the client trusts the consensus state
- `unbondingPeriod`: Duration for unbonding validators
- `maxClockDrift`: Maximum allowed clock drift
- `latestHeight`: Initial height for the client
- `proofSpecs`: Proof specifications for verification
- `upgradePath`: Path for client upgrades

### Attestor-Specific Parameters

- `attestorPubkeys`: List of authorized attestor public keys
- `threshold`: Minimum number of attestations required for updates

## Conclusion

The 07-tendermint-attestor light client provides an enhanced security model for IBC light clients by incorporating attestation-based verification. This implementation maintains compatibility with the standard Tendermint light client while adding an additional layer of trust through multiple attestor verification.

The threshold-based approach ensures that no single attestor can compromise the client's security, while the flexible configuration allows for various deployment scenarios. This makes it suitable for high-security applications where additional trust verification is required beyond the standard Tendermint consensus mechanism.
