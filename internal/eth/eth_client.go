package eth

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
)

// ConnectHTTP dials an Ethereum JSON-RPC endpoint (HTTP/HTTPS or WS).
func ConnectHTTP(ctx context.Context, url string) (*ethclient.Client, error) {
	return ethclient.DialContext(ctx, url)
}
