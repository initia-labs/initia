package keeper

import (
	"context"
	"encoding/json"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetTransferFunds(ctx context.Context, _ []byte) ([]byte, error) {
	transferFunds, err := k.transferFunds.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return json.Marshal(sdk.Coin{})
	} else if err != nil {
		return nil, err
	}
	return json.Marshal(transferFunds)
}
