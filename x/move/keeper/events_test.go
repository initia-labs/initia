package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/stretchr/testify/require"
)

func Test_EnumEventEmission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	moduleBundle := vmtypes.NewModuleBundle(vmtypes.NewModule(counterModule))
	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, moduleBundle, types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err, "failed to publish Counter module bundle before testing event emission")

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		"Counter",
		"increase",
		nil,
		nil,
	)
	require.NoError(t, err)

	found := false
	for _, e := range ctx.EventManager().Events() {
		if e.Type == "move" && e.Attributes[0].Key == "type_tag" && e.Attributes[0].Value == "0x1::Counter::TestEvent" {
			found = true

			require.Equal(t, 2, len(e.Attributes))
			require.Equal(t, "data", e.Attributes[1].Key)
			require.Equal(t, `{"shape":{"Circle":{"radius":"1"}}}`, e.Attributes[1].Value)
		}
	}
	require.True(t, found, "TestEvent not found in emitted events")
}
