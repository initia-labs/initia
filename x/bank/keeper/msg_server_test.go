package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	bankkeeper "github.com/initia-labs/initia/x/bank/keeper"
)

var govAcc = authtypes.NewEmptyModuleAccount(govtypes.ModuleName, authtypes.Minter)

func TestMsgUpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// default params
	params := banktypes.DefaultParams()

	testCases := []struct {
		name      string
		input     *banktypes.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &banktypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "send enabled param",
			input: &banktypes.MsgUpdateParams{
				Authority: input.BankKeeper.GetAuthority(),
				Params: banktypes.Params{
					SendEnabled: []*banktypes.SendEnabled{
						{Denom: "foo", Enabled: true},
					},
				},
			},
			expErr:    true,
			expErrMsg: "use of send_enabled in params is no longer supported",
		},
		{
			name: "all good",
			input: &banktypes.MsgUpdateParams{
				Authority: input.BankKeeper.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := bankkeeper.NewMsgServerImpl(input.BankKeeper).UpdateParams(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgSend(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	origCoins := sdk.NewCoins(sdk.NewInt64Coin("sendableCoin", 100))
	input.BankKeeper.SetSendEnabled(ctx, origCoins.Denoms()[0], true)
	input.BankKeeper.GetBlockedAddresses()
	atom0 := sdk.NewCoins(sdk.NewInt64Coin("atom", 0))
	atom123eth0 := sdk.Coins{sdk.NewInt64Coin("atom", 123), sdk.NewInt64Coin("eth", 0)}

	testCases := []struct {
		name      string
		input     *banktypes.MsgSend
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid send to blocked address",
			input: &banktypes.MsgSend{
				FromAddress: addrs[0].String(),
				ToAddress:   authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Amount:      origCoins,
			},
			expErr:    true,
			expErrMsg: "is not allowed to receive funds",
		},
		{
			name: "invalid coins",
			input: &banktypes.MsgSend{
				FromAddress: addrs[0].String(),
				ToAddress:   addrs[1].String(),
				Amount:      atom0,
			},
			expErr:    true,
			expErrMsg: "invalid coins",
		},
		{
			name: "123atom,0eth: invalid coins",
			input: &banktypes.MsgSend{
				FromAddress: addrs[0].String(),
				ToAddress:   addrs[1].String(),
				Amount:      atom123eth0,
			},
			expErr:    true,
			expErrMsg: "123atom,0eth: invalid coins",
		},
		{
			name: "invalid from address: empty address string is not allowed: invalid address",
			input: &banktypes.MsgSend{
				FromAddress: "",
				ToAddress:   addrs[1].String(),
				Amount:      origCoins,
			},
			expErr:    true,
			expErrMsg: "empty address string is not allowed",
		},
		{
			name: "invalid to address: empty address string is not allowed: invalid address",
			input: &banktypes.MsgSend{
				FromAddress: addrs[0].String(),
				ToAddress:   "",
				Amount:      origCoins,
			},
			expErr:    true,
			expErrMsg: "empty address string is not allowed",
		},
		{
			name: "all good",
			input: &banktypes.MsgSend{
				FromAddress: addrs[0].String(),
				ToAddress:   addrs[1].String(),
				Amount:      origCoins,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			if !tc.input.Amount.IsZero() && tc.input.FromAddress != "" {
				fromAddr, err := input.AccountKeeper.AddressCodec().StringToBytes(tc.input.FromAddress)
				require.NoError(t, err)
				input.Faucet.Fund(ctx, fromAddr, tc.input.Amount...)
			}

			_, err := bankkeeper.NewMsgServerImpl(input.BankKeeper).Send(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgMultiSend(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	origDenom := "sendableCoin"
	origCoins := sdk.NewCoins(sdk.NewInt64Coin(origDenom, 100))
	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(origDenom, 50))
	input.BankKeeper.SetSendEnabled(ctx, origDenom, true)

	testCases := []struct {
		name      string
		input     *banktypes.MsgMultiSend
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "no inputs to send transaction",
			input:     &banktypes.MsgMultiSend{},
			expErr:    true,
			expErrMsg: "no inputs to send transaction",
		},
		{
			name: "no inputs to send transaction",
			input: &banktypes.MsgMultiSend{
				Outputs: []banktypes.Output{
					{Address: addrs[4].String(), Coins: sendCoins},
				},
			},
			expErr:    true,
			expErrMsg: "no inputs to send transaction",
		},
		{
			name: "more than one inputs to send transaction",
			input: &banktypes.MsgMultiSend{
				Inputs: []banktypes.Input{
					{Address: addrs[0].String(), Coins: origCoins},
					{Address: addrs[0].String(), Coins: origCoins},
				},
			},
			expErr:    true,
			expErrMsg: "multiple senders not allowed",
		},
		{
			name: "no outputs to send transaction",
			input: &banktypes.MsgMultiSend{
				Inputs: []banktypes.Input{
					{Address: addrs[0].String(), Coins: origCoins},
				},
			},
			expErr:    true,
			expErrMsg: "no outputs to send transaction",
		},
		{
			name: "invalid send to blocked address",
			input: &banktypes.MsgMultiSend{
				Inputs: []banktypes.Input{
					{Address: addrs[0].String(), Coins: origCoins},
				},
				Outputs: []banktypes.Output{
					{Address: addrs[1].String(), Coins: sendCoins},
					{Address: authtypes.NewModuleAddress(govtypes.ModuleName).String(), Coins: sendCoins},
				},
			},
			expErr:    true,
			expErrMsg: "is not allowed to receive funds",
		},
		{
			name: "input/output amount mismatch",
			input: &banktypes.MsgMultiSend{
				Inputs: []banktypes.Input{
					{Address: addrs[0].String(), Coins: origCoins},
				},
				Outputs: []banktypes.Output{
					{Address: addrs[1].String(), Coins: origCoins},
					{Address: addrs[2].String(), Coins: sendCoins},
				},
			},
			expErr:    true,
			expErrMsg: "sum inputs != sum outputs",
		},
		{
			name: "valid send",
			input: &banktypes.MsgMultiSend{
				Inputs: []banktypes.Input{
					{Address: addrs[0].String(), Coins: origCoins},
				},
				Outputs: []banktypes.Output{
					{Address: addrs[1].String(), Coins: sendCoins},
					{Address: addrs[2].String(), Coins: sendCoins},
				},
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.input.Inputs) > 0 && !tc.input.Inputs[0].Coins.IsZero() && tc.input.Inputs[0].Address != "" {
				fromAddr, err := input.AccountKeeper.AddressCodec().StringToBytes(tc.input.Inputs[0].Address)
				require.NoError(t, err)
				input.Faucet.Fund(ctx, fromAddr, tc.input.Inputs[0].Coins...)
			}

			_, err := bankkeeper.NewMsgServerImpl(input.BankKeeper).MultiSend(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgSetSendEnabled(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	testCases := []struct {
		name     string
		req      *banktypes.MsgSetSendEnabled
		isExpErr bool
		errMsg   string
	}{
		{
			name: "all good",
			req: banktypes.NewMsgSetSendEnabled(
				govAcc.GetAddress().String(),
				[]*banktypes.SendEnabled{
					banktypes.NewSendEnabled("atom1", true),
				},
				[]string{},
			),
		},
		{
			name: "all good with two denoms",
			req: banktypes.NewMsgSetSendEnabled(
				govAcc.GetAddress().String(),
				[]*banktypes.SendEnabled{
					banktypes.NewSendEnabled("atom1", true),
					banktypes.NewSendEnabled("atom2", true),
				},
				[]string{"defcoinc", "defcoind"},
			),
		},
		{
			name: "duplicate denoms",
			req: banktypes.NewMsgSetSendEnabled(
				govAcc.GetAddress().String(),
				[]*banktypes.SendEnabled{
					banktypes.NewSendEnabled("atom", true),
					banktypes.NewSendEnabled("atom", true),
				},
				[]string{},
			),
			isExpErr: true,
			errMsg:   `duplicate denom entries found for "atom": invalid request`,
		},
		{
			name: "bad first denom name, (invalid send enabled denom present in list)",
			req: banktypes.NewMsgSetSendEnabled(
				govAcc.GetAddress().String(),
				[]*banktypes.SendEnabled{
					banktypes.NewSendEnabled("not a denom", true),
					banktypes.NewSendEnabled("somecoin", true),
				},
				[]string{},
			),
			isExpErr: true,
			errMsg:   `invalid SendEnabled denom "not a denom": invalid denom: not a denom: invalid request`,
		},
		{
			name: "bad second denom name, (invalid send enabled denom present in list)",
			req: banktypes.NewMsgSetSendEnabled(
				govAcc.GetAddress().String(),
				[]*banktypes.SendEnabled{
					banktypes.NewSendEnabled("somecoin", true),
					banktypes.NewSendEnabled("not a denom", true),
				},
				[]string{},
			),
			isExpErr: true,
			errMsg:   `invalid SendEnabled denom "not a denom": invalid denom: not a denom: invalid request`,
		},
		{
			name: "invalid UseDefaultFor denom",
			req: banktypes.NewMsgSetSendEnabled(
				govAcc.GetAddress().String(),
				[]*banktypes.SendEnabled{
					banktypes.NewSendEnabled("atom", true),
				},
				[]string{"not a denom"},
			),
			isExpErr: true,
			errMsg:   `invalid UseDefaultFor denom "not a denom": invalid denom: not a denom: invalid request`,
		},
		{
			name: "invalid authority",
			req: banktypes.NewMsgSetSendEnabled(
				"invalid",
				[]*banktypes.SendEnabled{
					banktypes.NewSendEnabled("atom", true),
				},
				[]string{},
			),
			isExpErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bankkeeper.NewMsgServerImpl(input.BankKeeper).SetSendEnabled(ctx, tc.req)

			if tc.isExpErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
