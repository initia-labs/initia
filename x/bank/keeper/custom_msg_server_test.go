package keeper_test

import (
	"testing"

	cosmosbanktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/initia-labs/initia/x/bank/keeper"
	"github.com/initia-labs/initia/x/bank/types"
	"github.com/stretchr/testify/require"
)

func Test_SetDenomMetadata(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	metadata := cosmosbanktypes.Metadata{
		Description: "hihi",
		Base:        "ufoo",
		Display:     "foo",
		DenomUnits: []*cosmosbanktypes.DenomUnit{
			{
				Denom:    "ufoo",
				Exponent: 0,
			},
			{
				Denom:    "foo",
				Exponent: 6,
			},
		},
		Name:   "foo coin",
		Symbol: "foo",
	}
	_, err := keeper.NewCustomMsgServerImpl(input.BankKeeper).SetDenomMetadata(ctx, &types.MsgSetDenomMetadata{
		Authority: input.BankKeeper.GetAuthority(),
		Metadata:  metadata,
	})
	require.NoError(t, err)

	_metadata, found := input.BankKeeper.GetDenomMetaData(ctx, "ufoo")
	require.True(t, found)
	require.Equal(t, metadata, _metadata)
}
