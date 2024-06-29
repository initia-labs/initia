package types

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// TODO: Why can't we just have one string description which can be JSON by convention
	MaxMonikerLength         = 70
	MaxIdentityLength        = 3000
	MaxWebsiteLength         = 140
	MaxSecurityContactLength = 140
	MaxDetailsLength         = 280
)

var (
	BondStatusUnspecified = BondStatus_name[int32(Unspecified)]
	BondStatusUnbonded    = BondStatus_name[int32(Unbonded)]
	BondStatusUnbonding   = BondStatus_name[int32(Unbonding)]
	BondStatusBonded      = BondStatus_name[int32(Bonded)]
)

var _ ValidatorI = Validator{}

// NewValidator constructs a new Validator
//
//nolint:interfacer
func NewValidator(operator string, pubKey cryptotypes.PubKey, description Description) (Validator, error) {
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return Validator{}, err
	}

	return Validator{
		OperatorAddress: operator,
		ConsensusPubkey: pkAny,
		Jailed:          false,
		Status:          Unbonded,
		Tokens:          sdk.NewCoins(),
		DelegatorShares: sdk.NewDecCoins(),
		Description:     description,
		UnbondingHeight: int64(0),
		UnbondingTime:   time.Unix(0, 0).UTC(),
		Commission:      NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		VotingPower:     math.ZeroInt(),
		VotingPowers:    sdk.NewCoins(),
	}, nil
}

// String implements the Stringer interface for a Validator object.
func (v Validator) String() string {
	out, _ := yaml.Marshal(v)
	return string(out)
}

// Validators is a collection of Validator
type Validators struct {
	Validators     []Validator
	ValidatorCodec address.Codec
}

func (v Validators) String() (out string) {
	for _, val := range v.Validators {
		out += val.String() + "\n"
	}

	return strings.TrimSpace(out)
}

// ToSDKValidators -  convenience function convert []Validator to []sdk.ValidatorI
func (v Validators) ToSDKValidators() (validators []ValidatorI) {
	for _, val := range v.Validators {
		validators = append(validators, val)
	}

	return validators
}

// Sort Validators sorts validator array in ascending operator address order
func (v Validators) Sort() {
	sort.Sort(v)
}

// Len implements sort interface
func (v Validators) Len() int {
	return len(v.Validators)
}

// Less implements sort interface
func (v Validators) Less(i, j int) bool {
	vi, err := v.ValidatorCodec.StringToBytes(v.Validators[i].GetOperator())
	if err != nil {
		panic(err)
	}
	vj, err := v.ValidatorCodec.StringToBytes(v.Validators[j].GetOperator())
	if err != nil {
		panic(err)
	}

	return bytes.Compare(vi, vj) == -1
}

// Swap implements sort interface
func (v Validators) Swap(i, j int) {
	v.Validators[i], v.Validators[j] = v.Validators[j], v.Validators[i]
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (v Validators) UnpackInterfaces(c codectypes.AnyUnpacker) error {
	for i := range v.Validators {
		if err := v.Validators[i].UnpackInterfaces(c); err != nil {
			return err
		}
	}
	return nil
}

// MustMarshalValidator returns the redelegation
func MustMarshalValidator(cdc codec.BinaryCodec, validator *Validator) []byte {
	return cdc.MustMarshal(validator)
}

// MustUnmarshalValidator unmarshal a redelegation from a store value
func MustUnmarshalValidator(cdc codec.BinaryCodec, value []byte) Validator {
	validator, err := UnmarshalValidator(cdc, value)
	if err != nil {
		panic(err)
	}

	return validator
}

// UnmarshalValidator unmarshal a redelegation from a store value
func UnmarshalValidator(cdc codec.BinaryCodec, value []byte) (v Validator, err error) {
	err = cdc.Unmarshal(value, &v)
	return v, err
}

// IsBonded checks if the validator status equals Bonded
func (v Validator) IsBonded() bool {
	return v.GetStatus() == Bonded
}

// IsUnbonded checks if the validator status equals Unbonded
func (v Validator) IsUnbonded() bool {
	return v.GetStatus() == Unbonded
}

// IsUnbonding checks if the validator status equals Unbonding
func (v Validator) IsUnbonding() bool {
	return v.GetStatus() == Unbonding
}

// DoNotModifyDesc constant used in flags to indicate that description field should not be updated
const DoNotModifyDesc = "[do-not-modify]"

func NewDescription(moniker, identity, website, securityContact, details string) Description {
	return Description{
		Moniker:         moniker,
		Identity:        identity,
		Website:         website,
		SecurityContact: securityContact,
		Details:         details,
	}
}

// String implements the Stringer interface for a Description object.
func (d Description) String() string {
	out, _ := yaml.Marshal(d)
	return string(out)
}

// UpdateDescription updates the fields of a given description. An error is
// returned if the resulting description contains an invalid length.
func (d Description) UpdateDescription(d2 Description) (Description, error) {
	if d2.Moniker == DoNotModifyDesc {
		d2.Moniker = d.Moniker
	}

	if d2.Identity == DoNotModifyDesc {
		d2.Identity = d.Identity
	}

	if d2.Website == DoNotModifyDesc {
		d2.Website = d.Website
	}

	if d2.SecurityContact == DoNotModifyDesc {
		d2.SecurityContact = d.SecurityContact
	}

	if d2.Details == DoNotModifyDesc {
		d2.Details = d.Details
	}

	return NewDescription(
		d2.Moniker,
		d2.Identity,
		d2.Website,
		d2.SecurityContact,
		d2.Details,
	).EnsureLength()
}

// EnsureLength ensures the length of a validator's description.
func (d Description) EnsureLength() (Description, error) {
	if len(d.Moniker) > MaxMonikerLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid moniker length; got: %d, max: %d", len(d.Moniker), MaxMonikerLength)
	}

	if len(d.Identity) > MaxIdentityLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid identity length; got: %d, max: %d", len(d.Identity), MaxIdentityLength)
	}

	if len(d.Website) > MaxWebsiteLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid website length; got: %d, max: %d", len(d.Website), MaxWebsiteLength)
	}

	if len(d.SecurityContact) > MaxSecurityContactLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid security contact length; got: %d, max: %d", len(d.SecurityContact), MaxSecurityContactLength)
	}

	if len(d.Details) > MaxDetailsLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid details length; got: %d, max: %d", len(d.Details), MaxDetailsLength)
	}

	return d, nil
}

// ABCIValidatorUpdateZero returns an abci.ValidatorUpdate from a staking validator type
// with zero power used for validator updates.
func (v Validator) ABCIValidatorUpdateZero() abci.ValidatorUpdate {
	tmProtoPk, err := v.TmConsPublicKey()
	if err != nil {
		panic(err)
	}

	return abci.ValidatorUpdate{
		PubKey: tmProtoPk,
		Power:  0,
	}
}

// SetInitialCommission attempts to set a validator's initial commission. An
// error is returned if the commission is invalid.
func (v Validator) SetInitialCommission(commission Commission) (Validator, error) {
	if err := commission.Validate(); err != nil {
		return v, err
	}

	v.Commission = commission

	return v, nil
}

// In some situations, the exchange rate becomes invalid, e.g. if
// Validator loses all tokens due to slashing. In this case,
// make all future delegations invalid.
func (v Validator) InvalidExRate() bool {
	for _, token := range v.Tokens {
		if token.IsZero() && v.DelegatorShares.AmountOf(token.Denom).IsPositive() {
			return true
		}
	}

	return false
}

// TokensFromShares calculates the token worth of provided shares
func (v Validator) TokensFromShares(shares sdk.DecCoins) sdk.DecCoins {
	tokens := sdk.NewDecCoins()
	for _, share := range shares {
		tokenAmount := sdk.NewDecCoinFromDec(
			share.Denom,
			share.Amount.MulInt(
				v.Tokens.AmountOf(share.Denom),
			).Quo(
				v.DelegatorShares.AmountOf(share.Denom),
			),
		)

		if tokenAmount.IsPositive() {
			tokens = append(
				tokens,
				tokenAmount,
			)
		}
	}

	return tokens
}

// TokensFromSharesTruncated calculates the token worth of provided shares, truncated
func (v Validator) TokensFromSharesTruncated(shares sdk.DecCoins) sdk.DecCoins {
	tokens := sdk.NewDecCoins()
	for _, share := range shares {
		tokenAmount := share.Amount.MulInt(
			v.Tokens.AmountOf(share.Denom),
		).QuoTruncate(
			v.DelegatorShares.AmountOf(share.Denom),
		)

		if tokenAmount.IsPositive() {
			tokens = append(
				tokens,
				sdk.NewDecCoinFromDec(
					share.Denom,
					tokenAmount,
				),
			)
		}
	}

	return tokens
}

// TokensFromSharesRoundUp returns the token worth of provided shares, rounded
// up.
func (v Validator) TokensFromSharesRoundUp(shares sdk.DecCoins) sdk.DecCoins {
	tokens := sdk.NewDecCoins()
	for _, share := range shares {
		tokenAmount := share.Amount.MulInt(
			v.Tokens.AmountOf(share.Denom),
		).QuoRoundUp(
			v.DelegatorShares.AmountOf(share.Denom),
		)

		if tokenAmount.IsPositive() {
			tokens = append(
				tokens,
				sdk.NewDecCoinFromDec(
					share.Denom,
					tokenAmount,
				),
			)
		}
	}

	return tokens
}

// SharesFromTokens returns the shares of a delegation given a bond amount. It
// returns an error if the validator has no tokens.
func (v Validator) SharesFromTokens(tokens sdk.Coins) (sdk.DecCoins, error) {
	shares := sdk.NewDecCoins()
	for _, token := range tokens {
		if share := v.ShareFromToken(token); share.IsPositive() {
			shares = append(shares, share)
		}
	}

	return shares, nil
}

// ShareFromToken returns the share of a delegation given a bond amount. It
// returns an error if the validator has no tokens.
func (v Validator) ShareFromToken(token sdk.Coin) sdk.DecCoin {
	totalAmount := v.Tokens.AmountOf(token.Denom)
	if totalAmount.IsZero() {
		// the first delegation to a validator sets the exchange rate to one
		return sdk.NewDecCoinFromCoin(token)
	}

	shareAmount := v.DelegatorShares.AmountOf(token.Denom).
		MulInt(token.Amount).
		QuoInt(totalAmount)

	return sdk.NewDecCoinFromDec(
		token.Denom,
		shareAmount,
	)
}

// SharesFromTokensTruncated returns the truncated shares of a delegation given
// a bond amount. It returns an error if the validator has no tokens.
func (v Validator) SharesFromTokensTruncated(tokens sdk.Coins) (sdk.DecCoins, error) {
	shares := sdk.NewDecCoins()
	for _, token := range tokens {
		if share := v.ShareFromTokenTruncated(token); share.IsPositive() {
			shares = append(shares, share)
		}
	}

	return shares, nil
}

// ShareFromTokenTruncated returns the truncated share of a delegation given
// a bond amount. It returns an error if the validator has no tokens.
func (v Validator) ShareFromTokenTruncated(token sdk.Coin) sdk.DecCoin {
	totalAmount := v.Tokens.AmountOf(token.Denom)
	if totalAmount.IsZero() {
		// the first delegation to a validator sets the exchange rate to one
		return sdk.NewDecCoinFromCoin(token)
	}

	shareAmount := v.DelegatorShares.AmountOf(token.Denom).
		MulInt(token.Amount).
		QuoTruncate(math.LegacyNewDecFromInt(totalAmount))
	return sdk.NewDecCoinFromDec(
		token.Denom,
		shareAmount,
	)
}

// BondedTokens returns the bonded tokens which the validator holds
func (v Validator) BondedTokens() sdk.Coins {
	if v.IsBonded() {
		return v.Tokens
	}

	return sdk.NewCoins()
}

// ConsensusPower converts voting power to consensus power with the given power reduction
func (v Validator) ConsensusPower(r math.Int) int64 {
	if v.IsBonded() {
		return v.PotentialConsensusPower(r)
	}
	return 0
}

// PotentialConsensusPower returns the potential consensus-engine power.
func (v Validator) PotentialConsensusPower(r math.Int) int64 {
	return sdk.TokensToConsensusPower(v.VotingPower, r)
}

// UpdateStatus updates the location of the shares within a validator
// to reflect the new status
func (v Validator) UpdateStatus(newStatus BondStatus) Validator {
	v.Status = newStatus
	return v
}

// ResetUnbondingInfos updates unbonding infos to default
func (v Validator) ResetUnbondingInfos() Validator {
	v.UnbondingId = 0
	v.UnbondingOnHoldRefCount = 0
	v.UnbondingHeight = 0
	v.UnbondingTime = time.UnixMicro(0)
	return v
}

// AddTokensFromDel adds tokens to a validator
func (v Validator) AddTokensFromDel(amounts sdk.Coins) (Validator, sdk.DecCoins) {
	// calculate the shares to issue
	var issuedShares sdk.DecCoins
	for _, coin := range amounts {
		share := v.ShareFromToken(coin)
		issuedShares = append(issuedShares, share)
	}

	v.Tokens = v.Tokens.Add(amounts...)
	v.DelegatorShares = v.DelegatorShares.Add(issuedShares...)

	return v, issuedShares
}

// RemoveTokens removes tokens from a validator
func (v Validator) RemoveTokens(tokens sdk.Coins) Validator {
	if tokens.IsAnyNegative() {
		panic(fmt.Sprintf("should not happen: trying to remove negative tokens %v", tokens))
	}

	if !v.Tokens.IsAllGTE(tokens) {
		panic(fmt.Sprintf("should not happen: only have %v tokens, trying to remove %v", v.Tokens, tokens))
	}

	v.Tokens = v.Tokens.Sub(tokens...)

	return v
}

// RemoveDelShares removes delegator shares from a validator.
// NOTE: because token fractions are left in the validator,
//
//	the exchange rate of future shares of this validator can increase.
func (v Validator) RemoveDelShares(delShares sdk.DecCoins) (Validator, sdk.Coins) {
	remainingShares := v.DelegatorShares.Sub(delShares)

	var issuedTokens sdk.Coins
	if remainingShares.IsZero() {
		// last delegation share gets any trimmings
		issuedTokens = v.Tokens
		v.Tokens = sdk.NewCoins()
	} else {
		// leave excess tokens in the validator
		// however fully use all the delegator shares
		issuedTokens, _ = v.TokensFromShares(delShares).TruncateDecimal()
		v.Tokens = v.Tokens.Sub(issuedTokens...)

		if v.Tokens.IsAnyNegative() {
			panic("attempting to remove more tokens than available in validator")
		}
	}

	v.DelegatorShares = remainingShares

	return v, issuedTokens
}

// MinEqual defines a more minimum set of equality conditions when comparing two
// validators.
func (v *Validator) MinEqual(other *Validator) bool {
	return v.OperatorAddress == other.OperatorAddress &&
		v.Status == other.Status &&
		v.Tokens.Equal(other.Tokens) &&
		v.DelegatorShares.Equal(other.DelegatorShares) &&
		v.Description.Equal(other.Description) &&
		v.Commission.Equal(other.Commission) &&
		v.Jailed == other.Jailed &&
		v.ConsensusPubkey.Equal(other.ConsensusPubkey)

}

// Equal checks if the receiver equals the parameter
func (v *Validator) Equal(v2 *Validator) bool {
	return v.MinEqual(v2) &&
		v.UnbondingHeight == v2.UnbondingHeight &&
		v.UnbondingTime.Equal(v2.UnbondingTime)
}

func (v Validator) IsJailed() bool        { return v.Jailed }
func (v Validator) GetMoniker() string    { return v.Description.Moniker }
func (v Validator) GetStatus() BondStatus { return v.Status }
func (v Validator) GetOperator() string {
	return v.OperatorAddress
}

// ConsPubKey returns the validator PubKey as a cryptotypes.PubKey.
func (v Validator) ConsPubKey() (cryptotypes.PubKey, error) {
	pk, ok := v.ConsensusPubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidType, "expecting cryptotypes.PubKey, got %T", pk)
	}

	return pk, nil

}

// Deprecated: use CmtConsPublicKey instead
func (v Validator) TmConsPublicKey() (cmtprotocrypto.PublicKey, error) {
	return v.CmtConsPublicKey()
}

// CmtConsPublicKey casts Validator.ConsensusPubkey to cmtprotocrypto.PubKey.
func (v Validator) CmtConsPublicKey() (cmtprotocrypto.PublicKey, error) {
	pk, err := v.ConsPubKey()
	if err != nil {
		return cmtprotocrypto.PublicKey{}, err
	}

	tmPk, err := cryptocodec.ToCmtProtoPublicKey(pk)
	if err != nil {
		return cmtprotocrypto.PublicKey{}, err
	}

	return tmPk, nil
}

// GetConsAddr extracts Consensus key address
func (v Validator) GetConsAddr() (sdk.ConsAddress, error) {
	pk, ok := v.ConsensusPubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidType, "expecting cryptotypes.PubKey, got %T", pk)
	}

	return sdk.ConsAddress(pk.Address()), nil
}

func (v Validator) GetTokens() sdk.Coins       { return v.Tokens }
func (v Validator) GetBondedTokens() sdk.Coins { return v.BondedTokens() }
func (v Validator) GetVotingPower() math.Int   { return v.VotingPower }
func (v Validator) GetVotingPowers() sdk.Coins { return v.VotingPowers }
func (v Validator) GetConsensusPower(r math.Int) int64 {
	return v.ConsensusPower(r)
}
func (v Validator) GetCommission() math.LegacyDec    { return v.Commission.Rate }
func (v Validator) GetDelegatorShares() sdk.DecCoins { return v.DelegatorShares }

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (v Validator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pk cryptotypes.PubKey
	return unpacker.UnpackAny(v.ConsensusPubkey, &pk)
}

// ABCIValidatorUpdate returns an abci.ValidatorUpdate from a staking validator type
// with the full validator power
func (v Validator) ABCIValidatorUpdate(r math.Int) abci.ValidatorUpdate {
	tmProtoPk, err := v.TmConsPublicKey()
	if err != nil {
		panic(err)
	}

	return abci.ValidatorUpdate{
		PubKey: tmProtoPk,
		Power:  v.ConsensusPower(r),
	}
}

// ValidatorsByVotingPower implements sort.Interface for []Validator based on
// the VotingPower and Address fields.
// The validators are sorted first by their voting power (descending). Secondary index - Address (ascending).
// Copied from tendermint/types/validator_set.go
type ValidatorsByVotingPower []Validator

func (valz ValidatorsByVotingPower) Len() int { return len(valz) }

func (valz ValidatorsByVotingPower) Less(i, j int, r math.Int) bool {
	if valz[i].ConsensusPower(r) == valz[j].ConsensusPower(r) {
		addrI, errI := valz[i].GetConsAddr()
		addrJ, errJ := valz[j].GetConsAddr()
		// If either returns error, then return false
		if errI != nil || errJ != nil {
			return false
		}
		return bytes.Compare(addrI, addrJ) == -1
	}
	return valz[i].ConsensusPower(r) > valz[j].ConsensusPower(r)
}

func (valz ValidatorsByVotingPower) Swap(i, j int) {
	valz[i], valz[j] = valz[j], valz[i]
}
