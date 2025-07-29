package keeper

import (
	"context"
	"slices"

	"cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	distrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (k Keeper) Whitelist(ctx context.Context, msg types.MsgWhitelist) error {
	if k.StakingKeeper == nil {
		return sdkerrors.ErrNotSupported
	}

	//
	// load metadata
	//

	metadataLP, err := types.AccAddressFromString(k.ac, msg.MetadataLP)
	if err != nil {
		return err
	}

	//
	// dex specific whitelist ops
	//

	if balancer, err := NewBalancerKeeper(&k).Whitelist(ctx, metadataLP); err != nil {
		return err
	} else if stableswap, err := NewStableSwapKeeper(&k).Whitelist(ctx, metadataLP); err != nil {
		return err
	} else if !balancer && !stableswap {
		return errors.Wrap(
			types.ErrInvalidRequest,
			"only the coins, which are generated from 0x1::dex or 0x1::stableswap module, can be whitelisted.",
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
	// load denoms
	//

	denomLP, err := types.DenomFromMetadataAddress(ctx, k.MoveBankKeeper(), metadataLP)
	if err != nil {
		return err
	}

	//
	// already registered check
	//

	// check bond denom was registered
	bondDenoms, err := k.StakingKeeper.BondDenoms(ctx)
	if err != nil {
		return err
	}

	if slices.Contains(bondDenoms, denomLP) {
		return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` was already registered as staking denom", metadataLP.String())
	}

	// check reward weights was registered
	rewardWeights, err := k.distrKeeper.GetRewardWeights(ctx)
	if err != nil {
		return err
	}

	for _, rw := range rewardWeights {
		if rw.Denom == denomLP {
			return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` reward weight was already registered", metadataLP.String())
		}
	}

	//
	// whitelist ops
	//

	// register denomLP as staking bond denom
	bondDenoms = append(bondDenoms, denomLP)
	if err := k.StakingKeeper.SetBondDenoms(ctx, bondDenoms); err != nil {
		return err
	}

	// append denomLP reward weight to distribution keeper
	rewardWeights = append(rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: msg.RewardWeight})
	err = k.distrKeeper.SetRewardWeights(ctx, rewardWeights)
	if err != nil {
		return err
	}

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

func (k Keeper) Delist(ctx context.Context, msg types.MsgDelist) error {
	if k.StakingKeeper == nil {
		return sdkerrors.ErrNotSupported
	}

	//
	// load metadata
	//

	metadataLP, err := types.AccAddressFromString(k.ac, msg.MetadataLP)
	if err != nil {
		return err
	}

	//
	// dex specific delist ops
	//

	if err := NewBalancerKeeper(&k).Delist(ctx, metadataLP); err != nil {
		return err
	}
	if err := NewStableSwapKeeper(&k).Delist(ctx, metadataLP); err != nil {
		return err
	}

	//
	// load denoms
	//

	denomLP, err := types.DenomFromMetadataAddress(ctx, k.MoveBankKeeper(), metadataLP)
	if err != nil {
		return err
	}

	//
	// registered check
	//

	bondDenoms, err := k.StakingKeeper.BondDenoms(ctx)
	if err != nil {
		return err
	}

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
	rewardWeights, err := k.distrKeeper.GetRewardWeights(ctx)
	if err != nil {
		return err
	}

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
	err = k.StakingKeeper.SetBondDenoms(ctx, bondDenoms)
	if err != nil {
		return err
	}

	// remove coinLP reward weight from the distribution reward weights
	rewardWeights = append(rewardWeights[:rewardWeightIndex], rewardWeights[rewardWeightIndex+1:]...)
	err = k.distrKeeper.SetRewardWeights(ctx, rewardWeights)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) GetWhitelistedTokens(ctx context.Context) ([]string, error) {
	whitelistedTokens := []string{}
	err := k.DexPairs.Walk(ctx, nil, func(key, value []byte) (stop bool, err error) {
		metadataQuote, err := vmtypes.NewAccountAddressFromBytes(key)
		if err != nil {
			return true, err
		}
		denomQuote, err := types.DenomFromMetadataAddress(ctx, k.MoveBankKeeper(), metadataQuote)
		if err != nil {
			return true, err
		}
		whitelistedTokens = append(whitelistedTokens, denomQuote)
		return false, nil
	})
	return whitelistedTokens, err
}
