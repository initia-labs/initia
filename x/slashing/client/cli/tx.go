package cli

import (
	"cosmossdk.io/core/address"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

// NewTxCmd returns a root CLI command handler for all x/slashing transaction commands.
func NewTxCmd(vc address.Codec) *cobra.Command {
	slashingTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Slashing transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	slashingTxCmd.AddCommand(NewUnjailTxCmd(vc))
	return slashingTxCmd
}

// NewUnjailTxCmd returns a CLI command handler for creating a MsgUnjail transaction.
func NewUnjailTxCmd(vc address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unjail",
		Args:  cobra.NoArgs,
		Short: "unjail validator previously jailed for downtime",
		Long: `unjail a jailed validator:

$ <appd> tx slashing unjail --from mykey
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			valAddr := clientCtx.GetFromAddress()
			valAddrStr, err := vc.BytesToString(valAddr)
			if err != nil {
				return err

			}

			msg := types.NewMsgUnjail(valAddrStr)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
