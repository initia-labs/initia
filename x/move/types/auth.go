package types

import (
	"fmt"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var (
	_ authtypes.AccountI       = (*ObjectAccount)(nil)
	_ authtypes.GenesisAccount = (*ObjectAccount)(nil)
)

// NewObjectAccountWithAddress create new object account with the given address.
func NewObjectAccountWithAddress(addr sdk.AccAddress) *ObjectAccount {
	return &ObjectAccount{
		authtypes.NewBaseAccountWithAddress(addr),
	}
}

// SetPubKey - Implements AccountI
func (ma ObjectAccount) SetPubKey(pubKey cryptotypes.PubKey) error {
	return fmt.Errorf("not supported for object accounts")
}
