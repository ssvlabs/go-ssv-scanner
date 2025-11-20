package cluster

import "math/big"

type ClusterSnapshot struct {
	ValidatorCount  uint32
	NetworkFeeIndex *big.Int
	Index           *big.Int
	Active          bool
	Balance         *big.Int
}

type ClusterResult struct {
	Payload map[string]string
	Cluster ClusterSnapshot
}
