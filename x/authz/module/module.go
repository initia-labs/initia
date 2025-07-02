package authz

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"

	"github.com/initia-labs/initia/x/authz/client/cli"
)

// AppModule defines the authz module.
type AppModule struct {
	authzmodule.AppModule
	cdc codec.Codec
}

// NewAppModule returns a new AppModule.
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper, accountKeeper authz.AccountKeeper, bankKeeper authz.BankKeeper, interfaceRegistry cdctypes.InterfaceRegistry) AppModule {
	return AppModule{
		AppModule: authzmodule.NewAppModule(cdc, keeper, accountKeeper, bankKeeper, interfaceRegistry),
		cdc:       cdc,
	}
}

// GetTxCmd returns custom tx commands for the authz module.
//
// NOTE: we are using autocli style to register tx commands.
func (ab AppModule) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd(
		ab.cdc.InterfaceRegistry().SigningContext().AddressCodec(),
		ab.cdc.InterfaceRegistry().SigningContext().ValidatorAddressCodec(),
	)
}
