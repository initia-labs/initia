package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	"gopkg.in/yaml.v3"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Default period for deposits & voting
const (
	DefaultPeriod                 time.Duration = time.Hour * 24 * 2 // 2 days
	DefaultEmergencyTallyInterval time.Duration = time.Minute * 5    // 5 minutes
)

// Default governance params
var (
	DefaultMinDepositTokens          = sdk.NewInt(10000000)
	DefaultEmergencyMinDepositTokens = sdk.NewInt(100000000)
	DefaultQuorum                    = sdk.NewDecWithPrec(334, 3)
	DefaultThreshold                 = sdk.NewDecWithPrec(5, 1)
	DefaultVetoThreshold             = sdk.NewDecWithPrec(334, 3)
	DefaultMinInitialDepositRatio    = sdk.ZeroDec()
	DefaultBurnProposalPrevote       = false // set to false to replicate behavior of when this change was made (0.47)
	DefaultBurnVoteQuorom            = false // set to false to  replicate behavior of when this change was made (0.47)
	DefaultBurnVoteVeto              = true  // set to true to replicate behavior of when this change was made (0.47)
)

// NewParams creates a new Params instance with given values.
func NewParams(
	minDeposit sdk.Coins, maxDepositPeriod, votingPeriod time.Duration,
	quorum, threshold, vetoThreshold, minInitialDepositRatio string,
	burnProposalDeposit, burnVoteQuorum, burnVoteVeto bool,
	emergencyMinDeposit sdk.Coins, emergencyTallyInterval time.Duration,
) Params {
	return Params{
		MinDeposit:                 minDeposit,
		MaxDepositPeriod:           maxDepositPeriod,
		VotingPeriod:               votingPeriod,
		Quorum:                     quorum,
		Threshold:                  threshold,
		VetoThreshold:              vetoThreshold,
		MinInitialDepositRatio:     minInitialDepositRatio,
		BurnProposalDepositPrevote: burnProposalDeposit,
		BurnVoteQuorum:             burnVoteQuorum,
		BurnVoteVeto:               burnVoteVeto,
		EmergencyMinDeposit:        emergencyMinDeposit,
		EmergencyTallyInterval:     emergencyTallyInterval,
	}
}

// DefaultParams returns the default governance params
func DefaultParams() Params {
	return NewParams(
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinDepositTokens)),
		DefaultPeriod,
		DefaultPeriod,
		DefaultQuorum.String(),
		DefaultThreshold.String(),
		DefaultVetoThreshold.String(),
		DefaultMinInitialDepositRatio.String(),
		DefaultBurnProposalPrevote,
		DefaultBurnVoteQuorom,
		DefaultBurnVoteVeto,
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultEmergencyMinDepositTokens)),
		DefaultEmergencyTallyInterval,
	)
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ValidateBasic performs basic validation on governance parameters.
func (p Params) ValidateBasic() error {
	if minDeposit := sdk.Coins(p.MinDeposit); minDeposit.Empty() || !minDeposit.IsValid() {
		return fmt.Errorf("invalid minimum deposit: %s", minDeposit)
	}

	if p.MaxDepositPeriod.Seconds() <= 0 {
		return fmt.Errorf("maximum deposit period must be positive: %d", p.MaxDepositPeriod)
	}

	quorum, err := sdk.NewDecFromStr(p.Quorum)
	if err != nil {
		return fmt.Errorf("invalid quorum string: %w", err)
	}
	if quorum.IsNegative() {
		return fmt.Errorf("quorom cannot be negative: %s", quorum)
	}
	if quorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorom too large: %s", p.Quorum)
	}

	threshold, err := sdk.NewDecFromStr(p.Threshold)
	if err != nil {
		return fmt.Errorf("invalid threshold string: %w", err)
	}
	if !threshold.IsPositive() {
		return fmt.Errorf("vote threshold must be positive: %s", threshold)
	}
	if threshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("vote threshold too large: %s", threshold)
	}

	vetoThreshold, err := sdk.NewDecFromStr(p.VetoThreshold)
	if err != nil {
		return fmt.Errorf("invalid vetoThreshold string: %w", err)
	}
	if !vetoThreshold.IsPositive() {
		return fmt.Errorf("veto threshold must be positive: %s", vetoThreshold)
	}
	if vetoThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("veto threshold too large: %s", vetoThreshold)
	}

	minInitialDepositRatio, err := sdk.NewDecFromStr(p.MinInitialDepositRatio)
	if err != nil {
		return fmt.Errorf("invalid minInitialDepositRatio string: %w", err)
	}
	if minInitialDepositRatio.IsNegative() {
		return fmt.Errorf("min initial deposit ratio must be zero or positive: %s", minInitialDepositRatio)
	}
	if minInitialDepositRatio.GT(math.LegacyOneDec()) {
		return fmt.Errorf("min initial deposit ratio too large: %s", minInitialDepositRatio)
	}

	if p.VotingPeriod.Seconds() <= 0 {
		return fmt.Errorf("voting period must be positive: %s", p.VotingPeriod)
	}

	if emergencyMinDeposit := sdk.Coins(p.EmergencyMinDeposit); emergencyMinDeposit.Empty() || !emergencyMinDeposit.IsValid() {
		return fmt.Errorf("invalid emergency minimum deposit: %s", emergencyMinDeposit)
	}

	if p.EmergencyTallyInterval.Seconds() <= 0 {
		return fmt.Errorf("emergency tally interval must be positive: %s", p.EmergencyTallyInterval)
	}

	if !sdk.Coins(p.EmergencyMinDeposit).IsAllGTE(p.MinDeposit) {
		return fmt.Errorf("emergency minimum deposit must be greater than or equal to minimum deposit")
	}

	return nil
}
