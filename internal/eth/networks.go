package eth

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// NetworkSettings holds per-network contract information
type NetworkSettings struct {
	Name            string
	ContractAddress string
	GenesisBlock    int64
	ABI             abi.ABI
}

const (
	// Mainnet
	DEFAULT_SSV_ADDRESS_MAINNET = "0xDD9BC35aE942eF0cFa76930954a156B3fF30a4E1"
	DEFAULT_SSV_GENESIS_MAINNET = 17507487

	// Hoodi
	DEFAULT_SSV_ADDRESS_HOODI = "0x58410Bef803ECd7E63B23664C586A6DB72DAf59c"
	DEFAULT_SSV_GENESIS_HOODI = 1065

	// Hoodi Stage
	DEFAULT_SSV_ADDRESS_HOODI_STAGE = "0x0aaace4e8affc47c6834171c88d342a4abd8f105"
	DEFAULT_SSV_GENESIS_HOODI_STAGE = 101653

	// Local Testnet
	DEFAULT_SSV_ADDRESS_LOCAL_TESTNET = "0xBFfF570853d97636b78ebf262af953308924D3D8"
	DEFAULT_SSV_GENESIS_LOCAL_TESTNET = 0
)

//go:embed abi/ssv_events.abi.json
var embeddedEventsABI []byte

// GetContractSettings returns the contract settings for a supported network.
// Supports: mainnet, hoodi, hoodi_stage, local_testnet.
func GetContractSettings(network string) (*NetworkSettings, error) {
	var addr string
	var genesis int64
	switch strings.ToLower(network) {
	case "mainnet":
		addr = getenvDefault("SSV_ADDRESS_MAINNET", DEFAULT_SSV_ADDRESS_MAINNET)
		genesis = getenvInt64Default("SSV_GENESIS_MAINNET", DEFAULT_SSV_GENESIS_MAINNET)
	case "hoodi":
		addr = getenvDefault("SSV_ADDRESS_HOODI", DEFAULT_SSV_ADDRESS_HOODI)
		genesis = getenvInt64Default("SSV_GENESIS_HOODI", DEFAULT_SSV_GENESIS_HOODI)
	case "hoodi_stage":
		addr = getenvDefault("SSV_ADDRESS_HOODI_STAGE", DEFAULT_SSV_ADDRESS_HOODI_STAGE)
		genesis = getenvInt64Default("SSV_GENESIS_HOODI_STAGE", DEFAULT_SSV_GENESIS_HOODI_STAGE)
	case "local_testnet":
		addr = getenvDefault("SSV_ADDRESS_LOCAL_TESTNET", DEFAULT_SSV_ADDRESS_LOCAL_TESTNET)
		genesis = getenvInt64Default("SSV_GENESIS_LOCAL_TESTNET", DEFAULT_SSV_GENESIS_LOCAL_TESTNET)
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	a, err := abi.JSON(strings.NewReader(string(embeddedEventsABI)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded ABI: %w", err)
	}
	return &NetworkSettings{Name: network, ContractAddress: addr, GenesisBlock: genesis, ABI: a}, nil
}
