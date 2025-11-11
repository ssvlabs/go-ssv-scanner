package nonce

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
    "github.com/ssvlabs/go-ssv-scanner/internal/eth"
    "github.com/ssvlabs/go-ssv-scanner/internal/service"
)

// ScanNonce counts ValidatorAdded events for an owner from genesis to latest.
// Degrades window size on provider errors (month -> week -> day), similar to upstream.
func ScanNonce(ctx context.Context, client *ethclient.Client, contractAddr common.Address, owner common.Address, genesis int64, validatorAddedID common.Hash, prog *eth.ScanProgress) (int, error) {
	latest, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	// sanity: ensure contract is reachable by attempting a light query (filter in tiny range)
	// not strictly required; provider errors will surface below.

	// Build initial month-sized ranges
	step := service.MonthBlocks
	ranges := make([][2]*big.Int, 0)
	for f := big.NewInt(genesis); f.Int64() <= int64(latest); {
		t := new(big.Int).Add(f, big.NewInt(step-1))
		if t.Int64() > int64(latest) {
			t = new(big.Int).SetUint64(latest)
		}
		ranges = append(ranges, [2]*big.Int{new(big.Int).Set(f), t})
		f = new(big.Int).Add(t, big.NewInt(1))
	}

	workers := runtime.NumCPU()
	if workers > 6 {
		workers = 6
	}
	ch := make(chan [2]*big.Int, len(ranges))
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	if prog != nil {
		prog.SetName("nonce")
		prog.SetBase(genesis)
		prog.SetTotal(int64(latest) - genesis + 1)
	}

	// recursive count with subdivision on errors
	var countRange func(from, to *big.Int, subStep int64)
	countRange = func(from, to *big.Int, subStep int64) {
		if prog != nil {
			prog.Update(from.Int64(), to.Int64(), int64(latest), subStep)
		}
		// Topic for indexed address must be left-padded to 32 bytes
		ownerTopic := common.BytesToHash(common.LeftPadBytes(owner.Bytes(), 32))
		q := ethereum.FilterQuery{
			FromBlock: from, ToBlock: to, Addresses: []common.Address{contractAddr},
			Topics: [][]common.Hash{{validatorAddedID}, {ownerTopic}},
		}
		logs, err := filterLogsWithRetry(ctx, client, q)
		if err == nil {
			mu.Lock()
			total += len(logs)
			mu.Unlock()
			if prog != nil {
				size := to.Int64() - from.Int64() + 1
				if size > 0 {
					prog.AddDone(size)
				}
			}
			return
		}
		// subdivide if possible
		var next int64
		switch subStep {
		case service.MonthBlocks:
			next = service.WeekBlocks
		case service.WeekBlocks:
			next = service.DayBlocks
		default:
			return
		}
		for s := new(big.Int).Set(from); s.Cmp(to) <= 0; {
			e := new(big.Int).Add(s, big.NewInt(next-1))
			if e.Cmp(to) > 0 {
				e = new(big.Int).Set(to)
			}
			countRange(s, e, next)
			s = new(big.Int).Add(e, big.NewInt(1))
		}
	}

	worker := func() {
		defer wg.Done()
		for r := range ch {
			countRange(r[0], r[1], step)
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}
	for _, r := range ranges {
		ch <- r
	}
	close(ch)
	wg.Wait()
	return total, nil
}
