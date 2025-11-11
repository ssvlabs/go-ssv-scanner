package operator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
    "github.com/ssvlabs/go-ssv-scanner/internal/eth"
    "github.com/ssvlabs/go-ssv-scanner/internal/service"
)

// ExportOperatorPubkeys scans OperatorAdded events and writes JSON file, returning the path.
func ExportOperatorPubkeys(ctx context.Context, client *ethclient.Client, network eth.NetworkSettings, outputDir string, prog *eth.ScanProgress) (string, int, error) {
	latest, err := client.BlockNumber(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	address := common.HexToAddress(network.ContractAddress)
	ev := network.ABI.Events["OperatorAdded"]

	// collect all logs forward
	step := service.MonthBlocks
	from := big.NewInt(network.GenesisBlock)
	if prog != nil {
		prog.SetName("operator")
		prog.SetBase(network.GenesisBlock)
		prog.SetTotal(int64(latest) - network.GenesisBlock + 1)
		prog.Update(from.Int64(), int64(latest), int64(latest), service.MonthBlocks)
	}
	var entries []OperatorEntry

	for from.Int64() <= int64(latest) {
		to := new(big.Int).Add(from, big.NewInt(step-1))
		if to.Int64() > int64(latest) {
			to = new(big.Int).SetUint64(latest)
		}

		q := ethereum.FilterQuery{
			FromBlock: from,
			ToBlock:   to,
			Addresses: []common.Address{address},
			Topics:    [][]common.Hash{{ev.ID}},
		}
		if prog != nil {
			prog.Update(from.Int64(), to.Int64(), int64(latest), step)
		}
		logs, err := service.FilterLogsTimeout(ctx, client, q)
		if err != nil {
			if step == service.MonthBlocks {
				step = service.WeekBlocks
				continue
			}
			if step == service.WeekBlocks {
				step = service.DayBlocks
				continue
			}
			return "", 0, err
		}

		for _, lg := range logs {
			// Extract the operatorId from indexed topic[1]
			// (topic[0] is the event signature, topic[1] = operatorId, topic[2] = owner)
			opID := new(big.Int).SetBytes(lg.Topics[1].Bytes()).Uint64()

			// Unpack non-indexed data for OperatorAdded (publicKey bytes, fee)
			out := map[string]interface{}{}
			if err := ev.Inputs.NonIndexed().UnpackIntoMap(out, lg.Data); err != nil {
				// fallback to ABI-level unpack in case of library differences
				if err := network.ABI.UnpackIntoMap(out, "OperatorAdded", lg.Data); err != nil {
					continue
				}
			}

			// Normalize publicKey to base64 string (matches TS scanner output)
			var pk string
			if b, ok := out["publicKey"].([]byte); ok {
				// Some providers/libraries may yield ABI-encoded dynamic bytes (head+len+data).
				// Normalize to the raw bytes payload if we detect that shape.
				nb := normalizeDynamicBytes(b)
				// If the payload already looks like a base64-encoded PEM header ("LS0t" -> "---"), keep as-is.
				if isBase64PEM(nb) {
					pk = string(nb)
				} else {
					pk = base64.StdEncoding.EncodeToString(nb)
				}
			} else {
				// Best-effort: if the ABI decoder returned something else, stringify
				pk = fmt.Sprintf("%v", out["publicKey"])
			}

			entries = append(entries, OperatorEntry{ID: opID, Pubkey: pk})
		}

		if prog != nil {
			size := to.Int64() - from.Int64() + 1
			if size > 0 {
				prog.AddDone(size)
			}
		}
		from = new(big.Int).Add(to, big.NewInt(1))
		if prog != nil {
			prog.Update(from.Int64(), to.Int64(), int64(latest), step)
		}
	}

	// ensure output dir
	if outputDir == "" {
		outputDir = filepath.Join("dist", "data")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", 0, err
	}

	filePath := filepath.Join(outputDir, fmt.Sprintf("operator-pubkeys-%s.json", network.Name))
	data, _ := json.MarshalIndent(entries, "", "  ")
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return "", 0, err
	}
	return filePath, len(entries), nil
}
