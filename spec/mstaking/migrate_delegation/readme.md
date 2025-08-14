# Migrate Delegation Specification

## Overview

The Migrate Delegation functionality in the Initia mstaking module is a **specialized function designed to handle the specific case of changing underlying assets** (like USDC to iUSD) within the Initia ecosystem. When the ecosystem needs to transition from one stablecoin or underlying asset to another, this feature allows delegators to seamlessly migrate their staked LP (Liquidity Provider) tokens without going through the traditional unbonding period.

This is **not a general-purpose portfolio rebalancing tool**, but rather a **governance-controlled mechanism** to facilitate ecosystem-wide asset transitions while maintaining staking rewards and validator relationships.

## Architecture

### Core Components

1. **Migration Registration System**: Allows governance to register valid migration paths between LP token pairs for ecosystem asset transitions
2. **Migration Execution**: Enables delegators to execute migrations using registered paths when ecosystem changes are announced
3. **Swap Contract Integration**: Integrates with Move-based swap contracts for token conversion during asset transitions
4. **State Management**: Maintains migration registrations and tracks migration events for ecosystem transition monitoring

### Data Structures

#### DelegationMigration

```protobuf
message DelegationMigration {
  // denom_in is the input denom of the swap contract
  string denom_in = 1;
  // denom_out is the output denom of the swap contract
  string denom_out = 2;
  // lp_denom_in is the denom of the lp token in
  string lp_denom_in = 3;
  // lp_denom_out is the denom of the lp token out
  string lp_denom_out = 4;
  // swap_contract_module_address is the address of the swap contract module
  bytes swap_contract_module_address = 5;
  // swap_contract_module_name is the name of the swap contract module
  string swap_contract_module_name = 6;
}
```

## Message Types

### MsgRegisterMigration

**Purpose**: Register a migration path between two LP token denominations for ecosystem asset transitions (e.g., USDC → iUSD)

**Signer**: Authority (governance)

**Parameters**:

- `authority`: Governance authority address
- `lp_denom_in`: Source LP token denomination
- `lp_denom_out`: Target LP token denomination  
- `denom_in`: Source underlying token denomination
- `denom_out`: Target underlying token denomination
- `swap_contract_address`: Move module address in format `<module_addr>::<module_name>`

**Validation**:

- Authority must be valid governance address
- Both LP denominations must exist in balancer pools
- Swap contract must implement required interface
- Swap contract address must follow format `<module_addr>::<module_name>`

### MsgMigrateDelegation

**Purpose**: Execute a delegation migration using registered migration path when ecosystem asset transitions are announced

**Signer**: Delegator

**Parameters**:

- `delegator_address`: Address of the delegator
- `validator_address`: Address of the validator
- `lp_denom_in`: Source LP token denomination
- `lp_denom_out`: Target LP token denomination

**Validation**:

- Delegator must have active delegation with specified validator
- Migration path must be registered for the source LP denomination
- Target LP denomination must be in bond denoms list

## Migration Process

### 1. Migration Registration (Governance)

1. **Ecosystem Decision**: Governance decides to transition from one underlying asset to another (e.g., USDC → iUSD)
2. **Validation**: Verify both LP denominations exist in balancer pools
3. **Contract Verification**: Validate swap contract implements required interface for the specific asset transition
4. **State Storage**: Store migration configuration in `RegisteredMigrations` collection
5. **Event Emission**: Emit `EventTypeRegisterMigration` event
6. **Community Notification**: Announce the planned ecosystem transition to delegators

### 2. Migration Execution (Delegator)

1. **Ecosystem Transition**: Delegator responds to announced ecosystem asset transition
2. **Path Lookup**: Retrieve registered migration for source LP and target LP denominations
3. **Delegation Unbonding**: Unbond delegation shares from validator
4. **Liquidity Withdrawal**: Withdraw liquidity from source DEX pool (e.g., USDC-based pool)
5. **Token Swap**: Execute swap through registered contract (e.g., USDC → iUSD)
6. **Liquidity Provision**: Provide liquidity to target DEX pool (e.g., iUSD-based pool)
7. **Re-delegation**: Delegate new LP tokens back to same validator
8. **Event Emission**: Emit `EventTypeMigrateDelegation` event

### 3. Swap Contract Requirements

The swap contract must implement the following Move functions:

```move
// Execute token swap
public fun swap(
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
- Swap contract must be verified and implement required interface
- Migration path must be pre-registered
- Swap contract address format must be valid (`<module_addr>::<module_name>`)

### Economic Security

- Full liquidity withdrawal and re-provision process
- Maintains validator relationship and staking rewards
- Prevents market manipulation during ecosystem transitions
- Ensures fair and transparent asset migration process

## Events

### EventTypeRegisterMigration

```go
sdk.NewEvent(
    types.EventTypeRegisterMigration,
    sdk.NewAttribute(types.AttributeKeyLpDenomIn, msg.LpDenomIn),
    sdk.NewAttribute(types.AttributeKeyLpDenomOut, msg.LpDenomOut),
    sdk.NewAttribute(types.AttributeKeyDenomIn, msg.DenomIn),
    sdk.NewAttribute(types.AttributeKeyDenomOut, msg.DenomOut),
    sdk.NewAttribute(types.AttributeKeySwapContractAddress, msg.SwapContractAddress),
)
```

### EventTypeMigrateDelegation

```go
sdk.NewEvent(
    types.EventTypeMigrateDelegation,
    sdk.NewAttribute(types.AttributeKeyDelegator, msg.DelegatorAddress),
    sdk.NewAttribute(types.AttributeKeyValidator, msg.ValidatorAddress),
    sdk.NewAttribute(types.AttributeKeyLpDenomIn, msg.LpDenomIn),
    sdk.NewAttribute(types.AttributeKeyDenomIn, msg.LpDenomOut),
    sdk.NewAttribute(types.AttributeKeyOriginShares, originShares.String()),
    sdk.NewAttribute(types.AttributeKeyNewShares, newShares.String()),
)
```
