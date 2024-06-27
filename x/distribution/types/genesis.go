package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

//nolint:interfacer
func NewGenesisState(
	params Params, fp distrtypes.FeePool, dwis []distrtypes.DelegatorWithdrawInfo, pp sdk.ConsAddress, r []ValidatorOutstandingRewardsRecord,
	acc []ValidatorAccumulatedCommissionRecord, historical []ValidatorHistoricalRewardsRecord,
	cur []ValidatorCurrentRewardsRecord, dels []DelegatorStartingInfoRecord, slashes []ValidatorSlashEventRecord,
) *GenesisState {

	return &GenesisState{
		Params:                          params,
		FeePool:                         fp,
		DelegatorWithdrawInfos:          dwis,
		PreviousProposer:                pp.String(),
		OutstandingRewards:              r,
		ValidatorAccumulatedCommissions: acc,
		ValidatorHistoricalRewards:      historical,
		ValidatorCurrentRewards:         cur,
		DelegatorStartingInfos:          dels,
		ValidatorSlashEvents:            slashes,
	}
}

// DefaultGenesisState returns raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		FeePool:                         distrtypes.InitialFeePool(),
		Params:                          DefaultParams(),
		DelegatorWithdrawInfos:          []distrtypes.DelegatorWithdrawInfo{},
		PreviousProposer:                "",
		OutstandingRewards:              []ValidatorOutstandingRewardsRecord{},
		ValidatorAccumulatedCommissions: []ValidatorAccumulatedCommissionRecord{},
		ValidatorHistoricalRewards:      []ValidatorHistoricalRewardsRecord{},
		ValidatorCurrentRewards:         []ValidatorCurrentRewardsRecord{},
		DelegatorStartingInfos:          []DelegatorStartingInfoRecord{},
		ValidatorSlashEvents:            []ValidatorSlashEventRecord{},
	}
}

// ValidateGenesis validates the genesis state of distribution genesis input
func ValidateGenesis(gs *GenesisState) error {
	if err := gs.Params.ValidateBasic(); err != nil {
		return err
	}
	return gs.FeePool.ValidateGenesis()
}
