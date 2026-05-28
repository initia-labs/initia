package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/initia/x/move/types"
)

func TestExecuteEntryFunctionWithFeePayer(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.MoveKeeper.PublishModuleBundle(
		ctx,
		vmtypes.TestAddress,
		vmtypes.NewModuleBundle(vmtypes.NewModule(txContextTestsModule)),
		types.UpgradePolicy_COMPATIBLE,
	)
	require.NoError(t, err)

	feePayer, err := vmtypes.NewAccountAddress("0xcafe")
	require.NoError(t, err)
	ctx = ctx.WithValue(types.FeePayerContextKey, &feePayer)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		vmtypes.TestAddress,
		vmtypes.TestAddress,
		"TxContextTests",
		"store_fee_payer",
		[]vmtypes.TypeTag{},
		nil,
	)
	require.NoError(t, err)

	res, _, err := input.MoveKeeper.ExecuteViewFunctionJSON(
		ctx,
		vmtypes.TestAddress,
		"TxContextTests",
		"read_stored_fee_payer",
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("\"%s\"", vmtypes.TestAddress)},
	)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("\"%s\"", feePayer), res.Ret)
}
