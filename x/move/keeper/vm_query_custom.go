package keeper

import (
	"encoding/json"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

type CustomQueryWhiteList map[string]CustomQuery

func DefaultCustomQueryWhiteList() CustomQueryWhiteList {
	res := make(CustomQueryWhiteList)
	res["amount_to_share"] = AmountToShare
	return res
}

func EmptyCustomQueryWhiteList() CustomQueryWhiteList {
	return make(CustomQueryWhiteList)
}

type CustomQuery func(sdk.Context, []byte, *Keeper) ([]byte, error)

type AmountToShareRequest struct {
	ValAddr  string `json:"val_addr"`
	Metadata string `json:"metadata"`
	Amount   uint64 `json:"amount"`
}

type AmountToShareResponse struct {
	Share uint64 `json:"share"`
}

func AmountToShare(ctx sdk.Context, req []byte, keeper *Keeper) ([]byte, error) {
	am := AmountToShareRequest{}
	err := json.Unmarshal(req, &am)
	if err != nil {
		return nil, err
	}

	valAddr, err := keeper.vc.StringToBytes(string(am.ValAddr))
	if err != nil {
		return nil, err
	}

	metadata, err := vmtypes.NewAccountAddress(am.Metadata)
	if err != nil {
		return nil, err
	}

	denom, err := types.DenomFromMetadataAddress(ctx, NewMoveBankKeeper(keeper), metadata)
	if err != nil {
		return nil, err
	}

	share, err := keeper.AmountToShare(ctx, valAddr, sdk.NewCoin(denom, math.NewIntFromUint64(am.Amount)))
	if err != nil {
		return nil, err
	}

	res := &AmountToShareResponse{
		Share: share.Uint64(),
	}
	return json.Marshal(res)
}
