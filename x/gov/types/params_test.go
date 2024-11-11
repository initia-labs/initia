package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/gov/types"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_Params(t *testing.T) {
	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	params := types.DefaultParams()
	require.NoError(t, params.Validate(ac))

	_emergencyMinDeposit := params.EmergencyMinDeposit
	params.EmergencyMinDeposit = []sdk.Coin{{
		Denom:  "foo",
		Amount: math.NewInt(-1),
	}}
	require.Error(t, params.Validate(ac))

	params.EmergencyMinDeposit = _emergencyMinDeposit

	_emergencyTallyInterval := params.EmergencyTallyInterval
	params.EmergencyTallyInterval = 0
	require.Error(t, params.Validate(ac))

	params.EmergencyTallyInterval = params.VotingPeriod
	require.Error(t, params.Validate(ac))

	params.EmergencyTallyInterval = _emergencyTallyInterval

	_minDeposit := params.MinDeposit
	params.MinDeposit = []sdk.Coin{{
		Denom:  "foo",
		Amount: math.NewInt(-1),
	}}
	require.Error(t, params.Validate(ac))

	params.MinDeposit = _minDeposit

	_maxDepositPeriod := params.MaxDepositPeriod
	params.MaxDepositPeriod = 0
	require.Error(t, params.Validate(ac))

	params.MaxDepositPeriod = _maxDepositPeriod

	_minInitialDepositRatio := params.MinInitialDepositRatio
	params.MinInitialDepositRatio = "a"
	require.Error(t, params.Validate(ac))
	params.MinInitialDepositRatio = "-1"
	require.Error(t, params.Validate(ac))
	params.MinInitialDepositRatio = "1.1"
	require.Error(t, params.Validate(ac))

	params.MinInitialDepositRatio = _minInitialDepositRatio

	_quorum := params.Quorum
	params.Quorum = "a"
	require.Error(t, params.Validate(ac))
	params.Quorum = "-1"
	require.Error(t, params.Validate(ac))
	params.Quorum = "1.1"
	require.Error(t, params.Validate(ac))

	params.Quorum = _quorum

	_threshold := params.Threshold
	params.Threshold = "a"
	require.Error(t, params.Validate(ac))
	params.Threshold = "-1"
	require.Error(t, params.Validate(ac))
	params.Threshold = "1.1"
	require.Error(t, params.Validate(ac))

	params.Threshold = _threshold

	_vetoThreshold := params.VetoThreshold
	params.VetoThreshold = "a"
	require.Error(t, params.Validate(ac))
	params.VetoThreshold = "-1"
	require.Error(t, params.Validate(ac))
	params.VetoThreshold = "1.1"
	require.Error(t, params.Validate(ac))

	params.VetoThreshold = _vetoThreshold

	_votingPeriod := params.VotingPeriod
	params.VotingPeriod = 0
	require.Error(t, params.Validate(ac))

	params.VotingPeriod = _votingPeriod
	require.NoError(t, params.Validate(ac))
}

func Test_Params_IsLowThresholdFunction(t *testing.T) {
	params := types.DefaultParams()
	require.False(t, params.IsLowThresholdFunction("foo"))

	params.LowThresholdFunctions = []string{"foo"}
	require.True(t, params.IsLowThresholdFunction("foo"))
}
