package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/initia-labs/initia/x/mstaking/types"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Implements ValidatorSet interface
var _ types.ValidatorSet = Keeper{}

// Implements DelegationSet interface
var _ types.DelegationSet = Keeper{}

// keeper of the staking store
type Keeper struct {
	storeKey          storetypes.StoreKey
	cdc               codec.BinaryCodec
	authKeeper        types.AccountKeeper
	bankKeeper        types.BankKeeper
	VotingPowerKeeper types.VotingPowerKeeper
	hooks             types.StakingHooks
	slashingHooks     types.SlashingHooks
	authority         string
}

// NewKeeper creates a new staking Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	vk types.VotingPowerKeeper,
	authority string,
) Keeper {
	// ensure bonded and not bonded module accounts are set
	if addr := ak.GetModuleAddress(types.BondedPoolName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.BondedPoolName))
	}

	if addr := ak.GetModuleAddress(types.NotBondedPoolName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.NotBondedPoolName))
	}

	// ensure that authority is a valid AccAddress
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic("authority is not a valid acc address")
	}

	return Keeper{
		storeKey:          key,
		cdc:               cdc,
		authKeeper:        ak,
		bankKeeper:        bk,
		VotingPowerKeeper: vk,
		hooks:             nil,
		slashingHooks:     nil,
		authority:         authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// Hooks gets the hooks for staking *Keeper {
func (k *Keeper) Hooks() types.StakingHooks {
	if k.hooks == nil {
		// return a no-op implementation if no hooks are set
		return types.MultiStakingHooks{}
	}

	return k.hooks
}

// Set the staking hooks
func (k *Keeper) SetHooks(sh types.StakingHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set staking hooks twice")
	}

	k.hooks = sh

	return k
}

// SlashingHooks gets the slashing hooks for staking *Keeper {
func (k *Keeper) SlashingHooks() types.SlashingHooks {
	if k.hooks == nil {
		// return a no-op implementation if no hooks are set
		return types.MultiSlashingHooks{}
	}

	return k.slashingHooks
}

// Set the slashing hooks
func (k *Keeper) SetSlashingHooks(sh types.SlashingHooks) *Keeper {
	if k.slashingHooks != nil {
		panic("cannot set slashing hooks twice")
	}

	k.slashingHooks = sh

	return k
}

// GetAuthority returns the x/staking module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
