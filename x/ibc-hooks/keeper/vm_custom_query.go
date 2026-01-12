package keeper

import (
	"context"
	"encoding/json"
	"errors"

	"cosmossdk.io/collections"
)

// QueryTransferFunds is a custom query that returns the transfer funds.
func (k Keeper) QueryTransferFunds(ctx context.Context, _ []byte) ([]byte, error) {
	transferFunds, err := k.GetTransferFunds(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return json.Marshal(nil)
	} else if err != nil {
		return nil, err
	}
	return json.Marshal(transferFunds)
}
