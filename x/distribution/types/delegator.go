package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// create a new DelegatorStartingInfo
func NewDelegatorStartingInfo(previousPeriod uint64, stakes sdk.DecCoins, height uint64) DelegatorStartingInfo {
	return DelegatorStartingInfo{
		PreviousPeriod: previousPeriod,
		Stakes:         stakes,
		Height:         height,
	}
}

// create a new DelegationDelegatorReward
func NewDelegationDelegatorReward(valAddr sdk.ValAddress, rewards DecPools) DelegationDelegatorReward {
	return DelegationDelegatorReward{
		ValidatorAddress: valAddr.String(),
		Reward:           rewards,
	}
}
