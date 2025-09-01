package keeper

import (
	"context"
	"encoding/json"
)

func (k Keeper) GetTransferFunds(ctx context.Context, _ []byte) ([]byte, error) {
	transferFunds, err := k.transferFunds.Get(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(transferFunds)
}
