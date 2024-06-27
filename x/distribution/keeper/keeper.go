package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
)

// Keeper of the distribution store
type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.Codec

	authKeeper    types.AccountKeeper
	bankKeeper    types.BankKeeper
	stakingKeeper customtypes.StakingKeeper
	dexKeeper     customtypes.DexKeeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string

	feeCollectorName string // name of the FeeCollector ModuleAccount

	Schema                          collections.Schema
	Params                          collections.Item[customtypes.Params]
	FeePool                         collections.Item[types.FeePool]
	PreviousProposerConsAddr        collections.Item[[]byte]
	ValidatorOutstandingRewards     collections.Map[[]byte, customtypes.ValidatorOutstandingRewards]
	DelegatorWithdrawAddrs          collections.Map[[]byte, []byte]
	DelegatorStartingInfos          collections.Map[collections.Pair[[]byte, []byte], customtypes.DelegatorStartingInfo]
	ValidatorHistoricalRewards      collections.Map[collections.Pair[[]byte, uint64], customtypes.ValidatorHistoricalRewards]
	ValidatorCurrentRewards         collections.Map[[]byte, customtypes.ValidatorCurrentRewards]
	ValidatorAccumulatedCommissions collections.Map[[]byte, customtypes.ValidatorAccumulatedCommission]
	ValidatorSlashEvents            collections.Map[collections.Triple[[]byte, uint64, uint64], customtypes.ValidatorSlashEvent]
}

// NewKeeper creates a new distribution Keeper instance
func NewKeeper(
	cdc codec.Codec, storeService store.KVStoreService,
	ak types.AccountKeeper, bk types.BankKeeper, sk customtypes.StakingKeeper,
	dk customtypes.DexKeeper, feeCollectorName string, authority string,
) *Keeper {

	// ensure distribution module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	if _, err := ak.AddressCodec().StringToBytes(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		storeService:                    storeService,
		cdc:                             cdc,
		authKeeper:                      ak,
		bankKeeper:                      bk,
		stakingKeeper:                   sk,
		dexKeeper:                       dk,
		feeCollectorName:                feeCollectorName,
		authority:                       authority,
		Params:                          collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[customtypes.Params](cdc)),
		FeePool:                         collections.NewItem(sb, types.FeePoolKey, "fee_pool", codec.CollValue[types.FeePool](cdc)),
		PreviousProposerConsAddr:        collections.NewItem(sb, types.ProposerKey, "previous_proposer_cons_addr", collections.BytesValue),
		ValidatorOutstandingRewards:     collections.NewMap(sb, types.ValidatorOutstandingRewardsPrefix, "validator_outstanding_rewards", collections.BytesKey, codec.CollValue[customtypes.ValidatorOutstandingRewards](cdc)),
		DelegatorWithdrawAddrs:          collections.NewMap(sb, types.DelegatorWithdrawAddrPrefix, "delegator_withdraw_addrs", collections.BytesKey, collections.BytesValue),
		DelegatorStartingInfos:          collections.NewMap(sb, types.DelegatorStartingInfoPrefix, "delegator_starting_infos", collections.PairKeyCodec(collections.BytesKey, collections.BytesKey), codec.CollValue[customtypes.DelegatorStartingInfo](cdc)),
		ValidatorHistoricalRewards:      collections.NewMap(sb, types.ValidatorHistoricalRewardsPrefix, "validator_historical_rewards", collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key), codec.CollValue[customtypes.ValidatorHistoricalRewards](cdc)),
		ValidatorCurrentRewards:         collections.NewMap(sb, types.ValidatorCurrentRewardsPrefix, "validator_current_rewards", collections.BytesKey, codec.CollValue[customtypes.ValidatorCurrentRewards](cdc)),
		ValidatorAccumulatedCommissions: collections.NewMap(sb, types.ValidatorAccumulatedCommissionPrefix, "validator_accumulated_commissions", collections.BytesKey, codec.CollValue[customtypes.ValidatorAccumulatedCommission](cdc)),
		ValidatorSlashEvents:            collections.NewMap(sb, types.ValidatorSlashEventPrefix, "validator_slash_events", collections.TripleKeyCodec(collections.BytesKey, collections.Uint64Key, collections.Uint64Key), codec.CollValue[customtypes.ValidatorSlashEvent](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// GetAuthority returns the x/distribution module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// SetWithdrawAddr sets a new address that will receive the rewards upon withdrawal
func (k Keeper) SetWithdrawAddr(ctx context.Context, delegatorAddr sdk.AccAddress, withdrawAddr sdk.AccAddress) error {
	if k.bankKeeper.BlockedAddr(withdrawAddr) {
		return errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive external funds", withdrawAddr)
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	if !params.WithdrawAddrEnabled {
		return types.ErrSetWithdrawAddrDisabled
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSetWithdrawAddress,
			sdk.NewAttribute(types.AttributeKeyWithdrawAddress, withdrawAddr.String()),
		),
	)

	return k.DelegatorWithdrawAddrs.Set(ctx, delegatorAddr, withdrawAddr)
}

// WithdrawDelegationRewards withdraws rewards from a delegation
func (k Keeper) WithdrawDelegationRewards(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (customtypes.Pools, error) {
	val, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	del, err := k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil, err
	}

	// withdraw rewards
	rewards, err := k.withdrawDelegationRewards(ctx, val, del)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawRewards,
			sdk.NewAttribute(types.AttributeKeyValidator, valAddr.String()),           // validator address
			sdk.NewAttribute(sdk.AttributeKeyAmount, rewards.Sum().String()),          // rewards
			sdk.NewAttribute(customtypes.AttributeKeyAmountPerPool, rewards.String()), // rewards per pool
		),
	)

	// reinitialize the delegation
	err = k.initializeDelegation(ctx, valAddr, delAddr)
	if err != nil {
		return nil, err
	}

	return rewards, nil
}

// WithdrawValidatorCommission withdraws validator commission
func (k Keeper) WithdrawValidatorCommission(ctx context.Context, valAddr sdk.ValAddress) (customtypes.Pools, error) {
	// fetch validator accumulated commission
	accumCommission, err := k.GetValidatorAccumulatedCommission(ctx, valAddr)
	if err != nil {
		return nil, err
	}
	if accumCommission.Commissions.IsEmpty() {
		return nil, types.ErrNoValidatorCommission
	}

	commissions, remainder := accumCommission.Commissions.TruncateDecimal()
	// leave remainder to withdraw later
	if err = k.ValidatorAccumulatedCommissions.Set(ctx, valAddr, customtypes.ValidatorAccumulatedCommission{Commissions: remainder}); err != nil {
		return nil, err
	}

	// update outstanding
	outstandingRewards, err := k.GetValidatorOutstandingRewards(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	err = k.ValidatorOutstandingRewards.Set(ctx, valAddr, customtypes.ValidatorOutstandingRewards{Rewards: outstandingRewards.Rewards.Sub(customtypes.NewDecPoolsFromPools(commissions))})
	if err != nil {
		return nil, err
	}

	commissionCoins := commissions.Sum()
	if !commissionCoins.IsZero() {
		accAddr := sdk.AccAddress(valAddr)
		withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, accAddr)
		if err != nil {
			return nil, err
		}

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, commissionCoins)
		if err != nil {
			return nil, err
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawCommission,
			sdk.NewAttribute(sdk.AttributeKeyAmount, commissions.Sum().String()),
			sdk.NewAttribute(customtypes.AttributeKeyAmountPerPool, commissions.String()),
		),
	)

	return commissions, nil
}

// GetTotalRewards returns the total amount of fee distribution rewards held in the store
func (k Keeper) GetTotalRewards(ctx context.Context) (totalRewards sdk.DecCoins) {
	err := k.ValidatorOutstandingRewards.Walk(ctx, nil,
		func(_ []byte, rewards customtypes.ValidatorOutstandingRewards) (stop bool, err error) {
			totalRewards = totalRewards.Add(rewards.Rewards.Sum()...)
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	return totalRewards
}

// FundCommunityPool allows an account to directly fund the community fund pool.
// The amount is first added to the distribution module account and then directly
// added to the pool. An error is returned if the amount cannot be sent to the
// module account.
func (k Keeper) FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount); err != nil {
		return err
	}

	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(amount...)...)
	return k.FeePool.Set(ctx, feePool)
}
