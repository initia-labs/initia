package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/gov/types"
)

func Test_Params(t *testing.T) {
	params := types.DefaultParams()
	require.NoError(t, params.ValidateBasic())

	_emergencyMinDeposit := params.EmergencyMinDeposit
	params.EmergencyMinDeposit = []sdk.Coin{{
		Denom:  "foo",
		Amount: sdk.NewInt(-1),
	}}
	require.Error(t, params.ValidateBasic())

	params.EmergencyMinDeposit = _emergencyMinDeposit

	_minDeposit := params.MinDeposit
	params.MinDeposit = []sdk.Coin{{
		Denom:  "foo",
		Amount: sdk.NewInt(-1),
	}}
	require.Error(t, params.ValidateBasic())

	params.MinDeposit = _minDeposit

	_maxDepositPeriod := params.MaxDepositPeriod
	params.MaxDepositPeriod = 0
	require.Error(t, params.ValidateBasic())

	params.MaxDepositPeriod = _maxDepositPeriod

	_minInitialDepositRatio := params.MinInitialDepositRatio
	params.MinInitialDepositRatio = "-1"
	require.Error(t, params.ValidateBasic())

	params.MinInitialDepositRatio = _minInitialDepositRatio

	_quorum := params.Quorum
	params.Quorum = "-1"
	require.Error(t, params.ValidateBasic())

	params.Quorum = _quorum

	_threshold := params.Threshold
	params.Threshold = "-1"
	require.Error(t, params.ValidateBasic())

	params.Threshold = _threshold

	_vetoThreshold := params.VetoThreshold
	params.VetoThreshold = "-1"
	require.Error(t, params.ValidateBasic())

	params.VetoThreshold = _vetoThreshold

	_votingPeriod := params.VotingPeriod
	params.VotingPeriod = 0
	require.Error(t, params.ValidateBasic())

	params.VotingPeriod = _votingPeriod
	require.NoError(t, params.ValidateBasic())
}
