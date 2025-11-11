package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
    "github.com/ssvlabs/go-ssv-scanner/config"
    "github.com/ssvlabs/go-ssv-scanner/internal/eth"
    "github.com/ssvlabs/go-ssv-scanner/internal/service/cluster"
    "github.com/ssvlabs/go-ssv-scanner/internal/service/nonce"
    "github.com/ssvlabs/go-ssv-scanner/internal/service/operator"

    appLogger "github.com/ssvlabs/go-ssv-scanner/internal/logger"
)

func main() {
	args, err := config.ParseCLI()
	if err != nil {
		// Ensure meaningful error output and a usage hint
		fmt.Fprintln(os.Stderr, "CLI error:", err)
		fmt.Fprint(os.Stderr, config.GlobalUsage())
		os.Exit(2)
	}

	// UI banner for consistent CLI experience
	appLogger.PrintScannerBanner()

	// Initialize structured logger (stderr + file) and set on args
	l, err := appLogger.GlobalInit()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	args.Logger = l
	if args.Debug {
		appLogger.SetLogLevel(l, slog.LevelDebug)
		l.Debug("debug mode enabled")
	}

	// Optionally load .env for network address overrides
	_ = godotenv.Load()

	// Resolve network settings and connect
	netCfg, err := eth.GetContractSettings(args.Network)
	if err != nil {
		log.Fatalf("%v", err)
	}

	client, err := eth.ConnectHTTP(args.NodeURL)
	if err != nil {
		log.Fatalf("failed to connect to node: %v", err)
	}

	ctx := context.Background()
	owner := common.HexToAddress(args.OwnerAddress)

	switch args.Command {
	case config.CmdNonce:
		appLogger.PrintBanner("🔢 Nonce Scan")
		evID := netCfg.ABI.Events["ValidatorAdded"].ID
		prog := &eth.ScanProgress{}
		// Start central progress ticker at Info level unconditionally
		stop := startProgressTicker(ctx, l, prog)
		n, err := nonce.ScanNonce(ctx, client, common.HexToAddress(netCfg.ContractAddress), owner, netCfg.GenesisBlock, evID, prog)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if stop != nil {
			stop()
		}
		if args.JSONOnly {
			out := map[string]interface{}{"owner": strings.ToLower(owner.Hex()), "nonce": n}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(out)
			appLogger.PrintKeyValueTable("Nonce Result", [][2]string{
				{"Network", netCfg.Name},
				{"Owner", strings.ToLower(owner.Hex())},
				{"Nonce", fmt.Sprintf("%d", n)},
			})
			appLogger.LogNonce(l, strings.ToLower(owner.Hex()), n)
		} else {
			appLogger.PrintKeyValueTable("Nonce Result", [][2]string{
				{"Network", netCfg.Name},
				{"Owner", strings.ToLower(owner.Hex())},
				{"Nonce", fmt.Sprintf("%d", n)},
			})
		}

	case config.CmdCluster:
		appLogger.PrintBanner("📦 Cluster Scanner")
		prog := &eth.ScanProgress{}
		// Start central progress ticker at Info level unconditionally
		stop := startProgressTicker(ctx, l, prog)
		res, err := cluster.GetLatestClusterSnapshot(ctx, client, *netCfg, owner, args.OperatorIDs, prog, l)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if stop != nil {
			stop()
		}
		// Always print summary and details tables to stderr
		appLogger.PrintKeyValueTable("Cluster Snapshot", [][2]string{
			{"Network", netCfg.Name},
			{"Owner", res.Payload["Owner"]},
			{"Operators", res.Payload["Operators"]},
			{"Block", res.Payload["Block"]},
			{"Data", res.Payload["Data"]},
		})
		appLogger.PrintKeyValueTable("Cluster snapshot:", [][2]string{
			{"validatorCount", fmt.Sprintf("%d", res.Cluster.ValidatorCount)},
			{"networkFeeIndex", res.Cluster.NetworkFeeIndex.String()},
			{"index", res.Cluster.Index.String()},
			{"active", fmt.Sprintf("%t", res.Cluster.Active)},
			{"balance", res.Cluster.Balance.String()},
		})

		// If JSON requested, also print structured JSON to stdout and log
		if args.JSONOnly {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]interface{}{
				"block":            res.Payload["Block"],
				"cluster snapshot": res.Cluster,
				"cluster":          []interface{}{res.Cluster.ValidatorCount, res.Cluster.NetworkFeeIndex, res.Cluster.Index, res.Cluster.Active, res.Cluster.Balance},
			})
			appLogger.LogCluster(l, res.Payload["Owner"], res.Payload["Block"], res.Payload["Operators"], res.Payload["Data"])
		}

	case config.CmdOperator:
		appLogger.PrintBanner("🧩 Operator Scanner")
		prog := &eth.ScanProgress{}
		// Start central progress ticker at Info level unconditionally
		stop := startProgressTicker(ctx, l, prog)
		path, count, err := operator.ExportOperatorPubkeys(ctx, client, *netCfg, args.OutputPath, prog)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if stop != nil {
			stop()
		}
		if args.JSONOnly {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]interface{}{"file": path, "count": count})
			appLogger.LogOperatorSaved(l, path, count)
		} else {
			fmt.Printf("\nOperator data has been saved to:\n %s\n", path)
		}
	default:
		log.Fatalf("unknown command")
	}

	// give time for output flush in some terminals
	time.Sleep(100 * time.Millisecond)
}
