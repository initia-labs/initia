package cli

import (
	"cosmossdk.io/core/address"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetQueryCmd returns the query commands for IBC connections
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "ibc-nft-transfer",
		Short:                      "IBC non-fungible token transfer query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdQueryClassTrace(),
		GetCmdQueryClassTraces(),
		GetCmdParams(),
		GetCmdQueryEscrowAddress(),
		GetCmdQueryClassHash(),
		GetCmdQueryClassData(),
	)

	return queryCmd
}

// NewTxCmd returns the transaction commands for IBC non-fungible token transfer
func NewTxCmd(ac address.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "ibc-nft-transfer",
		Short:                      "IBC non-fungible token transfer transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewNftTransferTxCmd(ac),
	)

	return txCmd
}
