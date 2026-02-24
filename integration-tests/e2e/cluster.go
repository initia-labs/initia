package e2e

import (
	"context"
	"testing"

	e2ecluster "github.com/initia-labs/initia/integration-tests/e2e/cluster"
)

type ClusterOptions = e2ecluster.ClusterOptions
type NodePorts = e2ecluster.NodePorts
type Node = e2ecluster.Node
type AccountMeta = e2ecluster.AccountMeta
type TxResult = e2ecluster.TxResult
type Cluster = e2ecluster.Cluster

const maxNodeCount = e2ecluster.MaxNodeCount

func NewCluster(ctx context.Context, t *testing.T, opts ClusterOptions) (*Cluster, error) {
	return e2ecluster.NewCluster(ctx, t, opts)
}
