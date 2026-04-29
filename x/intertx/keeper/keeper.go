package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"

	"github.com/initia-labs/initia/x/intertx/types"
)

type Keeper struct {
	cdc codec.Codec

	ac                  address.Codec
	icaControllerKeeper icacontrollerkeeper.Keeper
}

func NewKeeper(cdc codec.Codec, iaKeeper icacontrollerkeeper.Keeper, ac address.Codec) Keeper {
	return Keeper{
		cdc:                 cdc,
		icaControllerKeeper: iaKeeper,
		ac:                  ac,
	}
}

// Logger returns the application logger, scoped to the associated module
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
