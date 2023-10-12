package types

import (
	"cosmossdk.io/math"
	tmprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DelegationI delegation bond for a delegated proof of stake system
type DelegationI interface {
	GetDelegatorAddr() sdk.AccAddress // delegator sdk.AccAddress for the bond
	GetValidatorAddr() sdk.ValAddress // validator operator address
	GetShares() sdk.DecCoins          // amount of validator's shares held in this delegation
}

// ValidatorI expected validator functions
type ValidatorI interface {
	IsJailed() bool                                                // whether the validator is jailed
	GetMoniker() string                                            // moniker of the validator
	GetStatus() BondStatus                                         // status of the validator
	IsBonded() bool                                                // check if has a bonded status
	IsUnbonded() bool                                              // check if has status unbonded
	IsUnbonding() bool                                             // check if has status unbonding
	GetOperator() sdk.ValAddress                                   // operator address to receive/return validators coins
	ConsPubKey() (cryptotypes.PubKey, error)                       // validation consensus pubkey (cryptotypes.PubKey)
	TmConsPublicKey() (tmprotocrypto.PublicKey, error)             // validation consensus pubkey (Tendermint)
	GetConsAddr() (sdk.ConsAddress, error)                         // validation consensus address
	GetTokens() sdk.Coins                                          // validation tokens
	GetBondedTokens() sdk.Coins                                    // validator bonded tokens
	GetVotingPower() math.Int                                      // validation voting power
	GetVotingPowers() sdk.Coins                                    // validation voting powers
	GetConsensusPower(math.Int) int64                              // validation power in tendermint
	GetCommission() sdk.Dec                                        // validator commission rate
	GetDelegatorShares() sdk.DecCoins                              // total outstanding delegator shares
	TokensFromShares(sdk.DecCoins) sdk.DecCoins                    // token worth of provided delegator shares
	TokensFromSharesTruncated(sdk.DecCoins) sdk.DecCoins           // token worth of provided delegator shares, truncated
	TokensFromSharesRoundUp(sdk.DecCoins) sdk.DecCoins             // token worth of provided delegator shares, rounded up
	SharesFromTokens(amt sdk.Coins) (sdk.DecCoins, error)          // shares worth of delegator's bond
	SharesFromTokensTruncated(amt sdk.Coins) (sdk.DecCoins, error) // truncated shares worth of delegator's bond
}
