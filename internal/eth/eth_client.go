package eth

import "github.com/ethereum/go-ethereum/ethclient"

// ConnectHTTP dials an Ethereum JSON-RPC endpoint (HTTP/HTTPS or WS).
func ConnectHTTP(url string) (*ethclient.Client, error) {
	return ethclient.Dial(url)
}
