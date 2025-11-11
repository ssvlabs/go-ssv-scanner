package scanner

import (
	"log/slog"
	"time"

    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ssvlabs/go-ssv-scanner/internal/eth"
)

// Config defines initialization parameters for Scanner.
type Config struct {
	// Required
	Network string // mainnet | hoodi | hoodi_stage | local_testnet
	NodeURL string // ETH1 JSON-RPC endpoint

	// Optional
	Logger *slog.Logger // if nil, logging is disabled inside the scanner package
	// Optional: emit periodic progress logs to Logger (info level)
	EnableProgress   bool
	ProgressInterval time.Duration
}

// Request is a generic request for Scanner.Scan.
// Kind values: "nonce", "cluster", "operator".
type Request struct {
	Kind        string
	Owner       string   // 0x... (required for all kinds)
	OperatorIDs []uint64 // cluster only
	OutputPath  string   // operator only; optional output dir
}

// Scanner provides a reusable programmatic API for SSV scanning.
// It wraps the existing service packages and manages the network settings and client.
type Scanner struct {
	cfg     Config
	client  *ethclient.Client
	network *eth.NetworkSettings
	logger  *slog.Logger
}

// NonceResult is returned by Scanner.Scan when Kind == "nonce".
type NonceResult struct {
	Owner string
	Nonce int
}

// OperatorResult is returned by Scanner.Scan when Kind == "operator".
type OperatorResult struct {
	File  string
	Count int
}
