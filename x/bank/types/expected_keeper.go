package types

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	vmtypes "github.com/initia-labs/initiavm/types"
)

type MoveBankKeeper interface {
	// balance
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) (math.Int, error)
	GetPaginatedBalances(ctx context.Context, pageReq *query.PageRequest, addr sdk.AccAddress) (sdk.Coins, *query.PageResponse, error)
	GetPaginatedSupply(ctx sdk.Context, pageReq *query.PageRequest) (sdk.Coins, *query.PageResponse, error)
	IterateAccountBalances(ctx context.Context, addr sdk.AccAddress, cb func(sdk.Coin) (bool, error)) error
	IterateSupply(ctx context.Context, cb func(supply sdk.Coin) (bool, error)) error

	// store balance
	Balance(ctx context.Context, store vmtypes.AccountAddress) (vmtypes.AccountAddress, math.Int, error)

	// operations
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx context.Context, addr sdk.AccAddress, amount sdk.Coins) error
	BurnCoins(ctx context.Context, addr sdk.AccAddress, amount sdk.Coins) error

	// supply
	GetSupply(ctx context.Context, denom string) (math.Int, error)

	// fungible asset
	Issuer(context.Context, vmtypes.AccountAddress) (vmtypes.AccountAddress, error)
	Symbol(context.Context, vmtypes.AccountAddress) (string, error)
}
