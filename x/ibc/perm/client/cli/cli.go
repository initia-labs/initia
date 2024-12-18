package cli

import (
	"cosmossdk.io/core/address"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// NewTxCmd returns the transaction commands for IBC non-fungible token transfer
func NewTxCmd(ac address.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "ibc-perm",
		Short:                      "IBC channel permission tx subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewUpdateAdminCmd(ac),
		NewUpdatePermissionedRelayersCmd(ac),
	)

	return txCmd
}
