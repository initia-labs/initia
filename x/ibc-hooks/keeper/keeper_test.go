package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
)

func TestTransferFundsCRUD(t *testing.T) {
	ctx, keepers := createDefaultTestInput(t)
	keeper := keepers.IBCHooksKeeper

	transferFunds := ibchookstypes.TransferFunds{
		BalanceChange:  sdk.NewCoin(bondDenom, math.NewInt(10)),
		AmountInPacket: sdk.NewCoin("test1", math.NewInt(25)),
	}

	_, err := keeper.GetTransferFunds(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)

	err = keeper.SetTransferFunds(ctx, transferFunds)
	require.NoError(t, err)

	got, err := keeper.GetTransferFunds(ctx)
	require.NoError(t, err)
	require.Equal(t, transferFunds, got)

	err = keeper.RemoveTransferFunds(ctx)
	require.NoError(t, err)

	_, err = keeper.GetTransferFunds(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestAsyncCallbackCRUD(t *testing.T) {
	ctx, keepers := createDefaultTestInput(t)
	keeper := keepers.IBCHooksKeeper

	sourcePort := "transfer"
	sourceChannel := "channel-0"
	packetID := uint64(1)
	callbackData := []byte("callback-data")

	_, err := keeper.GetAsyncCallback(ctx, sourcePort, sourceChannel, packetID)
	require.ErrorIs(t, err, collections.ErrNotFound)

	err = keeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, packetID, callbackData)
	require.NoError(t, err)

	got, err := keeper.GetAsyncCallback(ctx, sourcePort, sourceChannel, packetID)
	require.NoError(t, err)
	require.Equal(t, callbackData, got)

	err = keeper.RemoveAsyncCallback(ctx, sourcePort, sourceChannel, packetID)
	require.NoError(t, err)

	_, err = keeper.GetAsyncCallback(ctx, sourcePort, sourceChannel, packetID)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
