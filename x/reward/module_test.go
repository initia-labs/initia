package reward_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/x/reward/types"
)

func TestItCreatesModuleAccountOnInitBlock(t *testing.T) {
	app := createApp(t)
	ctx := app.BaseApp.NewContext(true)
	acc := app.AccountKeeper.GetAccount(ctx, authtypes.NewModuleAddress(types.ModuleName))
	require.NotNil(t, acc)
}
