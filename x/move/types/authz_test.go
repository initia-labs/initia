package types_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	movetypes "github.com/initia-labs/initia/x/move/types"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestAuthzPublishAuthorization(t *testing.T) {
	app := createApp(t)
	ctx := app.BaseApp.NewContext(false).WithGasMeter(storetypes.NewInfiniteGasMeter())

	secpEncoded := "oRzrCwYAAAAMAQAGAgYOAxQ5BE0EBVFRB6IB6wEIjQMgBq0DMhDfA6sBCooFDAyWBb4BDdQGBAAAAAEAAgADBwAABAcAAgcHAQAAAAUAAQAABgIAAAAIAwQAAAkFBgAACgAHAAALCAAAAAwJCgAADQsKAAEPDg4AAhAQEQEAAhEMEQEACQEKAQEKAgEIAAEGCAADCgICBggBAQsCAQgAAwIKAgoCAgoCAQEIAQEGCAEDCgIGCAAGCAEBAQMKAgoCCgIAAQIBAwMLAgEIAAoCAQEJAAELAgEJAAlzZWNwMjU2azEFZXJyb3IGb3B0aW9uCVB1YmxpY0tleQlTaWduYXR1cmUVcHVibGljX2tleV9mcm9tX2J5dGVzE3B1YmxpY19rZXlfdG9fYnl0ZXMGT3B0aW9uEnJlY292ZXJfcHVibGljX2tleRtyZWNvdmVyX3B1YmxpY19rZXlfaW50ZXJuYWwUc2lnbmF0dXJlX2Zyb21fYnl0ZXMSc2lnbmF0dXJlX3RvX2J5dGVzBnZlcmlmeQ92ZXJpZnlfaW50ZXJuYWwFYnl0ZXMQaW52YWxpZF9hcmd1bWVudARzb21lBG5vbmUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQMIAgAAAAAAAAADCAEAAAAAAAAAAwggAAAAAAAAAAMIIQAAAAAAAAADCEAAAAAAAAAAE2luaXRpYTo6bWV0YWRhdGFfdjCVAQIBAAAAAAAAABNFX1dST05HX1BVQktFWV9TSVpFK1dyb25nIG51bWJlciBvZiBieXRlcyB3ZXJlIGdpdmVuIGFzIHB1YmtleS4CAAAAAAAAABRFX1dST05HX01FU1NBR0VfU0laRSxXcm9uZyBudW1iZXIgb2YgYnl0ZXMgd2VyZSBnaXZlbiBhcyBtZXNzYWdlLgAAAAIBDgoCAQIBDgoCAAEAAAwMDgBBDQcDIQQGBQkHAxEIJwsAEgACAQEAAAwECwAQABQCAgEAAA8eDgBBDQcCIQQGBQsLAgEHABEIJwsBCwALAhABFBEDDAUMBAsFBBoLBBEAOAAMAwUcOAEMAwsDAgMAAgAEAQAADAwOAEENBwQhBAYFCQcAEQgnCwASAQIFAQAADAQLABABFAIGAQAADBYOAEENBwIhBAYFDQsCAQsBAQcAEQgnCwALARAAFAsCEAEUEQcCBwACAAAAAQAA"
	secpCodeBytes, err := base64.RawStdEncoding.DecodeString(secpEncoded)
	require.NoError(t, err)

	addr1bech := "init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d"
	addr1hex := "0x0000000000000000000000000000000000000001"

	testCases := []struct {
		msg                  string
		pubItem              []string
		srvMsg               sdk.Msg
		expectErr            bool
		isDelete             bool
		updatedAuthorization *movetypes.PublishAuthorization
	}{
		{
			msg:                  "allow module",
			pubItem:              []string{"secp256k1"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow multiple modules",
			pubItem:              []string{"secp256k1", "ed25519"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow module with pattern: prefix",
			pubItem:              []string{"*256k1"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow module with pattern: infix",
			pubItem:              []string{"secp256*1"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow module with pattern: postfix",
			pubItem:              []string{"secp*"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow module with pattern: multi wildcards",
			pubItem:              []string{"secp*256*1"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow module with pattern: adjacent wildcards",
			pubItem:              []string{"secp**1"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow all",
			pubItem:              []string{"*"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "allow all for hex addr",
			pubItem:              []string{"*"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1hex, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg:                  "unauthorized module",
			pubItem:              []string{"notexistingmodule"},
			srvMsg:               &movetypes.MsgPublish{Sender: addr1bech, CodeBytes: [][]byte{secpCodeBytes}},
			expectErr:            true,
			isDelete:             false,
			updatedAuthorization: nil,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.msg, func(t *testing.T) {
			pubAuth, err := movetypes.NewPublishAuthorization(tc.pubItem)
			require.NoError(t, err)
			resp, err := pubAuth.Accept(ctx, tc.srvMsg)
			require.Equal(t, tc.isDelete, resp.Delete)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.updatedAuthorization != nil {
					require.Equal(t, tc.updatedAuthorization.String(), resp.Updated.String())
				}
			}
		})
	}
}

func TestAuthzExecuteAuthorization(t *testing.T) {
	app := createApp(t)
	ctx := app.BaseApp.NewContext(false).WithGasMeter(storetypes.NewInfiniteGasMeter())
	sender := "init1vrq4g0vq5ccq9khnnn9s3nzrlpvecaj2ext5an"
	addr1bech := "init1mz6qgwyu850l6xlnlauspwug9n7xun7g7m3n8m"
	addr1hex := "0xD8B404389C3D1FFD1BF3FF7900BB882CFC6E4FC8"
	addr2bech := "init1emrdsw8wwj0y4y903qzzaagxqqvgsgns32dkqp"

	testCases := []struct {
		msg                  string
		execItem             []movetypes.ExecuteAuthorizationItem
		srvMsg               sdk.Msg
		expectErr            bool
		isDelete             bool
		updatedAuthorization *movetypes.ExecuteAuthorization
	}{
		{
			msg: "allow with pattern: prefix",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"b*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow with pattern: in",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"b*r"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		}, {
			msg: "allow with pattern: postfix",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*r"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow with pattern: multiple wildcards",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"b*rb*r"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "barbar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow with pattern: adjest wildcard",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"b**r"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow all with bech32 addr",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow all with hex addr",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1hex, ModuleName: "foo", FunctionNames: []string{"*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1hex,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow multiple items",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
				{ModuleAddress: addr1bech, ModuleName: "foo2", FunctionNames: []string{"*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow all between different account format: hex-bech32",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1hex, ModuleName: "foo", FunctionNames: []string{"*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1bech,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "allow all between different account format: bech32-hex",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1hex,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            false,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "unauthorized module address",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr2bech, ModuleName: "foo", FunctionNames: []string{"*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1hex,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            true,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "unauthorized module name",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo2", FunctionNames: []string{"*"}},
				{ModuleAddress: addr2bech, ModuleName: "foo", FunctionNames: []string{"*"}},
				{ModuleAddress: addr2bech, ModuleName: "foo2", FunctionNames: []string{"*"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1hex,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            true,
			isDelete:             false,
			updatedAuthorization: nil,
		},
		{
			msg: "unauthorized function names",
			execItem: []movetypes.ExecuteAuthorizationItem{
				{ModuleAddress: addr1bech, ModuleName: "foo2", FunctionNames: []string{"zaar", "baar", "caar"}},
			},
			srvMsg: &movetypes.MsgExecute{
				Sender:        sender,
				ModuleAddress: addr1hex,
				ModuleName:    "foo",
				FunctionName:  "bar",
			},
			expectErr:            true,
			isDelete:             false,
			updatedAuthorization: nil,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.msg, func(t *testing.T) {
			execAuth, err := movetypes.NewExecuteAuthorization(address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()), tc.execItem)
			require.NoError(t, err)
			resp, err := execAuth.Accept(ctx, tc.srvMsg)
			require.Equal(t, tc.isDelete, resp.Delete)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.updatedAuthorization != nil {
					require.Equal(t, tc.updatedAuthorization.String(), resp.Updated.String())
				}
			}
		})
	}
}

func TestAuthzExecuteAuthorizationDuplicate(t *testing.T) {
	addr1bech := "init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d"
	addr1hex := "0x0000000000000000000000000000000000000001"
	addr2bech := "init1emrdsw8wwj0y4y903qzzaagxqqvgsgns32dkqp"

	ac := address.NewBech32Codec("init")

	// normal cases
	execAuth, err := movetypes.NewExecuteAuthorization(ac, []movetypes.ExecuteAuthorizationItem{
		{ModuleAddress: addr1hex, ModuleName: "foo", FunctionNames: []string{"*"}},
		{ModuleAddress: addr2bech, ModuleName: "foo", FunctionNames: []string{"*"}},
	})
	require.NoError(t, err)
	require.NoError(t, execAuth.ValidateBasic())

	execAuth, err = movetypes.NewExecuteAuthorization(ac, []movetypes.ExecuteAuthorizationItem{
		{ModuleAddress: addr1bech, ModuleName: "bar", FunctionNames: []string{"*"}},
		{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
	})
	require.NoError(t, err)
	require.NoError(t, execAuth.ValidateBasic())

	// duplicated cases
	execAuth, err = movetypes.NewExecuteAuthorization(ac, []movetypes.ExecuteAuthorizationItem{
		{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
		{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
	})
	require.NoError(t, err)
	require.Error(t, execAuth.ValidateBasic())

	execAuth, err = movetypes.NewExecuteAuthorization(ac, []movetypes.ExecuteAuthorizationItem{
		{ModuleAddress: addr1hex, ModuleName: "foo", FunctionNames: []string{"*"}},
		{ModuleAddress: addr1hex, ModuleName: "foo", FunctionNames: []string{"*"}},
	})
	require.NoError(t, err)
	require.Error(t, execAuth.ValidateBasic())

	execAuth, err = movetypes.NewExecuteAuthorization(ac, []movetypes.ExecuteAuthorizationItem{
		{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
		{ModuleAddress: addr1hex, ModuleName: "foo", FunctionNames: []string{"*"}},
	})
	require.NoError(t, err)
	require.Error(t, execAuth.ValidateBasic())

	execAuth, err = movetypes.NewExecuteAuthorization(ac, []movetypes.ExecuteAuthorizationItem{
		{ModuleAddress: addr1hex, ModuleName: "foo", FunctionNames: []string{"*"}},
		{ModuleAddress: addr1bech, ModuleName: "foo", FunctionNames: []string{"*"}},
	})
	require.NoError(t, err)
	require.Error(t, execAuth.ValidateBasic())

}
