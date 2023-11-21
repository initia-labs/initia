package types_test

import (
	"reflect"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/move/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func TestAuthCreateAccountsWithTypes(t *testing.T) {
	app := createApp(t)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{}).WithGasMeter(sdk.NewInfiniteGasMeter())
	testCases := []struct {
		msg           string
		accI          authtypes.AccountI
		accountNumber uint64
		expectErr     bool
	}{
		{
			msg:       "create base account",
			accI:      authtypes.NewBaseAccountWithAddress(addr1),
			expectErr: false,
		},
		{
			msg:       "create object account",
			accI:      types.NewObjectAccountWithAddress(addr2),
			expectErr: false,
		},
		{
			msg:       "create table account",
			accI:      types.NewTableAccountWithAddress(addr3),
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := (tc.accI).SetAccountNumber(tc.accI.GetAccountNumber())
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				app.AccountKeeper.SetAccount(ctx, tc.accI)
				app.AccountKeeper.NextAccountNumber(ctx)

				retrievedAccI := app.AccountKeeper.GetAccount(ctx, (tc.accI).GetAddress())

				require.Equal(t, reflect.TypeOf(tc.accI), reflect.TypeOf(retrievedAccI))
				require.Equal(t, retrievedAccI.GetAddress(), (tc.accI).GetAddress())
				require.Equal(t, retrievedAccI.GetAccountNumber(), tc.accountNumber)
			}
		})
	}

}
