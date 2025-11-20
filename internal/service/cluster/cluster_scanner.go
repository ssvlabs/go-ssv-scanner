package cluster

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"log/slog"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
    "github.com/ssvlabs/go-ssv-scanner/internal/eth"
    "github.com/ssvlabs/go-ssv-scanner/internal/service"
)

// GetLatestClusterSnapshot scans backward to find the latest cluster-affecting event for owner+operatorIds
func GetLatestClusterSnapshot(ctx context.Context, client *ethclient.Client, network eth.NetworkSettings, owner common.Address, operatorIDs []uint64, prog *eth.ScanProgress, l *slog.Logger) (*ClusterResult, error) {
	latest, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	step := service.MonthBlocks
	address := common.HexToAddress(network.ContractAddress)
	// Topic value for indexed address is left-padded to 32 bytes
	ownerTopic := common.BytesToHash(common.LeftPadBytes(owner.Bytes(), 32))

	// Event names of interest (cluster-affecting)
	events := []string{"ClusterDeposited", "ClusterWithdrawn", "ClusterReactivated", "ValidatorRemoved", "ValidatorAdded", "ClusterLiquidated"}
	eventIDs := map[common.Hash]string{}
	eventSigs := make([]common.Hash, 0, len(events))
	for _, name := range events {
		ev := network.ABI.Events[name]
		eventIDs[ev.ID] = name
		eventSigs = append(eventSigs, ev.ID)
	}

	type parsed struct {
		block       uint64
		txIdx       uint
		logIdx      uint
		cluster     ClusterSnapshot
		operatorIds []uint64
	}
	var candidates []parsed

	// Iterate windows backward
	if prog != nil {
		prog.SetName("cluster")
		prog.SetBase(network.GenesisBlock)
		prog.SetTotal(int64(latest) - network.GenesisBlock + 1)
	}
	for start := int64(latest); start > network.GenesisBlock; start -= step {
		end := start
		from := start - step + 1
		if from < network.GenesisBlock {
			from = network.GenesisBlock
		}
		fromBI := big.NewInt(from)
		endBI := big.NewInt(end)
		if prog != nil {
			prog.Update(from, end, int64(latest), step)
		}

		q := ethereum.FilterQuery{
			FromBlock: fromBI,
			ToBlock:   endBI,
			Addresses: []common.Address{address},
			// Restrict topic0 to cluster-affecting events, and topic1 to owner address
			Topics: [][]common.Hash{eventSigs, {ownerTopic}},
		}

		logsBatch, err := service.FilterLogsTimeout(ctx, client, q)
		if err != nil {
			if step == service.MonthBlocks {
				step = service.WeekBlocks
				start += service.WeekBlocks
				continue
			}
			if step == service.WeekBlocks {
				step = service.DayBlocks
				start += service.DayBlocks
				continue
			}
			return nil, fmt.Errorf("query error at %d-%d: %w", from, end, err)
		}
		if l != nil {
			l.Debug("cluster logs fetched", "from", from, "to", end, "count", len(logsBatch))
		}
		if prog != nil {
			size := end - from + 1
			if size > 0 {
				prog.AddDone(size)
			}
		}
		if len(logsBatch) == 0 {
			continue
		}

		for _, lg := range logsBatch {
			name, ok := eventIDs[lg.Topics[0]]
			if !ok {
				continue
			}
			if l != nil {
				l.Debug("event", "name", name, "block", lg.BlockNumber, "tx", lg.TxIndex, "log", lg.Index, "dataLen", len(lg.Data), "topics", len(lg.Topics))
			}
			ops, cl, err := parseEvent(network.ABI, name, lg)
			if err != nil {
				if l != nil {
					l.Debug("parseEvent error", "event", name, "err", err)
				}
				continue
			}
			if l != nil {
				l.Debug("parsed", "ops", ops, "vc", cl.ValidatorCount, "nfi", cl.NetworkFeeIndex, "idx", cl.Index, "active", cl.Active, "bal", cl.Balance)
			}
			// ensure parsed operator ids are sorted for strict comparison
			sort.Slice(ops, func(i, j int) bool { return ops[i] < ops[j] })
			if !equalOperatorIDs(ops, operatorIDs) {
				continue
			}
			if l != nil {
				l.Debug("candidate accepted", "event", name, "block", lg.BlockNumber)
			}
			candidates = append(candidates, parsed{
				block:       lg.BlockNumber,
				txIdx:       uint(lg.TxIndex),
				logIdx:      lg.Index,
				cluster:     cl,
				operatorIds: ops,
			})
		}

		if len(candidates) > 0 {
			// pick latest by block, then tx, then log
			sort.Slice(candidates, func(i, j int) bool {
				a, b := candidates[i], candidates[j]
				if a.block != b.block {
					return a.block > b.block
				}
				if a.txIdx != b.txIdx {
					return a.txIdx > b.txIdx
				}
				return a.logIdx > b.logIdx
			})

			latestCluster := candidates[0]
			payload := map[string]string{
				"Owner":     strings.ToLower(owner.Hex()),
				"Operators": joinUint64(operatorIDs, ","),
				"Block":     fmt.Sprintf("%d", latestCluster.block),
				"Data":      fmt.Sprintf("%d,%s,%s,%t,%s", latestCluster.cluster.ValidatorCount, latestCluster.cluster.NetworkFeeIndex.String(), latestCluster.cluster.Index.String(), latestCluster.cluster.Active, latestCluster.cluster.Balance.String()),
			}
			res := &ClusterResult{Payload: payload, Cluster: latestCluster.cluster}
			return res, nil
		}
	}

	// default empty
	empty := ClusterSnapshot{ValidatorCount: 0, NetworkFeeIndex: big.NewInt(0), Index: big.NewInt(0), Active: true, Balance: big.NewInt(0)}
	payload := map[string]string{
		"Owner":     strings.ToLower(owner.Hex()),
		"Operators": joinUint64(operatorIDs, ","),
		"Block":     fmt.Sprintf("%d", latest),
		"Data":      fmt.Sprintf("%d,%s,%s,%t,%s", empty.ValidatorCount, empty.NetworkFeeIndex.String(), empty.Index.String(), empty.Active, empty.Balance.String()),
	}
	return &ClusterResult{Payload: payload, Cluster: empty}, nil
}

func parseEvent(contractABI abi.ABI, name string, lg types.Log) ([]uint64, ClusterSnapshot, error) {
	// First, try map-based unpack which tends to produce portable shapes
	out := map[string]interface{}{}
	if err := contractABI.UnpackIntoMap(out, name, lg.Data); err == nil {
		var opIDs []uint64
		if v, ok := out["operatorIds"]; ok {
			switch arr := v.(type) {
			case []uint64:
				opIDs = arr
			case []*big.Int:
				opIDs = make([]uint64, len(arr))
				for i, bi := range arr {
					opIDs[i] = bi.Uint64()
				}
			case []interface{}:
				opIDs = make([]uint64, len(arr))
				for i, x := range arr {
					switch t := x.(type) {
					case uint64:
						opIDs[i] = t
					case *big.Int:
						opIDs[i] = t.Uint64()
					}
				}
			}
		}

		var snap ClusterSnapshot
		if c, ok := out["cluster"]; ok {
			switch tup := c.(type) {
			case []interface{}:
				if len(tup) == 5 {
					// validatorCount
					switch v := tup[0].(type) {
					case uint32:
						snap.ValidatorCount = v
					case *big.Int:
						snap.ValidatorCount = uint32(v.Uint64())
					case uint64:
						snap.ValidatorCount = uint32(v)
					}
					// networkFeeIndex
					switch v := tup[1].(type) {
					case *big.Int:
						snap.NetworkFeeIndex = new(big.Int).Set(v)
					case uint64:
						snap.NetworkFeeIndex = new(big.Int).SetUint64(v)
					default:
						snap.NetworkFeeIndex = big.NewInt(0)
					}
					// index
					switch v := tup[2].(type) {
					case *big.Int:
						snap.Index = new(big.Int).Set(v)
					case uint64:
						snap.Index = new(big.Int).SetUint64(v)
					default:
						snap.Index = big.NewInt(0)
					}
					// active
					if vb, ok := tup[3].(bool); ok {
						snap.Active = vb
					}
					// balance
					if vbi, ok := tup[4].(*big.Int); ok {
						snap.Balance = new(big.Int).Set(vbi)
					} else {
						snap.Balance = big.NewInt(0)
					}
				}
			case map[string]interface{}:
				if v, ok := tup["validatorCount"]; ok {
					switch vv := v.(type) {
					case uint32:
						snap.ValidatorCount = vv
					case *big.Int:
						snap.ValidatorCount = uint32(vv.Uint64())
					case uint64:
						snap.ValidatorCount = uint32(vv)
					}
				}
				if v, ok := tup["networkFeeIndex"]; ok {
					switch vv := v.(type) {
					case *big.Int:
						snap.NetworkFeeIndex = new(big.Int).Set(vv)
					case uint64:
						snap.NetworkFeeIndex = new(big.Int).SetUint64(vv)
					default:
						snap.NetworkFeeIndex = big.NewInt(0)
					}
				}
				if v, ok := tup["index"]; ok {
					switch vv := v.(type) {
					case *big.Int:
						snap.Index = new(big.Int).Set(vv)
					case uint64:
						snap.Index = new(big.Int).SetUint64(vv)
					default:
						snap.Index = big.NewInt(0)
					}
				}
				if v, ok := tup["active"]; ok {
					if b, ok := v.(bool); ok {
						snap.Active = b
					}
				}
				if v, ok := tup["balance"]; ok {
					if vbi, ok := v.(*big.Int); ok {
						snap.Balance = new(big.Int).Set(vbi)
					} else {
						snap.Balance = big.NewInt(0)
					}
				}
			default:
				if snap2, ok := decodeClusterAny(tup); ok {
					snap = snap2
				}
			}
		}

		if snap.NetworkFeeIndex == nil {
			snap.NetworkFeeIndex = big.NewInt(0)
		}
		if snap.Index == nil {
			snap.Index = big.NewInt(0)
		}
		if snap.Balance == nil {
			snap.Balance = big.NewInt(0)
		}
		return opIDs, snap, nil
	}

	// Fallback: use NonIndexed unpack
	ev, ok := contractABI.Events[name]
	if !ok {
		return nil, ClusterSnapshot{}, fmt.Errorf("event not found: %s", name)
	}
	vals, err := ev.Inputs.NonIndexed().Unpack(lg.Data)
	if err != nil {
		return nil, ClusterSnapshot{}, err
	}
	var opIdx, clusterIdx = -1, -1
	for i, arg := range ev.Inputs {
		if arg.Indexed {
			continue
		}
		if arg.Name == "operatorIds" && opIdx == -1 {
			opIdx = indexOfNonIndexed(ev.Inputs, i)
		}
		if arg.Name == "cluster" && clusterIdx == -1 {
			clusterIdx = indexOfNonIndexed(ev.Inputs, i)
		}
	}
	var opIDs []uint64
	if opIdx >= 0 && opIdx < len(vals) {
		switch arr := vals[opIdx].(type) {
		case []uint64:
			opIDs = arr
		case []*big.Int:
			opIDs = make([]uint64, len(arr))
			for i, bi := range arr {
				opIDs[i] = bi.Uint64()
			}
		case []interface{}:
			opIDs = make([]uint64, len(arr))
			for i, v := range arr {
				switch vv := v.(type) {
				case uint64:
					opIDs[i] = vv
				case *big.Int:
					opIDs[i] = vv.Uint64()
				}
			}
		}
	}
	var snap ClusterSnapshot
	if clusterIdx >= 0 && clusterIdx < len(vals) {
		if c, ok := vals[clusterIdx].([]interface{}); ok && len(c) == 5 {
			switch v := c[0].(type) {
			case uint32:
				snap.ValidatorCount = v
			case *big.Int:
				snap.ValidatorCount = uint32(v.Uint64())
			case uint64:
				snap.ValidatorCount = uint32(v)
			}
			switch v := c[1].(type) {
			case *big.Int:
				snap.NetworkFeeIndex = new(big.Int).Set(v)
			case uint64:
				snap.NetworkFeeIndex = new(big.Int).SetUint64(v)
			default:
				snap.NetworkFeeIndex = big.NewInt(0)
			}
			switch v := c[2].(type) {
			case *big.Int:
				snap.Index = new(big.Int).Set(v)
			case uint64:
				snap.Index = new(big.Int).SetUint64(v)
			default:
				snap.Index = big.NewInt(0)
			}
			if b, ok := c[3].(bool); ok {
				snap.Active = b
			}
			if vbi, ok := c[4].(*big.Int); ok {
				snap.Balance = new(big.Int).Set(vbi)
			} else {
				snap.Balance = big.NewInt(0)
			}
		} else if snap2, ok := decodeClusterAny(vals[clusterIdx]); ok {
			snap = snap2
		}
	}
	if snap.NetworkFeeIndex == nil {
		snap.NetworkFeeIndex = big.NewInt(0)
	}
	if snap.Index == nil {
		snap.Index = big.NewInt(0)
	}
	if snap.Balance == nil {
		snap.Balance = big.NewInt(0)
	}
	return opIDs, snap, nil
}
