//go:build e2e

package mempool

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkbech32 "github.com/cosmos/cosmos-sdk/types/bech32"
	e2e "github.com/initia-labs/initia/integration-tests/e2e"
	movetypes "github.com/initia-labs/initia/x/move/types"
	"github.com/stretchr/testify/require"
)

func TestMoveTableGeneratorResourceScenario(t *testing.T) {
	ctx := context.Background()

	cluster, err := e2e.NewCluster(ctx, t, e2e.ClusterOptions{
		NodeCount:    3,
		AccountCount: 4,
		ChainID:      "testnet-move-resource",
		BasePort:     26100,
		PortStride:   20,
	})
	require.NoError(t, err)
	defer cluster.Close()

	require.NoError(t, cluster.Start(ctx))
	require.NoError(t, cluster.WaitForReady(ctx, 90*time.Second))

	pickNode := RandomNodePicker(cluster.NodeCount(), rand.NewSource(time.Now().UnixNano()))

	acc1Addr, err := cluster.AccountAddress("acc1")
	require.NoError(t, err)
	_, acc1Bz, err := sdkbech32.DecodeAndConvert(acc1Addr)
	require.NoError(t, err)
	acc1SDK := sdk.AccAddress(acc1Bz)
	acc1VM := movetypes.ConvertSDKAddressToVMAddress(acc1SDK).String()

	modulePath, err := cluster.BuildMoveModule(
		ctx,
		cluster.RepoPath("x", "move", "keeper", "contracts"),
		"TableGenerator",
		map[string]string{"TestAccount": acc1VM},
	)
	require.NoError(t, err)
	publish := cluster.MovePublish(ctx, "acc1", []string{modulePath}, pickNode())
	require.NoError(t, publish.Err)
	require.EqualValues(t, 0, publish.Code)

	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 60*time.Second))

	txPerAccount := 3
	initial := map[string]e2e.AccountMeta{}
	for _, name := range cluster.AccountNames() {
		addr, err := cluster.AccountAddress(name)
		require.NoError(t, err)
		meta, err := cluster.QueryAccountMeta(ctx, pickNode(), addr)
		require.NoError(t, err)
		initial[name] = meta
	}

	queueResults := map[string][]e2e.TxResult{}
	for idx, name := range cluster.AccountNames() {
		meta := initial[name]
		seqs := SequencePattern(meta.Sequence, txPerAccount)

		local := make([]e2e.TxResult, 0, len(seqs))
		for _, seq := range seqs {
			newVMAddr := fmt.Sprintf("0x%064x", uint64((idx+11)*1_000_000)+seq)
			res := cluster.MoveExecuteJSONWithSequence(
				ctx,
				name,
				"0x1",
				"account",
				"create_account_script",
				nil,
				[]string{newVMAddr},
				meta.AccountNumber,
				seq,
				pickNode(),
			)
			local = append(local, res)
		}
		queueResults[name] = local
	}

	for name, txs := range queueResults {
		for i, txr := range txs {
			require.NoError(t, txr.Err, "%s queue tx[%d] err", name, i)
			require.EqualValues(t, 0, txr.Code, "%s queue tx[%d] code=%d txhash=%s raw_log=%s", name, i, txr.Code, txr.TxHash, txr.RawLog)
		}
	}
	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 90*time.Second))

	afterQueue := map[string]e2e.AccountMeta{}
	for _, name := range cluster.AccountNames() {
		addr, err := cluster.AccountAddress(name)
		require.NoError(t, err)
		meta, err := cluster.QueryAccountMeta(ctx, pickNode(), addr)
		require.NoError(t, err)
		afterQueue[name] = meta
		expected := initial[name].Sequence + uint64(txPerAccount)
		require.Equalf(t, expected, meta.Sequence, "%s queue-clear final sequence mismatch", name)
	}

	moduleAddr, err := cluster.AccountAddress("acc1")
	require.NoError(t, err)

	for i, name := range cluster.AccountNames() {
		a := 40 + (i * 10)
		meta := afterQueue[name]
		res := cluster.MoveExecuteJSONWithSequence(
			ctx,
			name,
			moduleAddr,
			"TableGenerator",
			"generate_table",
			nil,
			[]string{fmt.Sprintf("%d", a)},
			meta.AccountNumber,
			meta.Sequence,
			pickNode(),
		)
		require.NoError(t, res.Err)
		require.EqualValues(t, 0, res.Code)
	}

	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 60*time.Second))

	for _, name := range cluster.AccountNames() {
		addr, err := cluster.AccountAddress(name)
		require.NoError(t, err)
		out, err := cluster.MoveQueryResources(ctx, addr, pickNode())
		require.NoError(t, err)
		require.Contains(t, string(out), "TableGenerator::S")
	}
}
