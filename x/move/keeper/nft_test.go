package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"

	"github.com/stretchr/testify/require"
)

func Test_NftCreateOrUpdateClass(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)
	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	desc := `{"name": "name", "symbol": "symbol", "dummy": "dummy"}`
	data, err := types.ConvertDescriptionToICS721Data(desc)
	require.NoError(t, err)
	err = nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", data)
	require.NoError(t, err)

	uri, _data, err := nftKeeper.GetClassInfo(ctx, ibcClassId)
	require.NoError(t, err)
	require.Equal(t, "uri", uri)
	require.Equal(t, data, _data)

	// decode data
	_desc, err := types.ConvertICS721DataToDescription(_data)
	require.NoError(t, err)
	require.Equal(t, desc, _desc)
}

func Test_NftMint(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)

	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	desc := `{"name": "name", "symbol": "symbol", "dummy": "dummy"}`
	classData, err := types.ConvertDescriptionToICS721Data(desc)
	require.NoError(t, err)

	err = nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", classData)
	require.NoError(t, err)

	token1Desc, token2Desc := "desc1", "desc2"
	token1Data, err := types.ConvertDescriptionToICS721Data(token1Desc)
	require.NoError(t, err)
	token2Data, err := types.ConvertDescriptionToICS721Data(token2Desc)
	require.NoError(t, err)

	_, _, receiver := keyPubAddr()
	err = nftKeeper.Mints(
		ctx, receiver, ibcClassId,
		[]string{"token1", "token2"},
		[]string{"uri1", "uri2"},
		[]string{token1Data, token2Data},
	)
	require.NoError(t, err)

	uris, data, err := nftKeeper.GetTokenInfos(ctx, ibcClassId, []string{"token1", "token2"})
	require.NoError(t, err)
	require.Equal(t, []string{"uri1", "uri2"}, uris)
	require.Equal(t, []string{token1Data, token2Data}, data)
}

func Test_NftBurn(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)

	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	desc := `{"name": "name", "symbol": "symbol", "dummy": "dummy"}`
	classData, err := types.ConvertDescriptionToICS721Data(desc)
	require.NoError(t, err)

	err = nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", classData)
	require.NoError(t, err)

	token1Desc, token2Desc := "desc1", "desc2"
	token1Data, err := types.ConvertDescriptionToICS721Data(token1Desc)
	require.NoError(t, err)
	token2Data, err := types.ConvertDescriptionToICS721Data(token2Desc)
	require.NoError(t, err)

	_, _, receiver := keyPubAddr()
	err = nftKeeper.Mints(
		ctx, receiver, ibcClassId,
		[]string{"token1", "token2"},
		[]string{"uri1", "uri2"},
		[]string{token1Data, token2Data},
	)
	require.NoError(t, err)

	err = nftKeeper.Burns(ctx, receiver, ibcClassId, []string{"token1"})
	require.NoError(t, err)

	// should be deleted
	_, _, err = nftKeeper.GetTokenInfos(ctx, ibcClassId, []string{"token1"})
	require.Error(t, err)

	uris, data, err := nftKeeper.GetTokenInfos(ctx, ibcClassId, []string{"token2"})
	require.NoError(t, err)
	require.Equal(t, []string{"uri2"}, uris)
	require.Equal(t, []string{token2Data}, data)
}

func Test_NftTransfer(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)

	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	desc := `{"name": "name", "symbol": "symbol", "dummy": "dummy"}`
	classData, err := types.ConvertDescriptionToICS721Data(desc)
	require.NoError(t, err)

	err = nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", classData)
	require.NoError(t, err)

	token1Desc, token2Desc := "desc1", "desc2"
	token1Data, err := types.ConvertDescriptionToICS721Data(token1Desc)
	require.NoError(t, err)
	token2Data, err := types.ConvertDescriptionToICS721Data(token2Desc)
	require.NoError(t, err)

	_, _, sender := keyPubAddr()
	err = nftKeeper.Mints(
		ctx, sender, ibcClassId,
		[]string{"token1", "token2"},
		[]string{"uri1", "uri2"},
		[]string{token1Data, token2Data},
	)
	require.NoError(t, err)

	_, _, receiver := keyPubAddr()
	err = nftKeeper.Transfers(ctx, sender, receiver, ibcClassId, []string{"token1", "token2"})
	require.NoError(t, err)
}
