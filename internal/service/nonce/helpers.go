package nonce

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func filterLogsWithRetry(ctx context.Context, client *ethclient.Client, q ethereum.FilterQuery) ([]types.Log, error) {
	// Add basic timeout per call
	lctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return client.FilterLogs(lctx, q)
}
