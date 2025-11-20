package config

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
)

// CommandType represents a supported CLI subcommand
type CommandType string

const (
	CmdNone     CommandType = ""
	CmdCluster  CommandType = "cluster"
	CmdNonce    CommandType = "nonce"
	CmdOperator CommandType = "operator"
)

// CLIArgs captures parsed CLI arguments for all subcommands
type CLIArgs struct {
	Command      CommandType
	Network      string
	NodeURL      string
	OwnerAddress string
	OperatorIDs  []uint64
	OutputPath   string
	JSON         bool
	Logger       *slog.Logger
	Debug        bool
}

// ParseCLI is a convenience wrapper around ParseCLIArgs using os.Args
func ParseCLI() (*CLIArgs, error) {
	return ParseCLIArgs(os.Args[1:])
}

// ParseCLIArgs parses arguments into a structured CLIArgs object.
// Supported commands: cluster, nonce, operator
// Flags (by command):
//
//	-nw, --network       (required for all)
//	-n,  --node-url      (required for all)
//	-oa, --owner-address (required for all)
//	-oids, --operator-ids (cluster only, comma-separated)
//	-o, --output-path    (operator only)
//	--json               (optional pretty vs json output switch, reserved)
func ParseCLIArgs(args []string) (*CLIArgs, error) {
	if len(args) == 0 {
		printGlobalUsage()
		return nil, errors.New("no command provided")
	}

	cmd := CommandType(strings.ToLower(args[0]))
	switch cmd {
	case CmdCluster:
		return parseClusterArgs(args[1:])
	case CmdNonce:
		return parseNonceArgs(args[1:])
	case CmdOperator:
		return parseOperatorArgs(args[1:])
	case "-h", "--help", "help":
		printGlobalUsage()
		os.Exit(0)
	default:
		printGlobalUsage()
		return nil, fmt.Errorf("unknown command: %s", args[0])
	}
	// Unreachable, but keeps the compiler happy on some static paths
	return nil, fmt.Errorf("invalid usage")
}

func parseCommonFlags(fs *flag.FlagSet, network, nodeURL, owner *string, json, debug *bool) {
	fs.StringVar(network, "nw", "", "The network (mainnet, hoodi, hoodi_stage, local_testnet)")
	fs.StringVar(network, "network", "", "The network (mainnet, hoodi, hoodi_stage, local_testnet)")
	fs.StringVar(nodeURL, "n", "", "ETH1 (execution client) node endpoint url")
	fs.StringVar(nodeURL, "node-url", "", "ETH1 (execution client) node endpoint url")
	fs.StringVar(owner, "oa", "", "Cluster owner address (0x...)")
	fs.StringVar(owner, "owner-address", "", "Cluster owner address (0x...)")
	fs.BoolVar(json, "json", false, "Emit JSON-only output")
	fs.BoolVar(debug, "debug", false, "Enable debug logging")
}

func parseClusterArgs(args []string) (*CLIArgs, error) {
	fs := flag.NewFlagSet(string(CmdCluster), flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, clusterUsage()) }

	var network, nodeURL, owner string
	var json, debug bool
	parseCommonFlags(fs, &network, &nodeURL, &owner, &json, &debug)
	var operatorIDsCSV string
	fs.StringVar(&operatorIDsCSV, "oids", "", "Comma-separated operator IDs (3f+1)")
	fs.StringVar(&operatorIDsCSV, "operator-ids", "", "Comma-separated operator IDs (3f+1)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if err := validateNetwork(network); err != nil {
		return nil, err
	}
	if err := validateOwner(owner); err != nil {
		return nil, err
	}
	if nodeURL == "" {
		return nil, errors.New("node-url is required")
	}
	if operatorIDsCSV == "" {
		return nil, errors.New("operator-ids are required")
	}

	ids, err := parseOperatorIDs(operatorIDsCSV)
	if err != nil {
		return nil, err
	}
	if !isValid3fPlus1(len(ids)) {
		return nil, errors.New("operator-ids length must satisfy 3f+1 (4..13 and len%3==1)")
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	return &CLIArgs{
		Command:      CmdCluster,
		Network:      strings.ToLower(network),
		NodeURL:      nodeURL,
		OwnerAddress: strings.ToLower(owner),
		OperatorIDs:  ids,
		JSON:         json,
		Debug:        debug,
	}, nil
}

func parseNonceArgs(args []string) (*CLIArgs, error) {
	fs := flag.NewFlagSet(string(CmdNonce), flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, nonceUsage()) }

	var network, nodeURL, owner string
	var json, debug bool
	parseCommonFlags(fs, &network, &nodeURL, &owner, &json, &debug)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if err := validateNetwork(network); err != nil {
		return nil, err
	}
	if err := validateOwner(owner); err != nil {
		return nil, err
	}
	if nodeURL == "" {
		return nil, errors.New("node-url is required")
	}

	return &CLIArgs{
		Command:      CmdNonce,
		Network:      strings.ToLower(network),
		NodeURL:      nodeURL,
		OwnerAddress: strings.ToLower(owner),
		JSON:         json,
		Debug:        debug,
	}, nil
}

func parseOperatorArgs(args []string) (*CLIArgs, error) {
	fs := flag.NewFlagSet(string(CmdOperator), flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, operatorUsage()) }

	var network, nodeURL, owner string
	var json, debug bool
	parseCommonFlags(fs, &network, &nodeURL, &owner, &json, &debug)
	var output string
	fs.StringVar(&output, "o", "", "Output directory for operator pubkeys")
	fs.StringVar(&output, "output-path", "", "Output directory for operator pubkeys")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if err := validateNetwork(network); err != nil {
		return nil, err
	}
	if err := validateOwner(owner); err != nil {
		return nil, err
	}
	if nodeURL == "" {
		return nil, errors.New("node-url is required")
	}

	return &CLIArgs{
		Command:      CmdOperator,
		Network:      strings.ToLower(network),
		NodeURL:      nodeURL,
		OwnerAddress: strings.ToLower(owner),
		OutputPath:   output,
		JSON:         json,
		Debug:        debug,
	}, nil
}

func parseOperatorIDs(csv string) ([]uint64, error) {
	parts := strings.Split(csv, ",")
	ids := make([]uint64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid operator id: %s", p)
		}
		ids = append(ids, v)
	}
	if len(ids) == 0 {
		return nil, errors.New("no operator ids provided")
	}
	return ids, nil
}

func isValid3fPlus1(n int) bool {
	if n < 4 || n > 13 {
		return false
	}
	return n%3 == 1
}

func validateOwner(owner string) error {
	if owner == "" {
		return errors.New("owner-address is required")
	}
	if !strings.HasPrefix(owner, "0x") || len(owner) != 42 {
		return errors.New("invalid owner address")
	}
	return nil
}

func validateNetwork(nw string) error {
	switch strings.ToLower(nw) {
	case "mainnet", "hoodi", "hoodi_stage", "local_testnet":
		return nil
	default:
		return fmt.Errorf("unsupported network: %s (supported: mainnet, hoodi, hoodi_stage, local_testnet)", nw)
	}
}

func printGlobalUsage() { fmt.Fprint(os.Stderr, GlobalUsage()) }

// GlobalUsage returns a short usage banner for the CLI (exported for main).
func GlobalUsage() string {
	return `Usage: ssv-go-scanner <command> [options]

Commands:
  cluster    Get latest cluster snapshot
  nonce      Get current owner nonce
  operator   Export operator pubkeys to JSON

Run '<command> -h' for command-specific options.
`
}

func clusterUsage() string {
	return `Usage: ssv-go-scanner cluster -n <node-url> -nw <network> -oa <owner> -oids <id1,id2,...> [-json] [-debug]

Options:
  -n,  --node-url         ETH1 JSON-RPC endpoint
  -nw, --network          Network (mainnet, hoodi, hoodi_stage, local_testnet)
  -oa, --owner-address    Cluster owner address (0x...)
  -oids, --operator-ids   Comma-separated operator IDs (3f+1)
  -json                   Emit JSON-only output
  -debug                  Enable debug logging
`
}

func nonceUsage() string {
	return `Usage: ssv-go-scanner nonce -n <node-url> -nw <network> -oa <owner> [-json] [-debug]

Options:
  -n,  --node-url         ETH1 JSON-RPC endpoint
  -nw, --network          Network (mainnet, hoodi, hoodi_stage, local_testnet)
  -oa, --owner-address    Cluster owner address (0x...)
  -json                   Emit JSON-only output
  -debug                  Enable debug logging
`
}

func operatorUsage() string {
	return `Usage: ssv-go-scanner operator -n <node-url> -nw <network> -oa <owner> [-o <output-dir>] [-json] [-debug]

Options:
  -n,  --node-url         ETH1 JSON-RPC endpoint
  -nw, --network          Network (mainnet, hoodi, hoodi_stage, local_testnet)
  -oa, --owner-address    Owner address (0x...)
  -o,  --output-path      Output directory for JSON
  -json                   Emit JSON-only output
  -debug                  Enable debug logging
`
}
