package cli

import (
	"context"

	"github.com/spf13/cobra"

	"cosmossdk.io/core/address"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	consumertypes "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

// NewQueryCmd returns a root CLI command handler for all x/fetchprice transaction commands.
func NewQueryCmd(ac address.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "ibc-fetch-price",
		Short:                      "FetchPrice query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		GetPriceCmd(ac),
	)

	return txCmd
}

// GetPriceCmd returns a CLI command handler for querying a price of the given currency id.
func GetPriceCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price [currency-id]",
		Short: "Query currency price",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			currencyId := args[0]
			if _, err := oracletypes.CurrencyPairFromString(currencyId); err != nil {
				return err
			}

			queryClient := consumertypes.NewQueryClient(clientCtx)
			res, err := queryClient.Price(
				context.Background(),
				&consumertypes.QueryPriceRequest{
					CurrencyId: currencyId,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
