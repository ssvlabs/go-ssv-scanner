package service

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// block window sizes aligned to upstream (approx):
const (
	DayBlocks   = int64(5400)
	WeekBlocks  = DayBlocks * 7
	MonthBlocks = DayBlocks * 30
)

func FilterLogsTimeout(ctx context.Context, client *ethclient.Client, q ethereum.FilterQuery) ([]types.Log, error) {
	lctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return client.FilterLogs(lctx, q)
}
