package cluster

import "math/big"

type ClusterSnapshot struct {
	ValidatorCount  uint32
	NetworkFeeIndex *big.Int
	Index           *big.Int
	Active          bool
	Balance         *big.Int
	// EffectiveBalance is the balance after applying fees as of the eth_call block.
	// If the underlying call fails, it falls back to Balance.
	EffectiveBalance *big.Int
}

type ClusterResult struct {
	Payload map[string]string
	Cluster ClusterSnapshot
}
