package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// create a new ValidatorHistoricalRewards
func NewValidatorHistoricalRewards(cumulativeRewardRatios DecPools, referenceCount uint32) ValidatorHistoricalRewards {
	return ValidatorHistoricalRewards{
		CumulativeRewardRatios: cumulativeRewardRatios,
		ReferenceCount:         referenceCount,
	}
}

// create a new ValidatorCurrentRewards
func NewValidatorCurrentRewards(rewards DecPools, period uint64) ValidatorCurrentRewards {
	return ValidatorCurrentRewards{
		Rewards: rewards,
		Period:  period,
	}
}

// return the initial accumulated commission (zero)
func InitialValidatorAccumulatedCommission() ValidatorAccumulatedCommission {
	return ValidatorAccumulatedCommission{}
}

// create a new ValidatorSlashEvent
func NewValidatorSlashEvent(validatorPeriod uint64, fractions sdk.DecCoins) ValidatorSlashEvent {
	return ValidatorSlashEvent{
		ValidatorPeriod: validatorPeriod,
		Fractions:       fractions,
	}
}

func (vs ValidatorSlashEvents) String() string {
	var out strings.Builder
	out.WriteString("Validator Slash Events:\n")
	for i, sl := range vs.ValidatorSlashEvents {
		out.WriteString(fmt.Sprintf(`  Slash %d:
    Period:    %d
    Fractions: %s
`, i, sl.ValidatorPeriod, sl.Fractions))
	}
	return strings.TrimSpace(out.String())
}
