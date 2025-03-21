package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/dynamic-fee/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/math"
)

type Keeper struct {
	cdc          codec.Codec
	storeService corestoretypes.KVStoreService

	Schema collections.Schema

	Params collections.Item[types.Params]

	tokenPriceKeeper types.TokenPriceKeeper
	whitelistKeeper  types.WhitelistKeeper
	baseDenomKeeper  types.BaseDenomKeeper

	ac        address.Codec
	authority string
}

func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	tokenPriceKeeper types.TokenPriceKeeper,
	whitelistKeeper types.WhitelistKeeper,
	baseDenomKeeper types.BaseDenomKeeper,
	ac address.Codec,
	authority string,
) *Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:          cdc,
		storeService: storeService,

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),

		tokenPriceKeeper: tokenPriceKeeper,
		whitelistKeeper:  whitelistKeeper,
		baseDenomKeeper:  baseDenomKeeper,
		ac:               ac,
		authority:        authority,
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GetTokenPriceKeeper() types.TokenPriceKeeper {
	return k.tokenPriceKeeper
}

func (k Keeper) GetWhitelistKeeper() types.WhitelistKeeper {
	return k.whitelistKeeper
}

func (k Keeper) GetBaseDenomKeeper() types.BaseDenomKeeper {
	return k.baseDenomKeeper
}

func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.Params.Set(ctx, params)
}

func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	return k.Params.Get(ctx)
}

func (k Keeper) BaseGasPrice(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return params.BaseGasPrice, nil
}

// this should be called in EndBlocker
func (k Keeper) UpdateBaseGasPrice(ctx sdk.Context) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	gasUsed := ctx.BlockGasMeter().GasConsumed()

	// baseFeeMultiplier = (gasUsed - targetGas) / targetGas * maxChangeRate + 1
	baseFeeMultiplier := math.LegacyNewDec(int64(gasUsed) - params.TargetGas).QuoInt64(params.TargetGas).Mul(params.MaxChangeRate).Add(math.OneInt().ToLegacyDec())
	newBaseGasPrice := params.BaseGasPrice.Mul(baseFeeMultiplier)
	if newBaseGasPrice.LT(params.MinBaseGasPrice) {
		newBaseGasPrice = params.MinBaseGasPrice
	}
	if newBaseGasPrice.GT(params.MaxBaseGasPrice) {
		newBaseGasPrice = params.MaxBaseGasPrice
	}

	params.BaseGasPrice = newBaseGasPrice
	return k.SetParams(ctx, params)
}
