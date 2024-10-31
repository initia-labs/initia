package types

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	cosmosbanktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type MoveBankKeeper interface {
	// balance
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) (math.Int, error)
	GetPaginatedBalances(ctx context.Context, pageReq *query.PageRequest, addr sdk.AccAddress) (sdk.Coins, *query.PageResponse, error)
	GetPaginatedSupply(ctx context.Context, pageReq *query.PageRequest) (sdk.Coins, *query.PageResponse, error)
	IterateAccountBalances(ctx context.Context, addr sdk.AccAddress, cb func(sdk.Coin) (bool, error)) error
	IterateSupply(ctx context.Context, cb func(supply sdk.Coin) (bool, error)) error

	// operations
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx context.Context, addr sdk.AccAddress, amount sdk.Coins) error
	BurnCoins(ctx context.Context, addr sdk.AccAddress, amount sdk.Coins) error
	MultiSend(ctx context.Context, fromAddr sdk.AccAddress, denom string, toAddrs []sdk.AccAddress, amounts []math.Int) error

	// supply
	GetSupply(ctx context.Context, denom string) (math.Int, error)
	HasSupply(ctx context.Context, denom string) (bool, error)

	// fungible asset
	GetMetadata(ctx context.Context, denom string) (cosmosbanktypes.Metadata, error)
	HasMetadata(ctx context.Context, denom string) (bool, error)
}
