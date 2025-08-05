# Emergency Proposal

Emergency proposals are a special type of governance proposal that can be activated under specific conditions to handle urgent situations requiring immediate action through periodic tallying.

## Overview

An emergency proposal is a governance proposal that:

- Uses the same voting period as regular proposals but performs periodic tallying at short intervals
- Requires higher deposit amounts (EmergencyMinDeposit)
- Can only be activated by pre-authorized emergency submitters
- Performs continuous tallying every EmergencyTallyInterval until quorum is reached or voting period ends
- Can finish early if quorum and threshold conditions are met during any tally

## Activation Requirements

For a regular proposal to be converted into an emergency proposal, the following conditions must be met:

1. The proposal must be in the voting period (Status = StatusVotingPeriod)
2. The total deposit amount must meet or exceed the EmergencyMinDeposit threshold defined in params
3. The depositor(activator) must be in the list of authorized emergency submitters (EmergencySubmitters param)
4. The proposal must not already be marked as emergency

## How Emergency Proposals Work

### 1. Initial Submission

- A proposal is submitted normally through the governance module
- It goes through the regular deposit period and enters voting period

### 2. Activation Process

Emergency proposals can be activated in two ways:

#### A. During Deposit Processing

- When an authorized emergency submitter adds deposit that meets the `EmergencyMinDeposit` threshold
- The proposal is automatically converted to emergency status during the `AddDeposit` process

#### B. Manual Activation

- An authorized emergency submitter can call `MsgActivateEmergencyProposal`
- This explicitly converts a proposal in voting period to emergency status
- **All activation requirements must be met**: proposal in voting period, EmergencyMinDeposit threshold reached, and sender must be in EmergencySubmitters list

### 3. Emergency Proposal State

When activated, the proposal gets:

- `Emergency = true` flag
- `EmergencyStartTime` set to current block time
- `EmergencyNextTallyTime` set to EmergencyStartTime + EmergencyTallyInterval
- Added to `EmergencyProposalsQueue` for periodic processing

### 4. Periodic Tallying Process

Emergency proposals are processed differently from regular proposals:

#### A. Separate Processing Queue

- Emergency proposals are queued in `EmergencyProposalsQueue` with their next tally time
- They are processed separately from regular proposals in the ABCI EndBlocker

#### B. Continuous Tallying

- Every block, emergency proposals due for tallying are processed
- Tallying occurs every `EmergencyTallyInterval` (e.g., every 5 minutes by default)
- If quorum is reached, the proposal finishes immediately
- If quorum is not reached, the next tally time is scheduled

#### C. Early Termination

- If quorum and threshold conditions are met during any tally, the proposal passes/fails immediately
- The voting period does not need to complete - emergency proposals can finish early

#### D. Voting Period End

- If the voting period ends without quorum being reached, the proposal fails
- The final tally is performed at the voting end time

## Configuration

The following parameters control emergency proposals:

- **EmergencyMinDeposit**: Minimum deposit required to activate emergency status (must be >= ExpeditedMinDeposit)
- **EmergencySubmitters**: List of addresses authorized to trigger emergency proposals
- **EmergencyTallyInterval**: Duration between periodic tallies (must be < VotingPeriod)

### Default Values

| Parameter | Default Value | Description |
|-----------|---------------|-------------|
| EmergencyMinDeposit | 100,000,000 tokens | 10x the regular minimum deposit |
| EmergencyTallyInterval | 5 minutes | Time between periodic tallies |
| EmergencySubmitters | [] (empty) | Must be configured via governance |

## Example Flow

1. A regular proposal is submitted
2. The proposal reaches minimum deposit and enters voting period
3. An authorized emergency submitter adds deposit meeting `EmergencyMinDeposit` OR calls `ActivateEmergencyProposal`
4. The proposal is converted to emergency status with:
   - `Emergency` = true
   - `EmergencyStartTime` = current time
   - `EmergencyNextTallyTime` = `EmergencyStartTime` + `EmergencyTallyInterval`
5. The proposal is added to `EmergencyProposalsQueue`
6. Every `EmergencyTallyInterval`, the proposal is tallied
7. If quorum is reached during any tally, the proposal passes/fails immediately
8. If quorum is not reached, the next tally is scheduled
9. This continues until quorum is reached or voting period ends

## Authorization Management

Emergency proposal submitters can be managed through governance with:

- **`MsgAddEmergencyProposalSubmitters`**: Add new authorized submitters
- **`MsgRemoveEmergencyProposalSubmitters`**: Remove existing submitters

This ensures the list of emergency submitters can be updated as needed through proper governance channels.

## Key Differences from Regular Proposals

| Aspect | Regular Proposal | Emergency Proposal |
|--------|------------------|-------------------|
| Tallying | Only at voting period end | Every EmergencyTallyInterval |
| Early Finish | No | Yes (if quorum reached) |
| Processing | ActiveProposalsQueue | EmergencyProposalsQueue |
| Deposit Requirement | MinDeposit | EmergencyMinDeposit (>= ExpeditedMinDeposit) |
| Authorization | Anyone | Only EmergencySubmitters |
| State Tracking | Standard fields | + Emergency, EmergencyStartTime, EmergencyNextTallyTime |
