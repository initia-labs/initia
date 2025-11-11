package v1_2_0

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	initparams "github.com/initia-labs/initia/app/params"

	connecttypes "github.com/skip-mev/connect/v2/pkg/types"
	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
)

func setupMarketMapKeeper(t *testing.T) (sdk.Context, *marketmapkeeper.Keeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(marketmaptypes.StoreKey)
	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{
			marketmaptypes.StoreKey: storeKey,
		},
		map[string]*storetypes.TransientStoreKey{},
		map[string]*storetypes.MemoryStoreKey{},
	)

	encCfg := initparams.MakeEncodingConfig()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := marketmapkeeper.NewKeeper(runtime.NewKVStoreService(storeKey), encCfg.Codec, authority)

	params := marketmaptypes.DefaultParams()
	params.MarketAuthorities = []string{authority.String()}
	params.Admin = authority.String()

	require.NoError(t, k.SetParams(ctx, params))

	return ctx, k
}

func makeUSDCUSDMarket(providers ...string) marketmaptypes.Market {
	cfgs := make([]marketmaptypes.ProviderConfig, 0, len(providers))
	for _, provider := range providers {
		cfgs = append(cfgs, marketmaptypes.ProviderConfig{
			Name:           provider,
			OffChainTicker: "USDC/USD",
		})
	}

	return marketmaptypes.Market{
		Ticker: marketmaptypes.Ticker{
			CurrencyPair: connecttypes.CurrencyPair{
				Base:  "USDC",
				Quote: "USD",
			},
			Decimals:         6,
			MinProviderCount: 1,
		},
		ProviderConfigs: cfgs,
	}
}

func TestUpdateMarketMap_RemovesKraken(t *testing.T) {
	ctx, keeper := setupMarketMapKeeper(t)
	require.NoError(t, keeper.CreateMarket(ctx, makeUSDCUSDMarket("kraken_api", "coinbase_api")))

	err := updateMarketMap(ctx, keeper)
	require.NoError(t, err)

	updated, err := keeper.GetMarket(ctx, "USDC/USD")
	require.NoError(t, err)
	require.Len(t, updated.ProviderConfigs, 1)
	require.Equal(t, "coinbase_api", updated.ProviderConfigs[0].Name)
}

func TestUpdateMarketMap_NoOpCases(t *testing.T) {
	t.Run("missing market", func(t *testing.T) {
		ctx, keeper := setupMarketMapKeeper(t)

		err := updateMarketMap(ctx, keeper)
		require.NoError(t, err)
	})

	t.Run("provider already removed", func(t *testing.T) {
		ctx, keeper := setupMarketMapKeeper(t)
		require.NoError(t, keeper.CreateMarket(ctx, makeUSDCUSDMarket("coinbase_api")))

		err := updateMarketMap(ctx, keeper)
		require.NoError(t, err)

		unchanged, err := keeper.GetMarket(ctx, "USDC/USD")
		require.NoError(t, err)
		require.Len(t, unchanged.ProviderConfigs, 1)
		require.Equal(t, "coinbase_api", unchanged.ProviderConfigs[0].Name)
	})
}
