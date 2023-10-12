package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	customstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	customtypes "github.com/initia-labs/initia/x/slashing/types"
)

// Keeper of the slashing store
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec
	sk       customtypes.StakingKeeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a slashing keeper
func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, sk customtypes.StakingKeeper, authority string) Keeper {
	return Keeper{
		storeKey:  key,
		cdc:       cdc,
		sk:        sk,
		authority: authority,
	}
}

// GetAuthority returns the x/slashing module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// AddPubkey sets a address-pubkey relation
func (k Keeper) AddPubkey(ctx sdk.Context, pubkey cryptotypes.PubKey) error {
	bz, err := k.cdc.MarshalInterface(pubkey)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	key := types.AddrPubkeyRelationKey(pubkey.Address())
	store.Set(key, bz)
	return nil
}

// GetPubkey returns the pubkey from the adddress-pubkey relation
func (k Keeper) GetPubkey(ctx sdk.Context, a cryptotypes.Address) (cryptotypes.PubKey, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AddrPubkeyRelationKey(a))
	if bz == nil {
		return nil, fmt.Errorf("address %s not found", sdk.ConsAddress(a))
	}
	var pk cryptotypes.PubKey
	return pk, k.cdc.UnmarshalInterface(bz, &pk)
}

// Slash attempts to slash a validator. The slash is delegated to the staking
// module to make the necessary validator changes. It specifies no intraction reason.
func (k Keeper) Slash(ctx sdk.Context, consAddr sdk.ConsAddress, fraction sdk.Dec, power, distributionHeight int64) {
	k.SlashWithInfractionReason(ctx, consAddr, fraction, power, distributionHeight, stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
}

// SlashWithInfractionReason attempts to slash a validator. The slash is delegated to the staking
// module to make the necessary validator changes. It specifies an intraction reason.
func (k Keeper) SlashWithInfractionReason(ctx sdk.Context, consAddr sdk.ConsAddress, fraction sdk.Dec, power, distributionHeight int64, infraction stakingtypes.Infraction) {
	coinsBurned := k.sk.SlashWithInfractionReason(ctx, consAddr, distributionHeight, fraction, customstakingtypes.Infraction(infraction))
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlash,
			sdk.NewAttribute(types.AttributeKeyAddress, consAddr.String()),
			sdk.NewAttribute(types.AttributeKeyPower, fmt.Sprintf("%d", power)),
			sdk.NewAttribute(types.AttributeKeyReason, types.AttributeValueDoubleSign),
			sdk.NewAttribute(types.AttributeKeyBurnedCoins, coinsBurned.String()),
		),
	)
}

// Jail attempts to jail a validator. The slash is delegated to the staking module
// to make the necessary validator changes.
func (k Keeper) Jail(ctx sdk.Context, consAddr sdk.ConsAddress) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlash,
			sdk.NewAttribute(types.AttributeKeyJailed, consAddr.String()),
		),
	)

	k.sk.Jail(ctx, consAddr)
}

func (k Keeper) deleteAddrPubkeyRelation(ctx sdk.Context, addr cryptotypes.Address) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.AddrPubkeyRelationKey(addr))
}
