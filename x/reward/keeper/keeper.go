package keeper

import (
	"time"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/reward/types"
)

// Keeper of the reward store
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	accKeeper        types.AccountKeeper
	bankKeeper       types.BankKeeper
	feeCollectorName string

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a new reward Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key storetypes.StoreKey, ak types.AccountKeeper,
	bk types.BankKeeper, feeCollectorName, authority string,
) Keeper {
	// ensure reward module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the reward module account has not been set")
	}

	return Keeper{
		cdc:              cdc,
		storeKey:         key,
		accKeeper:        ak,
		bankKeeper:       bk,
		feeCollectorName: feeCollectorName,
		authority:        authority,
	}
}

// GetAuthority returns the x/reward module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// SetLastReleaseTimestamp stores the last mint timestamp
func (k Keeper) SetLastReleaseTimestamp(ctx sdk.Context, t time.Time) {
	store := ctx.KVStore(k.storeKey)
	bz := sdk.FormatTimeBytes(t)
	store.Set(types.LastReleaseTimestampKey, bz)
}

// GetLastReleaseTimestamp returns the last release timestamp
func (k Keeper) GetLastReleaseTimestamp(ctx sdk.Context) time.Time {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastReleaseTimestampKey)
	if bz == nil {
		panic("stored last release timestamp should not have been nil")
	}

	t, err := sdk.ParseTimeBytes(bz)
	if err != nil {
		panic("stored last release timestamp should not have invalid format")
	}

	return t
}

// SetLastDilutionTimestamp stores the last mint timestamp
func (k Keeper) SetLastDilutionTimestamp(ctx sdk.Context, t time.Time) {
	store := ctx.KVStore(k.storeKey)
	bz := sdk.FormatTimeBytes(t)
	store.Set(types.LastDilutionTimestampKey, bz)
}

// GetLastDilutionTimestamp returns the last mint timestamp
func (k Keeper) GetLastDilutionTimestamp(ctx sdk.Context) time.Time {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastDilutionTimestampKey)
	if bz == nil {
		panic("stored last dilution timestamp should not have been nil")
	}

	t, err := sdk.ParseTimeBytes(bz)
	if err != nil {
		panic("stored last dilution timestamp should not have invalid format")
	}

	return t
}

// GetAnnualProvisions returns the annual provisions based on current total
// supply and inflation rate.
func (k Keeper) GetAnnualProvisions(ctx sdk.Context) sdk.Dec {
	params := k.GetParams(ctx)

	annualProvisions := params.ReleaseRate.MulInt(k.bankKeeper.GetSupply(ctx, params.RewardDenom).Amount)
	return annualProvisions
}

// GetRemainRewardAmount implements an alias call to the underlying bank keeper's
// GetBalance of the reward module address.
func (k Keeper) GetRemainRewardAmount(ctx sdk.Context, denom string) math.Int {
	return k.bankKeeper.GetBalance(ctx, k.accKeeper.GetModuleAddress(types.ModuleName), denom).Amount
}

// AddCollectedFees send released reward coins to feeCollector module account.
func (k Keeper) AddCollectedFees(ctx sdk.Context, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, k.feeCollectorName, fees)
}
