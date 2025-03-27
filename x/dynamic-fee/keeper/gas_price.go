package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/tx"
)

var _ tx.GasPriceKeeper = Keeper{}

// GasPrices return gas prices for all whitelisted denoms
func (k Keeper) GasPrices(
	ctx context.Context,
) (sdk.DecCoins, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	baseGasPrice := params.BaseGasPrice
	baseDenom, err := k.baseDenomKeeper.BaseDenom(ctx)
	if err != nil {
		return nil, err
	}

	whitelistedTokens, err := k.whitelistKeeper.GetWhitelistedTokens(ctx)
	if err != nil {
		return nil, err
	}

	gasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(baseDenom, baseGasPrice))
	for _, denom := range whitelistedTokens {
		baseSpotPrice, err := k.tokenPriceKeeper.GetBaseSpotPrice(ctx, denom)
		if err != nil {
			return nil, err
		}
		if baseSpotPrice.IsZero() {
			return nil, fmt.Errorf("baseSpotPrice is zero: %s", denom)
		}

		gasPrice := baseGasPrice.Quo(baseSpotPrice)
		gasPrices = gasPrices.Add(sdk.NewDecCoinFromDec(denom, gasPrice))
	}

	return gasPrices, nil
}

// GasPrice return gas price for the given denom
func (k Keeper) GasPrice(
	ctx context.Context,
	denom string,
) (sdk.DecCoin, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return sdk.DecCoin{}, err
	}

	baseGasPrice := params.BaseGasPrice
	baseDenom, err := k.baseDenomKeeper.BaseDenom(ctx)
	if err != nil {
		return sdk.DecCoin{}, err
	}

	// if denom is base denom, return base gas price
	if denom == baseDenom {
		return sdk.NewDecCoinFromDec(baseDenom, baseGasPrice), nil
	}

	// if denom is not base denom, get base spot price
	baseSpotPrice, err := k.tokenPriceKeeper.GetBaseSpotPrice(ctx, denom)
	if err != nil {
		return sdk.DecCoin{}, err
	} else if baseSpotPrice.IsZero() {
		return sdk.DecCoin{}, fmt.Errorf("baseSpotPrice is zero: %s", denom)
	}

	return sdk.NewDecCoinFromDec(denom, baseGasPrice.Quo(baseSpotPrice)), nil
}
