package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	cmt "github.com/cometbft/cometbft/proto/tendermint/types"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/version"

	cmtlocal "github.com/cometbft/cometbft/rpc/client/local"
	"github.com/initia-labs/initia/tx/types"
)

const (
	FlagQuery     = "query"
	FlagType      = "type"
	FlagIndexerV2 = "v2"
)

func QueryGasPriceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-price [denom]",
		Short: "Query for the gas price",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			client := types.NewQueryClient(clientCtx)
			res, err := client.GasPrice(context.Background(), &types.QueryGasPriceRequest{Denom: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func QueryGasPricesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-prices",
		Short: "Query for the gas prices",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			client := types.NewQueryClient(clientCtx)
			res, err := client.GasPrices(context.Background(), &types.QueryGasPricesRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryBlocksCmd returns a command to search through blocks by events.
func QueryBlocksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocks-v2",
		Short: "Query for paginated blocks that match a set of events with indexer v2",
		Long: `Search for blocks that match the exact given events where results are paginated.
The events query is directly passed to CometBFT's RPC BlockSearch method and must
conform to CometBFT's query syntax.

Please refer to each module's documentation for the full set of events to query
for. Each module documents its respective events under 'xx_events.md'.

This method uses a bloom filter to speed up queries in most cases.
`,
		Example: fmt.Sprintf(
			"$ %s query blocks-v2 --query \"message.sender='cosmos1...' AND block.height > 7\" --page 1 --limit 30",
			version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			query, _ := cmd.Flags().GetString(FlagQuery)
			page, _ := cmd.Flags().GetInt(flags.FlagPage)
			limit, _ := cmd.Flags().GetInt(flags.FlagLimit)

			blocks, err := QueryBlocksV2(clientCtx, page, limit, query)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(blocks)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().Int(flags.FlagPage, querytypes.DefaultPage, "Query a specific page of paginated results")
	cmd.Flags().Int(flags.FlagLimit, querytypes.DefaultLimit, "Query number of transactions results per page returned")
	cmd.Flags().String(FlagQuery, "", "The blocks events query per CometBFT's query semantics")
	_ = cmd.MarkFlagRequired(FlagQuery)

	return cmd
}

// QueryTxsByEventsCmd returns a command to search through transactions by events.
func QueryTxsByEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "txs-v2",
		Short: "Query for paginated transactions that match a set of events with indexer v2",
		Long: `Search for transactions that match the exact given events where results are paginated.
The events query is directly passed to Tendermint's RPC TxSearch method and must
conform to Tendermint's query syntax.

Please refer to each module's documentation for the full set of events to query
for. Each module documents its respective events under 'xx_events.md'.

This method uses a bloom filter to speed up queries in most cases.
`,
		Example: fmt.Sprintf(
			"$ %s query txs-v2 --query \"message.sender='cosmos1...' AND message.action='withdraw_delegator_reward' AND tx.height > 7\" --page 1 --limit 30",
			version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			query, _ := cmd.Flags().GetString(FlagQuery)
			page, _ := cmd.Flags().GetInt(flags.FlagPage)
			limit, _ := cmd.Flags().GetInt(flags.FlagLimit)

			txs, err := QueryTxsByEventsV2(clientCtx, page, limit, query)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(txs)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().Int(flags.FlagPage, querytypes.DefaultPage, "Query a specific page of paginated results")
	cmd.Flags().Int(flags.FlagLimit, querytypes.DefaultLimit, "Query number of transactions results per page returned")
	cmd.Flags().String(FlagQuery, "", "The transactions events query per Tendermint's query semantics")
	_ = cmd.MarkFlagRequired(FlagQuery)

	return cmd
}

func QueryBlocksV2(clientCtx client.Context, page, limit int, query string) (*sdk.SearchBlocksResult, error) {
	node, err := clientCtx.GetNode()
	if err != nil {
		return nil, err
	}

	rpcClient, err := nodeToCometRPCV2(node)
	if err != nil {
		return nil, err
	}

	resBlocks, err := rpcClient.BlockSearchV2(context.Background(), query, &page, &limit, "")

	if err != nil {
		return nil, err
	}

	blocks, err := formatBlockResults(resBlocks.Blocks)
	if err != nil {
		return nil, err
	}

	result := sdk.NewSearchBlocksResult(int64(resBlocks.TotalCount), int64(len(blocks)), int64(page), int64(limit), blocks)

	return result, nil
}

func formatBlockResults(resBlocks []*coretypes.ResultBlock) ([]*cmt.Block, error) {
	out := make([]*cmt.Block, len(resBlocks))
	for i := range resBlocks {
		out[i] = sdk.NewResponseResultBlock(resBlocks[i], resBlocks[i].Block.Time.Format(time.RFC3339))
		if out[i] == nil {
			return nil, fmt.Errorf("unable to create response block from comet result block: %v", resBlocks[i])
		}
	}

	return out, nil
}

func QueryTxsByEventsV2(clientCtx client.Context, page, limit int, query string) (*sdk.SearchTxsResult, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// CometBFT node.TxSearch that is used for querying txs defines pages
	// starting from 1, so we default to 1 if not provided in the request.
	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = querytypes.DefaultLimit
	}

	node, err := clientCtx.GetNode()
	if err != nil {
		return nil, err
	}

	rpcClient, err := nodeToCometRPCV2(node)
	if err != nil {
		return nil, err
	}

	resTxs, err := rpcClient.TxSearchV2(context.Background(), query, false, &page, &limit, "")
	if err != nil {
		return nil, fmt.Errorf("failed to search for txs: %w", err)
	}

	resBlocks, err := getBlocksForTxResults(clientCtx, resTxs.Txs)
	if err != nil {
		return nil, err
	}

	txs, err := formatTxResults(clientCtx.TxConfig, resTxs.Txs, resBlocks)
	if err != nil {
		return nil, err
	}

	return sdk.NewSearchTxsResult(uint64(resTxs.TotalCount), uint64(len(txs)), uint64(page), uint64(limit), txs), nil
}

// formatTxResults parses the indexed txs into a slice of TxResponse objects.
func formatTxResults(txConfig client.TxConfig, resTxs []*coretypes.ResultTx, resBlocks map[int64]*coretypes.ResultBlock) ([]*sdk.TxResponse, error) {
	var err error
	out := make([]*sdk.TxResponse, len(resTxs))
	for i := range resTxs {
		out[i], err = mkTxResult(txConfig, resTxs[i], resBlocks[resTxs[i].Height])
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func getBlocksForTxResults(clientCtx client.Context, resTxs []*coretypes.ResultTx) (map[int64]*coretypes.ResultBlock, error) {
	node, err := clientCtx.GetNode()
	if err != nil {
		return nil, err
	}

	resBlocks := make(map[int64]*coretypes.ResultBlock)

	for _, resTx := range resTxs {
		if _, ok := resBlocks[resTx.Height]; !ok {
			resBlock, err := node.Block(context.Background(), &resTx.Height)
			if err != nil {
				return nil, err
			}

			resBlocks[resTx.Height] = resBlock
		}
	}

	return resBlocks, nil
}

func mkTxResult(txConfig client.TxConfig, resTx *coretypes.ResultTx, resBlock *coretypes.ResultBlock) (*sdk.TxResponse, error) {
	txb, err := txConfig.TxDecoder()(resTx.Tx)
	if err != nil {
		return nil, err
	}
	p, ok := txb.(intoAny)
	if !ok {
		return nil, fmt.Errorf("expecting a type implementing intoAny, got: %T", txb)
	}
	any := p.AsAny()
	return sdk.NewResponseResultTx(resTx, any, resBlock.Block.Time.Format(time.RFC3339)), nil
}

// Deprecated: this interface is used only internally for scenario we are
// deprecating (StdTxConfig support)
type intoAny interface {
	AsAny() *codectypes.Any
}

type CometRPCV2 interface {
	client.CometRPC
	TxSearchV2(
		ctx context.Context,
		query string,
		prove bool,
		page, perPage *int,
		orderBy string,
	) (*coretypes.ResultTxSearch, error)
	BlockSearchV2(
		ctx context.Context,
		query string,
		page, perPage *int,
		orderBy string,
	) (*coretypes.ResultBlockSearch, error)
}

func nodeToCometRPCV2(node client.CometRPC) (CometRPCV2, error) {
	switch node := node.(type) {
	case *rpchttp.HTTP:
		return node, nil
	case *cmtlocal.Local:
		return node, nil
	default:
		return nil, fmt.Errorf("invalid client")
	}
}
