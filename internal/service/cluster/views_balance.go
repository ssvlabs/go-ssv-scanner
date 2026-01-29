package cluster

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ssvlabs/go-ssv-scanner/internal/eth"
)

type viewsCluster struct {
	ValidatorCount  uint32   `abi:"validatorCount"`
	NetworkFeeIndex uint64   `abi:"networkFeeIndex"`
	Index           uint64   `abi:"index"`
	Active          bool     `abi:"active"`
	Balance         *big.Int `abi:"balance"`
}

func getClusterBalance(ctx context.Context, client *ethclient.Client, network eth.NetworkSettings, owner common.Address, operatorIDs []uint64, cluster ClusterSnapshot) (*big.Int, error) {
	viewsABI, err := eth.GetViewsABI()
	if err != nil {
		return nil, err
	}

	viewsAddress := common.HexToAddress(network.ViewsAddress)
	if (viewsAddress == common.Address{}) {
		viewsAddress = common.HexToAddress(network.ContractAddress)
	}

	nfi, err := bigIntToUint64(cluster.NetworkFeeIndex)
	if err != nil {
		return nil, fmt.Errorf("networkFeeIndex: %w", err)
	}
	idx, err := bigIntToUint64(cluster.Index)
	if err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}

	inputCluster := viewsCluster{
		ValidatorCount:  cluster.ValidatorCount,
		NetworkFeeIndex: nfi,
		Index:           idx,
		Active:          cluster.Active,
		Balance:         cluster.Balance,
	}

	data, err := viewsABI.Pack("getBalance", owner, operatorIDs, inputCluster)
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	res, err := client.CallContract(callCtx, ethereum.CallMsg{To: &viewsAddress, Data: data}, nil)
	if err != nil {
		return nil, err
	}

	out, err := viewsABI.Unpack("getBalance", res)
	if err != nil {
		return nil, err
	}
	if len(out) != 1 {
		return nil, fmt.Errorf("unexpected getBalance return length: %d", len(out))
	}

	balance, ok := out[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected getBalance return type: %T", out[0])
	}

	return balance, nil
}

func bigIntToUint64(v *big.Int) (uint64, error) {
	if v == nil {
		return 0, nil
	}
	if v.Sign() < 0 {
		return 0, fmt.Errorf("negative")
	}
	if v.BitLen() > 64 {
		return 0, fmt.Errorf("overflows uint64")
	}
	return v.Uint64(), nil
}
