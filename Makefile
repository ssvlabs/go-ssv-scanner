APP=go-ssv-scanner

.DEFAULT_GOAL := help
.PHONY: help build run nonce cluster operator lint

help:
	@echo "═══════════════════════════════════════════════════════════════════════════════════════════════════════"
	@echo "                                       🔎  SSV SCANNER (GO)  🔎                                        "
	@echo "═══════════════════════════════════════════════════════════════════════════════════════════════════════"
	@echo "  build                                                          - Build the CLI binary"
	@echo "  nonce NW=<net> NODE_URL=<url> OA=<0x..> [ARGS=...]              - Get owner nonce"
	@echo "  cluster NW=<net> NODE_URL=<url> OA=<0x..> OIDS=<csv> [ARGS=...] - Get latest cluster snapshot"
	@echo "  operator NW=<net> NODE_URL=<url> OA=<0x..> [OUT=dir] [ARGS=...] - Export operator pubkeys to JSON"
	@echo "  lint                                                           - Run basic linters (go vet; golangci-lint if installed)"
	@echo ""
	@echo "  Variables:"
	@echo "    NW         - Network (mainnet, hoodi, hoodi_stage, local_testnet)"
	@echo "    NODE_URL   - ETH1 JSON-RPC endpoint"
	@echo "    OA         - Owner address (0x...)"
	@echo "    OIDS       - Operator IDs CSV (e.g., 1,2,3,4)"
	@echo "    OUT        - Output directory for operator pubkeys"
	@echo "    ARGS       - Extra CLI args (e.g., --json --debug)"
	@echo ""
	@echo "═══════════════════════════════════════════════════════════════════════════════════════════════════════"
	@echo "                                       💡  USAGE EXAMPLES  💡                                         "
	@echo "═══════════════════════════════════════════════════════════════════════════════════════════════════════"
	@echo "  make nonce   NW=mainnet NODE_URL=https://mainnet.node OA=0x... ARGS=-debug"
	@echo "  make cluster NW=hoodi   NODE_URL=https://holesky.node OA=0x... OIDS=1,2,3,4 ARGS='-json -debug'"
	@echo "  make operator NW=mainnet NODE_URL=https://mainnet.node OA=0x... OUT=./out ARGS=-json"
	@echo "═══════════════════════════════════════════════════════════════════════════════════════════════════════"

build:
	mkdir -p ./cmd/bin
	go build -o ./cmd/bin/$(APP) ./cmd

run: build
	./cmd/bin/$(APP)

# Usage: make nonce NW=mainnet NODE_URL=https://... OA=0x...
nonce: build
	./cmd/bin/$(APP) nonce -nw $(NW) -n $(NODE_URL) -oa $(OA) $(ARGS) $(ARG)

# Usage: make cluster NW=mainnet NODE_URL=https://... OA=0x... OIDS=1,2,3,4
cluster: build
	./cmd/bin/$(APP) cluster -nw $(NW) -n $(NODE_URL) -oa $(OA) -oids $(OIDS) $(ARGS) $(ARG)

# Usage: make operator NW=mainnet NODE_URL=https://... OA=0x... [OUT=./out]
operator: build
	./cmd/bin/$(APP) operator -nw $(NW) -n $(NODE_URL) -oa $(OA) $(if $(OUT),-o $(OUT),) $(ARGS) $(ARG)

lint:
	@echo "Running go vet..."
	go vet ./...
	@# Run golangci-lint if installed; don't mask its exit status or misreport availability
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed; skipping"; \
	fi
