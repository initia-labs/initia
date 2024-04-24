package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/move/keeper"

	"github.com/stretchr/testify/require"
)

func Test_NftCreateOrUpdateClass(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)
	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	collectionDesc := `{"name": "name", "symbol": "symbol", "description": "desecsecsdcsecsdcecddesc"}`
	err := nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", collectionDesc)
	require.NoError(t, err)

	collectionUri, _collectionData, err := nftKeeper.GetClassInfo(ctx, ibcClassId)
	require.NoError(t, err)
	require.Equal(t, "uri", collectionUri)
	require.Equal(t, collectionDesc, _collectionData)
}

func Test_NftMint(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)

	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	collectionDesc := `{"name": "name", "symbol": "symbol", "description": "desecsecsdcsecsdcecddesc"}`
	err := nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", collectionDesc)
	require.NoError(t, err)

	token1Desc, token2Desc := "desc1", "desc2"
	_, _, receiver := keyPubAddr()
	err = nftKeeper.Mints(
		ctx, receiver, ibcClassId,
		[]string{"token1", "token2"},
		[]string{"uri1", "uri2"},
		[]string{token1Desc, token2Desc},
	)
	require.NoError(t, err)

	tokenUris, tokenDescs, err := nftKeeper.GetTokenInfos(ctx, ibcClassId, []string{"token1", "token2"})
	require.NoError(t, err)
	require.Equal(t, []string{"uri1", "uri2"}, tokenUris)
	require.Equal(t, []string{token1Desc, token2Desc}, tokenDescs)
}

func Test_NftBurn(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)

	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	collectionDesc := `{"name": "name", "symbol": "symbol", "description": "desecsecsdcsecsdcecddesc"}`
	err := nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", collectionDesc)
	require.NoError(t, err)

	token1Desc, token2Desc := "desc1", "desc2"

	_, _, receiver := keyPubAddr()
	err = nftKeeper.Mints(
		ctx, receiver, ibcClassId,
		[]string{"token1", "token2"},
		[]string{"uri1", "uri2"},
		[]string{token1Desc, token2Desc},
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
	require.Equal(t, []string{token2Desc}, data)
}

func Test_NftTransfer(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	nftKeeper := keeper.NewNftKeeper(&input.MoveKeeper)

	ibcClassId := "ibc/09120912091209120912091209120912091209120912091209120912"
	collectionDesc := `{"name": "name", "symbol": "symbol", "description": "desecsecsdcsecsdcecddesc"}`
	err := nftKeeper.CreateOrUpdateClass(ctx, ibcClassId, "uri", collectionDesc)
	require.NoError(t, err)

	token1Desc, token2Desc := "desc1", "desc2"

	_, _, sender := keyPubAddr()
	err = nftKeeper.Mints(
		ctx, sender, ibcClassId,
		[]string{"token1", "token2"},
		[]string{"uri1", "uri2"},
		[]string{token1Desc, token2Desc},
	)
	require.NoError(t, err)

	_, _, receiver := keyPubAddr()
	err = nftKeeper.Transfers(ctx, sender, receiver, ibcClassId, []string{"token1", "token2"})
	require.NoError(t, err)
}
