package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/v1/x/move/keeper"
	"github.com/initia-labs/initia/v1/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_Vesting(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	vestingKeeper := keeper.NewVestingKeeper(&input.MoveKeeper)

	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.TestAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(vestingModule)), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	creatorAccAddr := addrs[0]
	creatorAddr, err := vmtypes.NewAccountAddressFromBytes(creatorAccAddr)
	require.NoError(t, err)

	// create vesting table
	moduleAddr := vmtypes.TestAddress
	moduleName := "Vesting"

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(ctx, creatorAddr, moduleAddr, moduleName, "create_vesting_store", []vmtypes.TypeTag{}, []string{fmt.Sprintf("\"%s\"", metadata)})
	require.NoError(t, err)

	// get vesting table handle
	tableHandle, err := vestingKeeper.GetVestingHandle(ctx, moduleAddr[:], moduleName, creatorAddr[:])
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
			fmt.Sprintf("\"%d\"", 6000000),       // allocation
			fmt.Sprintf("\"%d\"", now.Unix()),    // start_time
			fmt.Sprintf("\"%d\"", 3600),          // vesting_period
			fmt.Sprintf("\"%d\"", 1800),          // cliff_period
			fmt.Sprintf("\"%d\"", 60),            // cliff_frequency
		},
	)
	require.NoError(t, err)

	// funding
	input.Faucet.Fund(ctx, creatorAccAddr, sdk.NewCoin(bondDenom, math.NewInt(10000000)))
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(ctx, creatorAddr, moduleAddr, moduleName, "fund_vesting", []vmtypes.TypeTag{}, []string{fmt.Sprintf("\"%s\"", creatorAddr), fmt.Sprintf("\"%d\"", 6000000)})
	require.NoError(t, err)

	// set time to half passed the cliff period
	ctx = ctx.WithBlockTime(now.Add(time.Minute * 15))

	// get unclaimed vested
	amount, err := vestingKeeper.GetUnclaimedVestedAmount(ctx, *tableHandle, recipientAccAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(1500000), amount.Uint64())

	// claim vested
	ctx = ctx.WithBlockTime(now.Add(time.Minute * 31))
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(ctx, recipientAddr, moduleAddr, moduleName, "claim_script", []vmtypes.TypeTag{}, []string{fmt.Sprintf("\"%s\"", creatorAddr)})
	require.NoError(t, err)

	// get unclaimed vested
	amount, err = vestingKeeper.GetUnclaimedVestedAmount(ctx, *tableHandle, recipientAccAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(0), amount.Uint64())
}
