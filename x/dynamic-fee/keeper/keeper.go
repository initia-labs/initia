package keeper

import (
	"context"
	"errors"

	"github.com/initia-labs/initia/x/dynamic-fee/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/math"
)

type Keeper struct {
	cdc              codec.Codec
	storeService     corestoretypes.KVStoreService
	transientService corestoretypes.TransientStoreService

	Schema          collections.Schema
	TransientSchema collections.Schema

	Params         collections.Item[types.Params]
	AccumulatedGas collections.Item[uint64]

	tokenPriceKeeper types.TokenPriceKeeper
	whitelistKeeper  types.WhitelistKeeper
	baseDenomKeeper  types.BaseDenomKeeper

	ac        address.Codec
	authority string
}

func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	transientService corestoretypes.TransientStoreService,
	tokenPriceKeeper types.TokenPriceKeeper,
	whitelistKeeper types.WhitelistKeeper,
	baseDenomKeeper types.BaseDenomKeeper,
	ac address.Codec,
	authority string,
) *Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	tsb := collections.NewSchemaBuilderFromAccessor(transientService.OpenTransientStore)
	k := &Keeper{
		cdc:              cdc,
		storeService:     storeService,
		transientService: transientService,

		Params:         collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		AccumulatedGas: collections.NewItem(tsb, types.AccumulatedGasKey, "accumulated_gas", collections.Uint64Value),

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

	tSchema, err := tsb.Build()
	if err != nil {
		panic(err)
	}
	k.TransientSchema = tSchema

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
	if params.TargetGas == 0 {
		return types.ErrTargetGasZero
	}

	accumulatedGas, err := k.GetAccumulatedGas(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		accumulatedGas = 0
	} else if err != nil {
		return err
	}

	// baseFeeMultiplier = (accumulatedGas - targetGas) / targetGas * maxChangeRate + 1
	baseFeeMultiplier := math.LegacyNewDec(int64(accumulatedGas) - params.TargetGas).
		QuoInt64(params.TargetGas).
		Mul(params.MaxChangeRate).
		Add(math.LegacyOneDec())
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

// AccumulateGas accumulates the gas used in the block
func (k Keeper) AccumulateGas(ctx context.Context, gas uint64) error {
	accumulatedGas, err := k.AccumulatedGas.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		accumulatedGas = 0
	} else if err != nil {
		return err
	}

	accumulatedGas += gas
	return k.AccumulatedGas.Set(ctx, accumulatedGas)
}

// GetAccumulatedGas returns the accumulated gas
func (k Keeper) GetAccumulatedGas(ctx context.Context) (uint64, error) {
	return k.AccumulatedGas.Get(ctx)
}

// ResetAccumulatedGas resets the accumulated gas for testing
func (k Keeper) ResetAccumulatedGas(ctx context.Context) error {
	return k.AccumulatedGas.Remove(ctx)
}
