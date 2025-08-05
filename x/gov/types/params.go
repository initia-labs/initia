package types

import (
	"fmt"
	"slices"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	"gopkg.in/yaml.v3"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// Default period for deposits & voting
const (
	DefaultPeriod                         time.Duration = time.Hour * 24 * 2 // 2 days
	DefaultEmergencyTallyInterval         time.Duration = time.Minute * 5    // 5 minutes
	DefaultExpeditedPeriod                time.Duration = time.Hour * 24 * 1 // 1 day
	DefaultMinExpeditedDepositTokensRatio               = 5
)

// Default governance params
var (
	DefaultMinDepositTokens          = math.NewInt(10000000)
	DefaultEmergencyMinDepositTokens = math.NewInt(100000000)
	DefaultMinExpeditedDepositTokens = DefaultMinDepositTokens.Mul(math.NewInt(DefaultMinExpeditedDepositTokensRatio))
	DefaultQuorum                    = math.LegacyNewDecWithPrec(334, 3)
	DefaultThreshold                 = math.LegacyNewDecWithPrec(5, 1)
	DefaultExpeditedThreshold        = math.LegacyNewDecWithPrec(667, 3)
	DefaultVetoThreshold             = math.LegacyNewDecWithPrec(334, 3)
	DefaultMinInitialDepositRatio    = math.LegacyZeroDec()
	DefaultProposalCancelRatio       = math.LegacyMustNewDecFromStr("0.5")
	DefaultProposalCancelDestAddress = ""
	DefaultBurnProposalPrevote       = false // set to false to replicate behavior of when this change was made (0.47)
	DefaultBurnVoteQuorom            = false // set to false to  replicate behavior of when this change was made (0.47)
	DefaultBurnVoteVeto              = true  // set to true to replicate behavior of when this change was made (0.47)
	DefaultMinDepositRatio           = math.LegacyMustNewDecFromStr("0.01")
	DefaultLowThresholdFunctions     = []string{"0x1::vip::register_snapshot"}
)

// NewParams creates a new Params instance with given values.
func NewParams(
	minDeposit, expeditedMinDeposit sdk.Coins, maxDepositPeriod, votingPeriod, expeditedVotingPeriod time.Duration,
	quorum, threshold, expeditedThreshold, vetoThreshold, minInitialDepositRatio, proposalCancelRatio, proposalCancelDest string,
	burnProposalDeposit, burnVoteQuorum, burnVoteVeto bool, minDepositRatio string,
	emergencyMinDeposit sdk.Coins, emergencyTallyInterval time.Duration, lowThresholdFunctions []string,
) Params {
	return Params{
		MinDeposit:                 minDeposit,
		ExpeditedMinDeposit:        expeditedMinDeposit,
		MaxDepositPeriod:           maxDepositPeriod,
		VotingPeriod:               votingPeriod,
		ExpeditedVotingPeriod:      expeditedVotingPeriod,
		Quorum:                     quorum,
		Threshold:                  threshold,
		ExpeditedThreshold:         expeditedThreshold,
		VetoThreshold:              vetoThreshold,
		MinInitialDepositRatio:     minInitialDepositRatio,
		ProposalCancelRatio:        proposalCancelRatio,
		ProposalCancelDest:         proposalCancelDest,
		BurnProposalDepositPrevote: burnProposalDeposit,
		BurnVoteQuorum:             burnVoteQuorum,
		BurnVoteVeto:               burnVoteVeto,
		MinDepositRatio:            minDepositRatio,
		EmergencyMinDeposit:        emergencyMinDeposit,
		EmergencyTallyInterval:     emergencyTallyInterval,
		LowThresholdFunctions:      lowThresholdFunctions,
	}
}

// DefaultParams returns the default governance params
func DefaultParams() Params {
	return NewParams(
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinDepositTokens)),
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinExpeditedDepositTokens)),
		DefaultPeriod,
		DefaultPeriod,
		DefaultExpeditedPeriod,
		DefaultQuorum.String(),
		DefaultThreshold.String(),
		DefaultExpeditedThreshold.String(),
		DefaultVetoThreshold.String(),
		DefaultMinInitialDepositRatio.String(),
		DefaultProposalCancelRatio.String(),
		DefaultProposalCancelDestAddress,
		DefaultBurnProposalPrevote,
		DefaultBurnVoteQuorom,
		DefaultBurnVoteVeto,
		DefaultMinDepositRatio.String(),
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultEmergencyMinDepositTokens)),
		DefaultEmergencyTallyInterval,
		DefaultLowThresholdFunctions,
	)
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// Validate performs basic validation on governance parameters.
func (p Params) Validate(ac address.Codec) error {
	minDeposit := sdk.Coins(p.MinDeposit)
	if minDeposit.Empty() || !minDeposit.IsValid() {
		return fmt.Errorf("invalid minimum deposit: %s", minDeposit)
	}

	if minExpeditedDeposit := sdk.Coins(p.ExpeditedMinDeposit); minExpeditedDeposit.Empty() || !minExpeditedDeposit.IsValid() {
		return fmt.Errorf("invalid expedited minimum deposit: %s", minExpeditedDeposit)
	} else if !minExpeditedDeposit.IsAllGT(minDeposit) {
		return fmt.Errorf("expedited minimum deposit must be greater than minimum deposit: %s", minExpeditedDeposit)
	}

	if p.MaxDepositPeriod.Seconds() <= 0 {
		return fmt.Errorf("maximum deposit period must be positive: %d", p.MaxDepositPeriod)
	}

	quorum, err := math.LegacyNewDecFromStr(p.Quorum)
	if err != nil {
		return fmt.Errorf("invalid quorum string: %w", err)
	}
	if quorum.IsNegative() {
		return fmt.Errorf("quorom cannot be negative: %s", quorum)
	}
	if quorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorom too large: %s", p.Quorum)
	}

	threshold, err := math.LegacyNewDecFromStr(p.Threshold)
	if err != nil {
		return fmt.Errorf("invalid threshold string: %w", err)
	}
	if !threshold.IsPositive() {
		return fmt.Errorf("vote threshold must be positive: %s", threshold)
	}
	if threshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("vote threshold too large: %s", threshold)
	}

	expeditedThreshold, err := math.LegacyNewDecFromStr(p.ExpeditedThreshold)
	if err != nil {
		return fmt.Errorf("invalid expedited threshold string: %w", err)
	}
	if !threshold.IsPositive() {
		return fmt.Errorf("expedited vote threshold must be positive: %s", threshold)
	}
	if threshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("expedited vote threshold too large: %s", threshold)
	}
	if expeditedThreshold.LTE(threshold) {
		return fmt.Errorf("expedited vote threshold %s, must be greater than the regular threshold %s", expeditedThreshold, threshold)
	}

	vetoThreshold, err := math.LegacyNewDecFromStr(p.VetoThreshold)
	if err != nil {
		return fmt.Errorf("invalid vetoThreshold string: %w", err)
	}
	if !vetoThreshold.IsPositive() {
		return fmt.Errorf("veto threshold must be positive: %s", vetoThreshold)
	}
	if vetoThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("veto threshold too large: %s", vetoThreshold)
	}

	if p.VotingPeriod.Seconds() <= 0 {
		return fmt.Errorf("voting period must be positive: %s", p.VotingPeriod)
	}

	if p.ExpeditedVotingPeriod.Seconds() <= 0 {
		return fmt.Errorf("expedited voting period must be positive: %s", p.ExpeditedVotingPeriod)
	}
	if p.ExpeditedVotingPeriod.Seconds() >= p.VotingPeriod.Seconds() {
		return fmt.Errorf("expedited voting period %s must be strictly less that the regular voting period %s", p.ExpeditedVotingPeriod, p.VotingPeriod)
	}

	minInitialDepositRatio, err := math.LegacyNewDecFromStr(p.MinInitialDepositRatio)
	if err != nil {
		return fmt.Errorf("invalid minimum initial deposit ratio of proposal: %w", err)
	}
	if minInitialDepositRatio.IsNegative() {
		return fmt.Errorf("minimum initial deposit ratio of proposal must be positive: %s", minInitialDepositRatio)
	}
	if minInitialDepositRatio.GT(math.LegacyOneDec()) {
		return fmt.Errorf("minimum initial deposit ratio of proposal is too large: %s", minInitialDepositRatio)
	}

	proposalCancelRate, err := math.LegacyNewDecFromStr(p.ProposalCancelRatio)
	if err != nil {
		return fmt.Errorf("invalid burn rate of cancel proposal: %w", err)
	}
	if proposalCancelRate.IsNegative() {
		return fmt.Errorf("burn rate of cancel proposal must be positive: %s", proposalCancelRate)
	}
	if proposalCancelRate.GT(math.LegacyOneDec()) {
		return fmt.Errorf("burn rate of cancel proposal is too large: %s", proposalCancelRate)
	}

	if len(p.ProposalCancelDest) != 0 {
		_, err := ac.StringToBytes(p.ProposalCancelDest)
		if err != nil {
			return fmt.Errorf("deposits destination address is invalid: %s", p.ProposalCancelDest)
		}
	}

	if emergencyMinDeposit := sdk.Coins(p.EmergencyMinDeposit); emergencyMinDeposit.Empty() || !emergencyMinDeposit.IsValid() {
		return fmt.Errorf("invalid emergency minimum deposit: %s", emergencyMinDeposit)
	}

	if p.EmergencyTallyInterval.Seconds() <= 0 {
		return fmt.Errorf("emergency tally interval must be positive: %s", p.EmergencyTallyInterval)
	}
	if p.EmergencyTallyInterval.Seconds() >= p.VotingPeriod.Seconds() {
		return fmt.Errorf("emergency tally interval %s must be strictly less than the voting period %s", p.EmergencyTallyInterval, p.VotingPeriod)
	}

	if minEmergencyDeposit := sdk.Coins(p.EmergencyMinDeposit); !minEmergencyDeposit.IsAllGTE(p.ExpeditedMinDeposit) {
		return fmt.Errorf("emergency minimum deposit must be greater than or equal to expedited minimum deposit")
	}

	return nil
}

func (p Params) ToV1() v1.Params {
	return v1.Params{
		MinDeposit:                 p.MinDeposit,
		MaxDepositPeriod:           &p.MaxDepositPeriod,
		VotingPeriod:               &p.VotingPeriod,
		Quorum:                     p.Quorum,
		Threshold:                  p.Threshold,
		VetoThreshold:              p.VetoThreshold,
		MinInitialDepositRatio:     p.MinInitialDepositRatio,
		ProposalCancelRatio:        p.ProposalCancelRatio,
		ProposalCancelDest:         p.ProposalCancelDest,
		ExpeditedVotingPeriod:      &p.ExpeditedVotingPeriod,
		ExpeditedThreshold:         p.ExpeditedThreshold,
		ExpeditedMinDeposit:        p.ExpeditedMinDeposit,
		BurnVoteQuorum:             p.BurnVoteQuorum,
		BurnProposalDepositPrevote: p.BurnProposalDepositPrevote,
		BurnVoteVeto:               p.BurnVoteVeto,
		MinDepositRatio:            p.MinDepositRatio,
	}
}

func (p Params) IsLowThresholdFunction(fid string) bool {
	return slices.Contains(p.LowThresholdFunctions, fid)
}
