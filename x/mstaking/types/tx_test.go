package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	"github.com/initia-labs/initia/x/mstaking/types"
)

var (
	coinPos  = sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000))
	coinZero = sdk.NewCoins()
)

func TestMsgDecode(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	// firstly we start testing the pubkey serialization

	pk1bz, err := cdc.MarshalInterface(pk1)
	require.NoError(t, err)
	var pkUnmarshaled cryptotypes.PubKey
	err = cdc.UnmarshalInterface(pk1bz, &pkUnmarshaled)
	require.NoError(t, err)
	require.True(t, pk1.Equals(pkUnmarshaled.(*ed25519.PubKey)))

	// now let's try to serialize the whole message

	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	valAddr1Str, err := valAddrCodec.BytesToString(valAddr1)
	require.NoError(t, err)

	commission1 := types.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())
	msg, err := types.NewMsgCreateValidator(valAddr1Str, pk1, coinPos, types.Description{}, commission1)
	require.NoError(t, err)
	msgSerialized, err := cdc.MarshalInterface(msg)
	require.NoError(t, err)

	var msgUnmarshaled sdk.Msg
	err = cdc.UnmarshalInterface(msgSerialized, &msgUnmarshaled)
	require.NoError(t, err)
	msg2, ok := msgUnmarshaled.(*types.MsgCreateValidator)
	require.True(t, ok)
	require.True(t, msg.Amount.Equal(msg2.Amount))
	require.True(t, msg.Pubkey.Equal(msg2.Pubkey))
}

// test Validate for MsgCreateValidator
func TestMsgCreateValidator(t *testing.T) {
	commission1 := types.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())
	commission2 := types.NewCommissionRates(math.LegacyNewDec(5), math.LegacyNewDec(5), math.LegacyNewDec(5))

	tests := []struct {
		name, moniker, identity, website, securityContact, details string
		CommissionRates                                            types.CommissionRates
		validatorAddr                                              sdk.ValAddress
		pubkey                                                     cryptotypes.PubKey
		bond                                                       sdk.Coins
		expectPass                                                 bool
	}{
		{"basic good", "a", "b", "c", "d", "e", commission1, valAddr1, pk1, coinPos, true},
		{"partial description", "", "", "c", "", "", commission1, valAddr1, pk1, coinPos, true},
		{"empty description", "", "", "", "", "", commission2, valAddr1, pk1, coinPos, false},
		{"empty address", "a", "b", "c", "d", "e", commission2, emptyAddr, pk1, coinPos, false},
		{"empty pubkey", "a", "b", "c", "d", "e", commission1, valAddr1, emptyPubkey, coinPos, false},
		{"empty bond", "a", "b", "c", "d", "e", commission2, valAddr1, pk1, coinZero, false},
		{"nil bond", "a", "b", "c", "d", "e", commission2, valAddr1, pk1, sdk.Coins{}, false},
	}

	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	for _, tc := range tests {
		description := types.NewDescription(tc.moniker, tc.identity, tc.website, tc.securityContact, tc.details)

		valAddrStr, err := valAddrCodec.BytesToString(tc.validatorAddr)
		require.NoError(t, err)

		msg, err := types.NewMsgCreateValidator(valAddrStr, tc.pubkey, tc.bond, description, tc.CommissionRates)
		require.NoError(t, err)
		if tc.expectPass {
			require.Nil(t, msg.Validate(valAddrCodec), "test: %v", tc.name)
		} else {
			require.NotNil(t, msg.Validate(valAddrCodec), "test: %v", tc.name)
		}
	}
}

// test Validate for MsgEditValidator
func TestMsgEditValidator(t *testing.T) {
	tests := []struct {
		name, moniker, identity, website, securityContact, details string
		validatorAddr                                              sdk.ValAddress
		expectPass                                                 bool
	}{
		{"basic good", "a", "b", "c", "d", "e", valAddr1, true},
		{"partial description", "", "", "c", "", "", valAddr1, true},
		{"empty description", "", "", "", "", "", valAddr1, false},
		{"empty address", "a", "b", "c", "d", "e", emptyAddr, false},
	}

	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	for _, tc := range tests {
		description := types.NewDescription(tc.moniker, tc.identity, tc.website, tc.securityContact, tc.details)
		newRate := math.LegacyZeroDec()

		valAddrStr, err := valAddrCodec.BytesToString(tc.validatorAddr)
		require.NoError(t, err)

		msg := types.NewMsgEditValidator(valAddrStr, description, &newRate)
		if tc.expectPass {
			require.Nil(t, msg.Validate(valAddrCodec), "test: %v", tc.name)
		} else {
			require.NotNil(t, msg.Validate(valAddrCodec), "test: %v", tc.name)
		}
	}
}

// test Validate for MsgDelegate
func TestMsgDelegate(t *testing.T) {
	tests := []struct {
		name          string
		delegatorAddr sdk.AccAddress
		validatorAddr sdk.ValAddress
		bond          sdk.Coins
		expectPass    bool
	}{
		{"basic good", sdk.AccAddress(valAddr1), valAddr2, coinPos, true},
		{"self bond", sdk.AccAddress(valAddr1), valAddr1, coinPos, true},
		{"empty delegator", sdk.AccAddress(emptyAddr), valAddr1, coinPos, false},
		{"empty validator", sdk.AccAddress(valAddr1), emptyAddr, coinPos, false},
		{"empty bond", sdk.AccAddress(valAddr1), valAddr2, coinZero, false},
		{"nil bold", sdk.AccAddress(valAddr1), valAddr2, sdk.Coins{}, false},
	}

	accAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	for _, tc := range tests {
		delAddrStr, err := accAddrCodec.BytesToString(tc.delegatorAddr)
		require.NoError(t, err)
		valAddrStr, err := valAddrCodec.BytesToString(tc.validatorAddr)
		require.NoError(t, err)

		msg := types.NewMsgDelegate(delAddrStr, valAddrStr, tc.bond)
		if tc.expectPass {
			require.Nil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		} else {
			require.NotNil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		}
	}
}

// test Validate for MsgUnbond
func TestMsgBeginRedelegate(t *testing.T) {
	tests := []struct {
		name             string
		delegatorAddr    sdk.AccAddress
		validatorSrcAddr sdk.ValAddress
		validatorDstAddr sdk.ValAddress
		amount           sdk.Coins
		expectPass       bool
	}{
		{"regular", sdk.AccAddress(valAddr1), valAddr2, valAddr3, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)), true},
		{"zero amount", sdk.AccAddress(valAddr1), valAddr2, valAddr3, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)), false},
		{"nil amount", sdk.AccAddress(valAddr1), valAddr2, valAddr3, sdk.Coins{}, false},
		{"empty delegator", sdk.AccAddress(emptyAddr), valAddr1, valAddr3, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)), false},
		{"empty source validator", sdk.AccAddress(valAddr1), emptyAddr, valAddr3, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)), false},
		{"empty destination validator", sdk.AccAddress(valAddr1), valAddr2, emptyAddr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)), false},
	}

	accAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	for _, tc := range tests {
		delAddrStr, err := accAddrCodec.BytesToString(tc.delegatorAddr)
		require.NoError(t, err)
		srcValAddrStr, err := valAddrCodec.BytesToString(tc.validatorSrcAddr)
		require.NoError(t, err)
		dstValAddrStr, err := valAddrCodec.BytesToString(tc.validatorDstAddr)
		require.NoError(t, err)

		msg := types.NewMsgBeginRedelegate(delAddrStr, srcValAddrStr, dstValAddrStr, tc.amount)
		if tc.expectPass {
			require.Nil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		} else {
			require.NotNil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		}
	}
}

// test Validate for MsgUnbond
func TestMsgUndelegate(t *testing.T) {
	tests := []struct {
		name          string
		delegatorAddr sdk.AccAddress
		validatorAddr sdk.ValAddress
		amount        sdk.Coins
		expectPass    bool
	}{
		{"regular", sdk.AccAddress(valAddr1), valAddr2, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)), true},
		{"zero amount", sdk.AccAddress(valAddr1), valAddr2, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)), false},
		{"nil amount", sdk.AccAddress(valAddr1), valAddr2, sdk.Coins{}, false},
		{"empty delegator", sdk.AccAddress(emptyAddr), valAddr1, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)), false},
		{"empty validator", sdk.AccAddress(valAddr1), emptyAddr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)), false},
	}

	accAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	for _, tc := range tests {
		delAddrStr, err := accAddrCodec.BytesToString(tc.delegatorAddr)
		require.NoError(t, err)
		valAddrStr, err := valAddrCodec.BytesToString(tc.validatorAddr)
		require.NoError(t, err)

		msg := types.NewMsgUndelegate(delAddrStr, valAddrStr, tc.amount)
		if tc.expectPass {
			require.Nil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		} else {
			require.NotNil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		}
	}
}

// test Validate for MsgRegisterMigration
func TestMsgRegisterMigration(t *testing.T) {
	tests := []struct {
		name          string
		authority     sdk.AccAddress
		lpDenomFrom   string
		lpDenomTo     string
		moduleAddress string
		moduleName    string
		expectPass    bool
	}{
		{
			name:          "basic good",
			authority:     sdk.AccAddress(valAddr1),
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			moduleAddress: "0x2",
			moduleName:    "dex_migration",
			expectPass:    true,
		},
		{
			name:          "empty authority",
			authority:     sdk.AccAddress(emptyAddr),
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			moduleAddress: "0x2",
			moduleName:    "dex_migration",
			expectPass:    false,
		},
		{
			name:          "empty lp denom from",
			authority:     sdk.AccAddress(valAddr1),
			lpDenomFrom:   "",
			lpDenomTo:     "ulpinitiausdt",
			moduleAddress: "0x2",
			moduleName:    "dex_migration",
			expectPass:    false,
		},
		{
			name:          "empty lp denom to",
			authority:     sdk.AccAddress(valAddr1),
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "",
			moduleAddress: "0x2",
			moduleName:    "dex_migration",
			expectPass:    false,
		},
		{
			name:          "empty module address",
			authority:     sdk.AccAddress(valAddr1),
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			moduleAddress: "",
			moduleName:    "dex_migration",
			expectPass:    false,
		},
		{
			name:          "empty module name",
			authority:     sdk.AccAddress(valAddr1),
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			moduleAddress: "0x2",
			moduleName:    "",
			expectPass:    false,
		},
		{
			name:          "valid with different module address",
			authority:     sdk.AccAddress(valAddr1),
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			moduleAddress: "0x1",
			moduleName:    "dex_migration",
			expectPass:    true,
		},
		{
			name:          "same lp denoms",
			authority:     sdk.AccAddress(valAddr1),
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdc",
			moduleAddress: "0x2",
			moduleName:    "dex_migration",
			expectPass:    false,
		},
	}

	accAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	for _, tc := range tests {
		authorityStr, err := accAddrCodec.BytesToString(tc.authority)
		require.NoError(t, err)

		msg := &types.MsgRegisterMigration{
			Authority:     authorityStr,
			DenomLpFrom:   tc.lpDenomFrom,
			DenomLpTo:     tc.lpDenomTo,
			ModuleAddress: tc.moduleAddress,
			ModuleName:    tc.moduleName,
		}

		if tc.expectPass {
			require.Nil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		} else {
			require.NotNil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		}
	}
}

// test Validate for MsgMigrateDelegation
func TestMsgMigrateDelegation(t *testing.T) {
	tests := []struct {
		name          string
		delegatorAddr sdk.AccAddress
		validatorAddr sdk.ValAddress
		lpDenomFrom   string
		lpDenomTo     string
		expectPass    bool
	}{
		{
			name:          "basic good",
			delegatorAddr: sdk.AccAddress(valAddr1),
			validatorAddr: valAddr2,
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			expectPass:    true,
		},
		{
			name:          "self delegation",
			delegatorAddr: sdk.AccAddress(valAddr1),
			validatorAddr: valAddr1,
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			expectPass:    true,
		},
		{
			name:          "empty delegator address",
			delegatorAddr: sdk.AccAddress(emptyAddr),
			validatorAddr: valAddr2,
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			expectPass:    false,
		},
		{
			name:          "empty validator address",
			delegatorAddr: sdk.AccAddress(valAddr1),
			validatorAddr: emptyAddr,
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			expectPass:    false,
		},
		{
			name:          "empty lp denom from",
			delegatorAddr: sdk.AccAddress(valAddr1),
			validatorAddr: valAddr2,
			lpDenomFrom:   "",
			lpDenomTo:     "ulpinitiausdt",
			expectPass:    false,
		},
		{
			name:          "empty lp denom to",
			delegatorAddr: sdk.AccAddress(valAddr1),
			validatorAddr: valAddr2,
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "",
			expectPass:    false,
		},
		{
			name:          "same lp denoms",
			delegatorAddr: sdk.AccAddress(valAddr1),
			validatorAddr: valAddr2,
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdc",
			expectPass:    false,
		},
		{
			name:          "different lp denoms",
			delegatorAddr: sdk.AccAddress(valAddr1),
			validatorAddr: valAddr2,
			lpDenomFrom:   "ulpinitiausdc",
			lpDenomTo:     "ulpinitiausdt",
			expectPass:    true,
		},
	}

	accAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	for _, tc := range tests {
		delAddrStr, err := accAddrCodec.BytesToString(tc.delegatorAddr)
		require.NoError(t, err)
		valAddrStr, err := valAddrCodec.BytesToString(tc.validatorAddr)
		require.NoError(t, err)

		msg := &types.MsgMigrateDelegation{
			DelegatorAddress: delAddrStr,
			ValidatorAddress: valAddrStr,
			DenomLpFrom:      tc.lpDenomFrom,
			DenomLpTo:        tc.lpDenomTo,
		}

		if tc.expectPass {
			require.Nil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		} else {
			require.NotNil(t, msg.Validate(accAddrCodec, valAddrCodec), "test: %v", tc.name)
		}
	}
}
