<div align="center">

# SSV Scanner (Go)

<p>
CLI tool for retrieving <strong>SSV cluster snapshots</strong>, <strong>owner nonce</strong>, and <strong>operator pubkeys</strong> from the SSV Network contracts.<br>
Targets feature parity with the official JS tool at <a href="https://github.com/ssvlabs/ssv-scanner">ssvlabs/ssv-scanner</a>, with Go-centric concurrency and structured logging.
</p>

</div>

## Features

- Cluster snapshot: Latest cluster state for a given owner and operator IDs.
- Owner nonce: Count of `ValidatorAdded` events for an owner.
- Operator pubkeys: Export all `OperatorAdded` public keys to JSON.
- Network presets: `mainnet`, `hoodi`, `hoodi_stage`, `local_testnet`.
- Robust scanning: Windowed pagination with fallback (Month → Week → Day).
- Structured logging: JSON logs via `slog` to stderr and timestamped file.

## Requirements

- Go 1.22+
- Access to an Ethereum JSON-RPC endpoint for the chosen network.

## Build

```bash
go build -o ./cmd/bin/ssv-go-scanner ./cmd
# or
make build
```

Then run the binary directly, or use the Make targets below.

## Usage

Commands mirror the upstream scanner and accept the same flags.

Common flags
- `-n, --node-url`: ETH1 JSON-RPC endpoint (required)
- `-nw, --network`: `mainnet`, `hoodi`, `hoodi_stage`, or `local_testnet` (required)
- `-oa, --owner-address`: 0x-prefixed EOA (required)
- `-json`: Emit JSON to stdout; human-readable messages go to stderr
- `-debug`: Enable debug logging (stderr and log file)

### Nonce

Get the current nonce for an owner (number of `ValidatorAdded` for the owner):

```bash
./cmd/bin/ssv-go-scanner nonce -nw mainnet -n https://mainnet.node -oa 0xYourOwner
# or via Make
make nonce NW=mainnet NODE_URL=https://mainnet.node OA=0xYourOwner
```

Output
- Always: prints a summary table to stderr.
- With `-json`: also prints `{ "owner": ..., "nonce": ... }` to stdout and a structured log to stderr.

### Cluster

Get the latest cluster snapshot for an owner and operator IDs (must satisfy 3f+1 − length in [4..13] and `len%3==1`).

```bash
./cmd/bin/ssv-go-scanner cluster -nw mainnet -n https://mainnet.node -oa 0xYourOwner -oids 1,2,3,4
# or via Make
make cluster NW=mainnet NODE_URL=https://mainnet.node OA=0xYourOwner OIDS=1,2,3,4
```

Output
- Always: prints summary and detailed tables to stderr.
- With `-json`: also prints a JSON object to stdout with `block`, `cluster snapshot` (structured), and `cluster` (tuple-like array), and logs a structured summary.

### Operator Pubkeys

Export all operator pubkeys (`OperatorAdded`) to a JSON file.

```bash
./cmd/bin/ssv-go-scanner operator -nw mainnet -n https://mainnet.node -oa 0xYourOwner -o ./out
# or via Make
make operator NW=mainnet NODE_URL=https://mainnet.node OA=0xYourOwner OUT=./out
```

Output
- Writes `operator-pubkeys-<network>.json` to the specified output directory (default: `dist/data`).
- Without `-json`: prints the file path to stdout.
- With `-json`: prints `{ "file": <path>, "count": <n> }` to stdout and logs a structured summary.

## Library Usage

Use this as a module in other Go apps via `pkg/scanner`.

Install

```bash
go get github.com/yoaz-ssvlabs/go-ssv-scanner@latest
```

Import

```go
import scanner "github.com/yoaz-ssvlabs/go-ssv-scanner/pkg/scanner"
```

Quick Start

```go
ctx := context.Background()

// Optional: set up a logger and enable periodic progress logs to it
// Example uses JSON to stderr at info level
l := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

sc, err := scanner.NewScanner(scanner.Config{
  Network:          "mainnet",                 // mainnet|hoodi|hoodi_stage|local_testnet
  NodeURL:          "https://your-exec-node",  // required
  Logger:           l,                          // optional; used for progress and debug logs
  EnableProgress:   true,                       // optional; emit periodic progress logs
  ProgressInterval: 2 * time.Second,            // optional; default 2s when zero
})
if err != nil { panic(err) }

// 1) Nonce (number of ValidatorAdded for owner)
n, err := sc.Nonce(ctx, "0xOwner...")
if err != nil { panic(err) }
fmt.Println("nonce:", n)

// 2) Cluster snapshot (latest snapshot affecting given operator IDs)
cr, err := sc.Cluster(ctx, "0xOwner...", []uint64{1,2,3,4})
if err != nil { panic(err) }
fmt.Printf("cluster at block %s: %+v\n", cr.Payload["Block"], cr.Cluster)

// 3) Operator pubkeys (write JSON to disk)
file, count, err := sc.OperatorPubkeys(ctx, "./out")
if err != nil { panic(err) }
fmt.Println("operators saved:", file, "count:", count)

// 4) Generic Scan API (optional convenience)
out, err := sc.Scan(ctx, scanner.Request{Kind: "operator", Owner: "0xOwner...", OutputPath: "./out"})
if err != nil { panic(err) }
op := out.(scanner.OperatorResult)
fmt.Println("operator file:", op.File, "count:", op.Count)
```

API Reference

- `type Config`
  - `Network`: one of `mainnet`, `hoodi`, `hoodi_stage`, `local_testnet` (required)
  - `NodeURL`: ETH1 JSON-RPC endpoint URL (required)
- `Logger`: optional `*slog.Logger` used by the scanner; progress logging is included only if `EnableProgress` is true
 - `EnableProgress`: when true and `Logger` is set, emits periodic "scan progress" logs at info level
 - `ProgressInterval`: interval for progress logs (default 2s when zero)

- `func NewScanner(cfg Config) (*Scanner, error)`
  - Initializes network settings and dials the ETH client.

- `func (s *Scanner) Nonce(ctx context.Context, ownerHex string) (int, error)`
  - Returns count of `ValidatorAdded` events for `ownerHex`.

- `func (s *Scanner) Cluster(ctx context.Context, ownerHex string, operatorIDs []uint64) (*cluster.ClusterResult, error)`
  - Scans backwards and returns the latest cluster snapshot affecting the given operator IDs.

- `func (s *Scanner) OperatorPubkeys(ctx context.Context, outputDir string) (string, int, error)`
  - Exports all `OperatorAdded` pubkeys to JSON; returns `(filePath, count)`.

- `type Request`
  - `Kind`: `"nonce" | "cluster" | "operator"`
  - `Owner`: 0x-prefixed address (required)
  - `OperatorIDs`: cluster-only
  - `OutputPath`: operator-only

- `func (s *Scanner) Scan(ctx context.Context, req Request) (any, error)`
  - Returns one of: `NonceResult`, `*cluster.ClusterResult`, or `OperatorResult`.

Types

```go
type NonceResult struct { Owner string; Nonce int }
type OperatorResult struct { File string; Count int }
```

Notes
- The library does not print banners. Progress logs are opt-in via `Logger` + `EnableProgress`.
- If you need structured logs, provide a `*slog.Logger` in `scanner.Config`.

## Logging

The CLI initializes a `slog` JSON logger that writes to stderr and a timestamped file under `./logs/<timestamp>/<timestamp>.log`. This ensures stdout remains clean for JSON outputs when `-json` is used.

Helper logging functions (in `logger` package) emit structured entries for progress and results, e.g., `next nonce`, `cluster snapshot`, and `operator data saved`.

## Networks

- `mainnet` – contract: `0xDD9BC35aE942eF0cFa76930954a156B3fF30a4E1`, genesis: `17507487`.
- `hoodi` – contract: `0x58410Bef803ECd7E63B23664C586A6DB72DAf59c`, genesis: `1065`.
- `hoodi_stage` – contract: `0x0aaace4e8affc47c6834171c88d342a4abd8f105`, genesis: `101653`.
- `local_testnet` – contract: `0xBFfF570853d97636b78ebf262af953308924D3D8`, genesis: `0`.

Env overrides (optional)
- `SSV_ADDRESS_MAINNET`, `SSV_GENESIS_MAINNET`
- `SSV_ADDRESS_HOODI`, `SSV_GENESIS_HOODI`
- `SSV_ADDRESS_HOODI_STAGE`, `SSV_GENESIS_HOODI_STAGE`
- `SSV_ADDRESS_LOCAL_TESTNET`, `SSV_GENESIS_LOCAL_TESTNET`

## Notes

- Queries use windowed ranges with automatic fallback on provider errors (Month → Week → Day). For highly rate-limited endpoints, reduce window size or add retries.
- Event decoding uses a minimal event-only ABI embedded from `internal/eth/abi/ssv_events.abi.json`.
  Network presets and ABI live under the `internal/eth` package.
- By default, the latest block is used

## Development

- Lint: `make lint` (runs `go vet`; runs `golangci-lint` if available)
- Build: `make build` or `go build ./...`
- Run: use the Make targets for convenience.