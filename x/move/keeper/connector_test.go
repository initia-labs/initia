package keeper_test

import (
	"testing"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func TestInitializeIBCCoin(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

	denomTrace := transfertypes.DenomTrace{
		Path:      "",
		BaseDenom: "ufoo",
	}
	denom := denomTrace.IBCDenom()

	err := moveBankKeeper.InitializeCoin(ctx, denom)
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(denom)
	require.NoError(t, err)

	bz, err := input.MoveKeeper.GetResourceBytes(ctx, metadata, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameFungibleAsset,
		Name:     types.ResourceNameMetadata,
		TypeArgs: []vmtypes.TypeTag{},
	})
	require.NoError(t, err)

	symbol := types.ReadSymbolFromMetadata(bz)
	require.Equal(t, denom, symbol)
}
