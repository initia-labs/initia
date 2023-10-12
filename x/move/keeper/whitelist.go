package keeper

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	distrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func (k Keeper) Whitelist(ctx sdk.Context, msg types.MsgWhitelist) error {
	if k.StakingKeeper == nil {
		return sdkerrors.ErrNotSupported
	}

	dexKeeper := NewDexKeeper(&k)

	//
	// load metadata
	//

	denomBase := k.BaseDenom(ctx)
	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return err
	}

	metadataLP, err := types.AccAddressFromString(msg.MetadataLP)
	if err != nil {
		return err
	}

	metadataA, metadataB, err := dexKeeper.GetPoolMetadata(ctx, metadataLP)
	if err != nil {
		return err
	}

	var metadataQuote vmtypes.AccountAddress
	if metadataBase == metadataA {
		metadataQuote = metadataB
	} else if metadataBase == metadataB {
		metadataQuote = metadataA
	} else {
		return errors.Wrapf(
			types.ErrInvalidDexConfig,
			"To be whitelisted, a dex should contain `%s` in its pair", denomBase,
		)
	}

	//
	// assertions
	//

	if ok, err := k.HasResource(ctx, metadataLP, vmtypes.StructTag{
		Address: vmtypes.StdAddress,
		Module:  types.MoveModuleNameCoin,
		Name:    types.ResourceNameManagingRefs,
	}); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(
			types.ErrInvalidRequest,
			"only the coins, which are generated from 0x1::coin module, can be whitelisted.",
		)
	}

	//
	// compute weights
	//

	weightBase, weightQuote, err := dexKeeper.getPoolWeights(ctx, metadataLP)
	if err != nil {
		return err
	}

	if weightBase.LT(weightQuote) {
		return errors.Wrapf(types.ErrInvalidDexConfig,
			"base weight `%s` must be bigger than quote weight `%s`", weightBase, weightQuote)
	}

	//
	// load denoms
	//

	denomLP, err := types.DenomFromMetadataAddress(ctx, NewMoveBankKeeper(&k), metadataLP)
	if err != nil {
		return err
	}

	//
	// already registered check
	//

	// check bond denom was registered
	bondDenoms := k.StakingKeeper.BondDenoms(ctx)
	for _, denom := range bondDenoms {
		if denom == denomLP {
			return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` was already registered as staking denom", metadataLP.String())
		}
	}

	// check reward weights was registered
	rewardWeights := k.distrKeeper.GetRewardWeights(ctx)
	for _, rw := range rewardWeights {
		if rw.Denom == denomLP {
			return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` reward weight was already registered", metadataLP.String())
		}
	}

	// check dex pair was registered

	if found, err := dexKeeper.hasDexPair(ctx, metadataQuote); err != nil {
		return err
	} else if found {
		return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` was already whitelisted", metadataQuote.String())
	}

	//
	// whitelist ops
	//

	// register denomLP as staking bond denom
	bondDenoms = append(bondDenoms, denomLP)
	k.StakingKeeper.SetBondDenoms(ctx, bondDenoms)

	// append denomLP reward weight to distribution keeper
	rewardWeights = append(rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: msg.RewardWeight})
	k.distrKeeper.SetRewardWeights(ctx, rewardWeights)

	// store dex pair
	dexKeeper.setDexPair(ctx, metadataQuote, metadataLP)

	// execute register if global store not found
	if found, err := k.HasStakingState(ctx, metadataLP); err != nil {
		return err
	} else if !found {

		// register LP coin to staking move module
		if err := k.InitializeStakingWithMetadata(ctx, metadataLP); err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) Delist(ctx sdk.Context, msg types.MsgDelist) error {
	if k.StakingKeeper == nil {
		return sdkerrors.ErrNotSupported
	}

	dexKeeper := NewDexKeeper(&k)

	//
	// load metadata
	//

	metadataLP, err := types.AccAddressFromString(msg.MetadataLP)
	if err != nil {
		return err
	}

	metadataA, metadataB, err := dexKeeper.GetPoolMetadata(ctx, metadataLP)
	if err != nil {
		return err
	}

	//
	// load denoms
	//

	denomLP, err := types.DenomFromMetadataAddress(ctx, NewMoveBankKeeper(&k), metadataLP)
	if err != nil {
		return err
	}

	//
	// registered check
	//

	bondDenoms := k.StakingKeeper.BondDenoms(ctx)

	// check bond denom was registered
	bondDenomIndex := -1
	for i, denom := range bondDenoms {
		if denom == denomLP {
			bondDenomIndex = i
			break
		}
	}
	if bondDenomIndex == -1 {
		return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` was not registered as staking denom", metadataLP.String())
	}

	// check reward weights was registered
	rewardWeightIndex := -1
	rewardWeights := k.distrKeeper.GetRewardWeights(ctx)
	for i, rw := range rewardWeights {
		if rw.Denom == denomLP {
			rewardWeightIndex = i
			break
		}
	}
	if rewardWeightIndex == -1 {
		return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` reward weight was not registered", metadataLP.String())
	}

	//
	// delist ops
	//

	// remove coinLP denom from the staking bond denoms
	bondDenoms = append(bondDenoms[:bondDenomIndex], bondDenoms[bondDenomIndex+1:]...)
	k.StakingKeeper.SetBondDenoms(ctx, bondDenoms)

	// remove coinLP reward weight from the distribution reward weights
	rewardWeights = append(rewardWeights[:rewardWeightIndex], rewardWeights[rewardWeightIndex+1:]...)
	k.distrKeeper.SetRewardWeights(ctx, rewardWeights)

	// delete dex pair
	dexKeeper.deleteDexPair(ctx, metadataA)
	dexKeeper.deleteDexPair(ctx, metadataB)

	return nil
}
