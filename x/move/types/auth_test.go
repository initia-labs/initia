package types_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/move/types"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func TestAuthCreateAccountsWithTypes(t *testing.T) {
	app := createApp(t)
	ctx := app.BaseApp.NewContext(false).WithGasMeter(storetypes.NewInfiniteGasMeter())
	testCases := []struct {
		msg  string
		accI sdk.AccountI
	}{
		{
			msg:  "create base account",
			accI: authtypes.NewBaseAccountWithAddress(addr1),
		},
		{
			msg:  "create object account",
			accI: types.NewObjectAccountWithAddress(addr2),
		},
		{
			msg:  "create table account",
			accI: types.NewTableAccountWithAddress(addr3),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			accI := app.AccountKeeper.NewAccount(ctx, tc.accI)
			require.NotPanics(t, func() {
				app.AccountKeeper.SetAccount(ctx, accI)
			})

			retrievedAccI := app.AccountKeeper.GetAccount(ctx, (tc.accI).GetAddress())

			require.Equal(t, reflect.TypeOf(tc.accI), reflect.TypeOf(retrievedAccI))
			require.Equal(t, retrievedAccI.GetAddress(), (tc.accI).GetAddress())
		})
	}

}
