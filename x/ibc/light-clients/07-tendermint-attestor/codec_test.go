package tendermintattestor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		expPass bool
	}{
		{
			"success: ClientState",
			sdk.MsgTypeURL(&ibctmattestor.ClientState{}),
			true,
		},
		{
			"success: ConsensusState",
			sdk.MsgTypeURL(&ibctmattestor.ConsensusState{}),
			true,
		},
		{
			"success: Header",
			sdk.MsgTypeURL(&ibctmattestor.Header{}),
			true,
		},
		{
			"success: Misbehaviour",
			sdk.MsgTypeURL(&ibctmattestor.Misbehaviour{}),
			true,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(ibctmattestor.AppModuleBasic{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expPass {
				require.NotNil(t, msg)
				require.NoError(t, err)
			} else {
				require.Nil(t, msg)
				require.Error(t, err)
			}
		})
	}
}
