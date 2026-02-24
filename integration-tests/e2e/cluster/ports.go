package cluster

import "fmt"

type NodePorts struct {
	P2P  int
	RPC  int
	GRPC int
	API  int
}

func allocatePorts(index, basePort, stride int) (NodePorts, error) {
	if index < 0 {
		return NodePorts{}, fmt.Errorf("invalid node index: %d", index)
	}
	if basePort <= 0 {
		return NodePorts{}, fmt.Errorf("invalid base port: %d", basePort)
	}
	if stride < 10 {
		return NodePorts{}, fmt.Errorf("port stride must be >= 10, got %d", stride)
	}

	start := basePort + (index * stride)
	return NodePorts{
		P2P:  start,
		RPC:  start + 1,
		GRPC: start + 2,
		API:  start + 3,
	}, nil
}
