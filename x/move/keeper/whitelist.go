package keeper

import (
	"context"
	"slices"

	"cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	vmtypes "github.com/initia-labs/movevm/types"

	distrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/move/types"
)

func (k Keeper) WhitelistGasPrice(ctx context.Context, msg types.MsgWhitelistGasPrice) error {
	//
	// load metadata
	//

	metadataQuote, err := types.AccAddressFromString(k.ac, msg.MetadataQuote)
	if err != nil {
		return err
	}

	metadataLP, err := types.AccAddressFromString(k.ac, msg.MetadataLP)
	if err != nil {
		return err
	}

	//
	// dex specific validation
	//

	dexKeeper := NewDexKeeper(&k)
	if found, err := dexKeeper.hasDexPair(ctx, metadataQuote); err != nil {
		return err
	} else if found {
		return errors.Wrapf(types.ErrInvalidRequest, "coin `%s` was already whitelisted", metadataQuote.String())
	}

	if balancer, err := NewBalancerKeeper(&k).WhitelistGasPrice(ctx, metadataQuote, metadataLP); err != nil {
		return err
	} else if balancer {
		return dexKeeper.setDexPair(ctx, metadataQuote, metadataLP)
	}

	if stableswap, err := NewStableSwapKeeper(&k).WhitelistGasPrice(ctx, metadataQuote, metadataLP); err != nil {
		return err
	} else if stableswap {
		return dexKeeper.setDexPair(ctx, metadataQuote, metadataLP)
	}

	// CLAMM pool: module address comes from params
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	if params.ClammModuleAddress != "" {
		clammModuleAddr, err := types.AccAddressFromString(k.ac, params.ClammModuleAddress)
		if err != nil {
			return err
		}
		if ok, err := NewCLAMMKeeper(&k, clammModuleAddr).WhitelistGasPrice(ctx, metadataQuote, metadataLP); err != nil {
			return err
		} else if ok {
			return dexKeeper.setDexPair(ctx, metadataQuote, metadataLP)
		}
	}

	return errors.Wrap(
		types.ErrInvalidRequest,
		"only the coins, which are generated from 0x1::dex, 0x1::stableswap, or a CLAMM module, can be whitelisted.",
	)
}

func (k Keeper) DelistGasPrice(ctx context.Context, msg types.MsgDelistGasPrice) error {
	//
	// load metadata
	//

	metadataQuote, err := types.AccAddressFromString(k.ac, msg.MetadataQuote)
	if err != nil {
		return err
	}

	metadataLP, err := types.AccAddressFromString(k.ac, msg.MetadataLP)
	if err != nil {
		return err
	}

	//
	// dex specific validation
	//

	dexKeeper := NewDexKeeper(&k)
	storedMetadataLP, err := dexKeeper.getMetadataLP(ctx, metadataQuote)
	if err != nil {
		return err
	}
	if storedMetadataLP != metadataLP {
		return errors.Wrapf(
			types.ErrInvalidRequest,
			"invalid metadata LP `%s` for quote `%s`; currently registered LP is `%s`",
			metadataLP.String(),
			metadataQuote.String(),
			storedMetadataLP.String(),
		)
	}

	if balancer, err := NewBalancerKeeper(&k).DelistGasPrice(ctx, metadataQuote, metadataLP); err != nil {
		return err
	} else if balancer {
		return dexKeeper.deleteDexPair(ctx, metadataQuote)
	}

	if stableswap, err := NewStableSwapKeeper(&k).DelistGasPrice(ctx, metadataQuote, metadataLP); err != nil {
		return err
	} else if stableswap {
		return dexKeeper.deleteDexPair(ctx, metadataQuote)
	}

	// CLAMM pool: module address comes from params
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	if params.ClammModuleAddress != "" {
		clammModuleAddr, err := types.AccAddressFromString(k.ac, params.ClammModuleAddress)
		if err != nil {
			return err
		}
		if ok, err := NewCLAMMKeeper(&k, clammModuleAddr).DelistGasPrice(ctx, metadataQuote, metadataLP); err != nil {
			return err
		} else if ok {
			return dexKeeper.deleteDexPair(ctx, metadataQuote)
		}
	}

	return errors.Wrap(
		types.ErrInvalidRequest,
		"only the coins, which are generated from 0x1::dex, 0x1::stableswap, or a CLAMM module, can be delisted.",
	)
}

func (k Keeper) WhitelistStaking(ctx context.Context, msg types.MsgWhitelistStaking) error {
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

	if balancer, err := NewBalancerKeeper(&k).WhitelistStaking(ctx, metadataLP); err != nil {
		return err
	} else if stableswap, err := NewStableSwapKeeper(&k).WhitelistStaking(ctx, metadataLP); err != nil {
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

func (k Keeper) DelistStaking(ctx context.Context, msg types.MsgDelistStaking) error {
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

	if err := NewBalancerKeeper(&k).DelistStaking(ctx, metadataLP); err != nil {
		return err
	}
	if err := NewStableSwapKeeper(&k).DelistStaking(ctx, metadataLP); err != nil {
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

func (k Keeper) GetWhitelistedGasTokens(ctx context.Context) ([]string, error) {
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
