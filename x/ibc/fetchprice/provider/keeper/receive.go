package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	data types.FetchPricePacketData,
) (*types.FetchPriceAckData, error) {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return nil, err
	}

	// only attempt the application logic if the packet data
	// was successfully decoded
	resCurrencyPrices := make([]types.CurrencyPrice, 0, len(data.CurrencyIds))
	for _, id := range data.CurrencyIds {
		cp, err := oracletypes.CurrencyPairFromString(id)

		if err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidType, err.Error())
		}

		quotePrice, err := k.oracleKeeper.GetPriceForCurrencyPair(ctx, cp)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrFailedToFetchPrice, err.Error())
		}

		resCurrencyPrices = append(resCurrencyPrices, types.CurrencyPrice{
			CurrencyId: id,
			QuotePrice: types.QuotePrice{
				Price:          quotePrice.Price,
				Decimals:       uint64(cp.Decimals()),
				BlockTimestamp: quotePrice.BlockTimestamp,
				BlockHeight:    quotePrice.BlockHeight,
			},
		})

	}

	return &types.FetchPriceAckData{
		CurrencyPrices: resCurrencyPrices,
	}, nil
}
