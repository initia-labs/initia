package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/initia/x/gov/keeper"
	customtypes "github.com/initia-labs/initia/x/gov/types"
	movetypes "github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_isLowThresholdProposal(t *testing.T) {
	params := customtypes.DefaultParams()

	messages := []sdk.Msg{
		&movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "vip",
			FunctionName:  "register_snapshot",
		},
		&movetypes.MsgExecuteJSON{
			ModuleAddress: "0x1",
			ModuleName:    "vip",
			FunctionName:  "register_snapshot",
		},
	}
	proposal, err := customtypes.NewProposal(messages, 1, time.Now().UTC(), time.Now().UTC().Add(time.Hour), "", "", "", addrs[0], true)
	require.NoError(t, err)
	require.True(t, keeper.IsLowThresholdProposal(params, proposal))

	messages = []sdk.Msg{
		&movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "vip",
			FunctionName:  "register_snapshot",
		},
		&movetypes.MsgScript{},
	}
	proposal, err = customtypes.NewProposal(messages, 1, time.Now().UTC(), time.Now().UTC().Add(time.Hour), "", "", "", addrs[0], true)
	require.NoError(t, err)
	require.False(t, keeper.IsLowThresholdProposal(params, proposal))
}

func setupVesting(t *testing.T, ctx sdk.Context, input TestKeepers) {
	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.TestAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(vestingModule)), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	creatorAccAddr := addrs[0]
	creatorAddr, err := vmtypes.NewAccountAddressFromBytes(creatorAccAddr)
	require.NoError(t, err)

	// create vesting table
	moduleAddr := vmtypes.TestAddress
	moduleName := "Vesting"

	metadata, err := movetypes.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(ctx, creatorAddr, moduleAddr, moduleName, "create_vesting_store", []vmtypes.TypeTag{}, []string{fmt.Sprintf("\"%s\"", metadata)})
	require.NoError(t, err)

	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)

	// add vesting
	recipientAccAddr := addrs[1]
	recipientAddr, err := vmtypes.NewAccountAddressFromBytes(recipientAccAddr)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(ctx, creatorAddr, moduleAddr, moduleName, "add_vesting", []vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", recipientAddr), // recipient
			fmt.Sprintf("\"%d\"", 6_000_000),     // allocation
			fmt.Sprintf("\"%d\"", now.Unix()),    // start_time
			fmt.Sprintf("\"%d\"", 3600),          // vesting_period
			fmt.Sprintf("\"%d\"", 1800),          // cliff_period
			fmt.Sprintf("\"%d\"", 60),            // cliff_frequency
		},
	)
	require.NoError(t, err)

	// update vesting params
	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	params.Vesting = &customtypes.Vesting{
		ModuleAddr:  movetypes.TestAddr.String(),
		ModuleName:  "Vesting",
		CreatorAddr: creatorAccAddr.String(),
	}

	err = input.GovKeeper.Params.Set(ctx, params)
	require.NoError(t, err)
}

func Test_Tally(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	setupVesting(t, ctx, input)

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "test", "description", addrs[0], false)
	require.NoError(t, err)

	proposalID := proposal.Id
	proposal.Status = v1.StatusVotingPeriod
	err = input.GovKeeper.SetProposal(ctx, proposal)
	require.NoError(t, err)

	proposal, err = input.GovKeeper.Proposals.Get(ctx, proposalID)
	require.NoError(t, err)

	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)

	quorumReached, passed, _, _, err := input.GovKeeper.Tally(ctx, params, proposal)
	require.NoError(t, err)
	require.False(t, quorumReached)
	require.False(t, passed)

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		1,
	)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		2,
	)

	voterAddr1 := sdk.AccAddress(valAddr1)
	voterAddr2 := sdk.AccAddress(valAddr2)

	// vote yes
	err = input.GovKeeper.AddVote(ctx, proposalID, voterAddr1, v1.WeightedVoteOptions{
		{
			Option: v1.OptionYes,
			Weight: "1",
		},
	}, "")
	require.NoError(t, err)

	// vote no
	err = input.GovKeeper.AddVote(ctx, proposalID, voterAddr2, v1.WeightedVoteOptions{
		{
			Option: v1.OptionNo,
			Weight: "1",
		},
	}, "")
	require.NoError(t, err)

	// add vesting vote
	vestingVoter := addrs[1]
	err = input.GovKeeper.AddVote(ctx, proposalID, vestingVoter, v1.WeightedVoteOptions{
		{
			Option: v1.OptionYes,
			Weight: "1",
		},
	}, "")
	require.NoError(t, err)

	// 15 minutes passed
	ctx = ctx.WithBlockTime(time.Now().Add(time.Minute * 15))

	quorumReached, passed, burnDeposits, tallyResults, err := input.GovKeeper.Tally(ctx, params, proposal)
	require.NoError(t, err)
	require.True(t, quorumReached)
	require.True(t, passed)
	require.False(t, burnDeposits)
	require.Equal(t, tallyResults.YesCount, math.LegacyNewDec(1_500_000+100_000_000).TruncateInt().String())
	require.Equal(t, tallyResults.NoCount, math.LegacyNewDec(100_000_000).TruncateInt().String())
}
