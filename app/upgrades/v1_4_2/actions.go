package v1_4_0

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/initia-labs/initia/app/upgrades"
)

func updateTotalEscrowAmount(ctx context.Context, app upgrades.InitiaApp) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalEscrows := sdk.NewCoins()

	// update total escrow amount by iterating all ibc channels
	var err error
	app.GetIBCKeeper().ChannelKeeper.IterateChannels(sdkCtx, func(channel channeltypes.IdentifiedChannel) bool {
		if channel.PortId != transfertypes.PortID {
			return false
		}

		escrowAddr := transfertypes.GetEscrowAddress(channel.PortId, channel.ChannelId)
		err = app.GetMoveKeeper().MoveBankKeeper().IterateAccountBalances(ctx, escrowAddr, func(c sdk.Coin) (bool, error) {
			totalEscrows = totalEscrows.Add(c)
			return false, nil
		})

		// if error occurs during iteration, break the loop and return the error
		return err != nil
	})
	if err != nil {
		return err
	}

	// Zero out stale escrow entries for denoms no longer escrowed on any transfer channel.
	for _, coin := range app.GetTransferKeeper().GetAllTotalEscrowed(sdkCtx) {
		if totalEscrows.AmountOf(coin.Denom).IsZero() {
			app.GetTransferKeeper().SetTotalEscrowForDenom(sdkCtx, sdk.NewCoin(coin.Denom, math.ZeroInt()))
		}
	}

	for _, coin := range totalEscrows {
		app.GetTransferKeeper().SetTotalEscrowForDenom(sdkCtx, coin)
	}

	return nil
}

func setupClammModuleAddress(ctx context.Context, app upgrades.InitiaApp) error {
	params, err := app.GetMoveKeeper().GetParams(ctx)
	if err != nil {
		return err
	}

	if chainID := sdk.UnwrapSDKContext(ctx).ChainID(); chainID == upgrades.MainnetChainID {
		params.ClammModuleAddress = "0xd78a3b72c7ef0cfba7286bfb8c618aa4d6011dce05a832871cc9ab323c25f55e"
	} else if chainID == upgrades.TestnetChainID {
		params.ClammModuleAddress = "0x6b41bf295bc31cd9bef75a9a5a67e5a8d6749b34a7ab3105808251fa2697823d"
	} else {
		params.ClammModuleAddress = ""
	}

	if err := app.GetMoveKeeper().SetParams(ctx, params); err != nil {
		return err
	}

	return nil
}
