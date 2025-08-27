# Migrate Delegation Specification

## Overview

The Migrate Delegation functionality in the Initia mstaking module is a **specialized function designed to handle the specific case of changing underlying assets** (like USDC to USDT) within the Initia ecosystem. When the ecosystem needs to transition from one stablecoin or underlying asset to another, this feature allows delegators to seamlessly migrate their staked LP (Liquidity Provider) tokens without going through the traditional unbonding period.

This is **not a general-purpose portfolio rebalancing tool**, but rather a **governance-controlled mechanism** to facilitate ecosystem-wide asset transitions while maintaining staking rewards and validator relationships.

## Architecture

### Core Components

1. **Migration Registration System**: Allows governance to register valid migration paths between LP token pairs for ecosystem asset transitions
2. **Migration Execution**: Enables delegators to execute migrations using registered paths when ecosystem changes are announced
3. **DEX Migration Integration**: Integrates with Move-based DEX migration contracts for token conversion during asset transitions
4. **State Management**: Maintains migration registrations and tracks migration events for ecosystem transition monitoring

### Data Structures

#### DelegationMigration

```protobuf
message DelegationMigration {
  // denom_lp_from is the source LP token denomination
  string denom_lp_from = 1;
  // denom_lp_to is the target LP token denomination
  string denom_lp_to = 2;
  // module_address is the address of the migration module
  bytes module_address = 3;
  // module_name is the name of the migration module
  string module_name = 4;
}
```

## Message Types

### MsgRegisterMigration

**Purpose**: Register a migration path between two LP token denominations for ecosystem asset transitions (e.g., USDC → USDT)

**Signer**: Authority (governance)

**Parameters**:

- `authority`: Governance authority address
- `denom_lp_from`: Source LP token denomination
- `denom_lp_to`: Target LP token denomination  
- `module_address`: Move module address for the migration contract
- `module_name`: Move module name for the migration contract

**Validation**:

- Authority must be valid governance address
- Both LP denominations must exist in balancer pools
- Module address must be a valid Move address
- Module name must be a valid Move module name

### MsgMigrateDelegation

**Purpose**: Execute a delegation migration using registered migration path when ecosystem asset transitions are announced

**Signer**: Delegator

**Parameters**:

- `delegator_address`: Address of the delegator
- `validator_address`: Address of the validator
- `denom_lp_from`: Source LP token denomination
- `denom_lp_to`: Target LP token denomination
- `new_delegator_address`: Optional new delegator address. If not provided, the original delegator address will be used

**Validation**:

- Delegator must have active delegation with specified validator
- Migration path must be registered for the source and target LP denominations
- Target LP denomination must be in bond denoms list
- If `new_delegator_address` is provided, it must be a valid address
- Source and target LP denominations must be different

## Migration Process

### 1. Migration Registration (Governance)

1. **Ecosystem Decision**: Governance decides to transition from one underlying asset to another (e.g., USDC → USDT)
2. **Validation**: Verify both LP denominations exist in balancer pools
3. **Module Verification**: Validate migration module exists and implements required interface
4. **State Storage**: Store migration configuration in `Migrations` collection
5. **Event Emission**: Emit `EventTypeRegisterMigration` event
6. **Community Notification**: Announce the planned ecosystem transition to delegators

### 2. Migration Execution (Delegator)

1. **Ecosystem Transition**: Delegator responds to announced ecosystem asset transition
2. **Path Lookup**: Retrieve registered migration for source LP and target LP denominations
3. **Delegation Unbonding**: Unbond delegation shares from validator (instant unbonding, no waiting period)
4. **Liquidity Migration**: Execute migration through the registered migration module which handles:
   - Withdrawing liquidity from source DEX pool
   - Converting underlying tokens through the migration module's `convert` function (e.g., USDC → USDT)
   - Providing liquidity to target DEX pool
   - Managing fee rates during migration
5. **Re-delegation**: New delegator delegates LP tokens back to same validator (if new delegator specified, otherwise original delegator)
6. **Event Emission**: Emit `EventTypeMigrateDelegation` event

### 3. Migration Module Requirements

The migration module must implement the following Move functions:

```move
// Execute token conversion during migration
public entry fun convert(
    account: &signer,
    coin_in: Object<Metadata>,
    coin_out: Object<Metadata>,
    amount: u64
)
```

## Security Considerations

### Access Control

- Only governance authority can register migrations for ecosystem asset transitions
- Only delegators can execute migrations for their own delegations
- Validators cannot execute migrations on behalf of delegators
- Migration paths are governance-controlled to prevent abuse

### Validation Checks

- Source and target LP denominations must exist in balancer pools
- Target LP denomination must be in bond denoms list
- Migration module must be verified and implement required interface
- Migration path must be pre-registered
- Module address and name must be valid
- New delegator address (if provided) must be a valid address

### Economic Security

- Full liquidity withdrawal and re-provision process through DEX migration keeper
- Maintains validator relationship and staking rewards
- Prevents market manipulation during ecosystem transitions
- Ensures fair and transparent asset migration process

## Events

### EventTypeRegisterMigration

```go
sdk.NewEvent(
    types.EventTypeRegisterMigration,
    sdk.NewAttribute(types.AttributeKeyDenomLpFrom, msg.DenomLpFrom),
    sdk.NewAttribute(types.AttributeKeyDenomLpTo, msg.DenomLpTo),
    sdk.NewAttribute(types.AttributeKeyMigrationModule, fmt.Sprintf("%s::%s", msg.ModuleAddress, msg.ModuleName)),
)
```

### EventTypeMigrateDelegation

```go
sdk.NewEvent(
    types.EventTypeMigrateDelegation,
    sdk.NewAttribute(types.AttributeKeyDelegator, msg.DelegatorAddress),
    sdk.NewAttribute(types.AttributeKeyValidator, msg.ValidatorAddress),
    sdk.NewAttribute(types.AttributeKeyNewDelegator, msg.NewDelegatorAddress),
    sdk.NewAttribute(types.AttributeKeyDenomLpFrom, msg.DenomLpFrom),
    sdk.NewAttribute(types.AttributeKeyDenomLpTo, msg.DenomLpTo),
    sdk.NewAttribute(types.AttributeKeyOriginShares, originShares.String()),
    sdk.NewAttribute(types.AttributeKeyNewShares, newShares.String()),
)
```
