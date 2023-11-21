package types

import (
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// StakingKeeper defines the staking module interface contract needed by the
	// evidence module.
	StakingKeeper interface {
		ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI
	}
)
