package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/initia/x/gov/types"
)

func TestEmptyGenesis(t *testing.T) {
	state1 := types.GenesisState{}
	require.True(t, state1.Empty())

	state2 := types.DefaultGenesisState()
	require.False(t, state2.Empty())
}

func TestValidateGenesis(t *testing.T) {
	params := types.DefaultParams()

	testCases := []struct {
		name         string
		genesisState func() *types.GenesisState
		expErr       bool
	}{
		{
			name: "valid",
			genesisState: func() *types.GenesisState {
				return types.NewGenesisState(v1.DefaultStartingProposalID, params)
			},
		},
		{
			name: "invalid StartingProposalId",
			genesisState: func() *types.GenesisState {
				return types.NewGenesisState(0, params)
			},
			expErr: true,
		},
		{
			name: "invalid min deposit",
			genesisState: func() *types.GenesisState {
				params1 := params
				params1.MinDeposit = sdk.Coins{{
					Denom:  sdk.DefaultBondDenom,
					Amount: math.NewInt(-100),
				}}

				return types.NewGenesisState(0, params1)
			},
			expErr: true,
		},
		{
			name: "invalid max deposit period",
			genesisState: func() *types.GenesisState {
				params1 := params
				params1.MaxDepositPeriod = 0

				return types.NewGenesisState(0, params1)
			},
			expErr: true,
		},
		{
			name: "invalid quorum",
			genesisState: func() *types.GenesisState {
				params1 := params
				params1.Quorum = "2"

				return types.NewGenesisState(0, params1)
			},
			expErr: true,
		},
		{
			name: "invalid threshold",
			genesisState: func() *types.GenesisState {
				params1 := params
				params1.Threshold = "2"

				return types.NewGenesisState(0, params1)
			},
			expErr: true,
		},
		{
			name: "invalid veto threshold",
			genesisState: func() *types.GenesisState {
				params1 := params
				params1.VetoThreshold = "2"

				return types.NewGenesisState(0, params1)
			},
			expErr: true,
		},
	}

	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateGenesis(tc.genesisState(), ac)
			if tc.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
