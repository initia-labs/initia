package keeper

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/reward/types"
)

// Keeper of the reward store
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService corestoretypes.KVStoreService

	accKeeper        types.AccountKeeper
	bankKeeper       types.BankKeeper
	feeCollectorName string

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string

	Schema                collections.Schema
	Params                collections.Item[types.Params]
	LastReleaseTimestamp  collections.Item[time.Time]
	LastDilutionTimestamp collections.Item[time.Time]
}

// NewKeeper creates a new reward Keeper instance
func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	feeCollectorName, authority string,
) *Keeper {
	// ensure reward module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the reward module account has not been set")
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := &Keeper{
		cdc:              cdc,
		storeService:     storeService,
		accKeeper:        ak,
		bankKeeper:       bk,
		feeCollectorName: feeCollectorName,
		authority:        authority,
		Params:           collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the x/reward module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// SetLastReleaseTimestamp stores the last mint timestamp
func (k Keeper) SetLastReleaseTimestamp(ctx context.Context, t time.Time) error {
	return k.LastReleaseTimestamp.Set(ctx, t)
}

// GetLastReleaseTimestamp returns the last release timestamp
func (k Keeper) GetLastReleaseTimestamp(ctx context.Context) (time.Time, error) {
	return k.LastReleaseTimestamp.Get(ctx)
}

// SetLastDilutionTimestamp stores the last mint timestamp
func (k Keeper) SetLastDilutionTimestamp(ctx context.Context, t time.Time) error {
	return k.LastDilutionTimestamp.Set(ctx, t)

}

// GetLastDilutionTimestamp returns the last mint timestamp
func (k Keeper) GetLastDilutionTimestamp(ctx context.Context) (time.Time, error) {
	return k.LastDilutionTimestamp.Get(ctx)
}

// GetAnnualProvisions returns the annual provisions based on current total
// supply and inflation rate.
func (k Keeper) GetAnnualProvisions(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	annualProvisions := params.ReleaseRate.MulInt(k.bankKeeper.GetSupply(ctx, params.RewardDenom).Amount)
	return annualProvisions, nil
}

// GetRemainRewardAmount implements an alias call to the underlying bank keeper's
// GetBalance of the reward module address.
func (k Keeper) GetRemainRewardAmount(ctx context.Context, denom string) math.Int {
	return k.bankKeeper.GetBalance(ctx, k.accKeeper.GetModuleAddress(types.ModuleName), denom).Amount
}

// AddCollectedFees send released reward coins to feeCollector module account.
func (k Keeper) AddCollectedFees(ctx context.Context, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, k.feeCollectorName, fees)
}
