package types

import (
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/initiavm/types"
)

type MoveBankKeeper interface {
	// balance
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) (math.Int, error)
	GetUserStores(ctx sdk.Context, addr sdk.AccAddress) (*prefix.Store, error)

	// store balance
	Balance(ctx sdk.Context, store vmtypes.AccountAddress) (vmtypes.AccountAddress, math.Int, error)

	// operations
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, addr sdk.AccAddress, amount sdk.Coins) error
	BurnCoins(ctx sdk.Context, addr sdk.AccAddress, amount sdk.Coins) error

	// supply
	GetSupply(ctx sdk.Context, denom string) (math.Int, error)
	GetIssuers(ctx sdk.Context) (prefix.Store, error)

	// fungible asset
	Issuer(sdk.Context, vmtypes.AccountAddress) (vmtypes.AccountAddress, error)
	Symbol(sdk.Context, vmtypes.AccountAddress) (string, error)
}
