package eth

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abi/ssv_views.abi.json
var embeddedViewsABI []byte

var (
	viewsABIOnce sync.Once
	viewsABI     abi.ABI
	viewsABIErr  error
)

func GetViewsABI() (*abi.ABI, error) {
	viewsABIOnce.Do(func() {
		viewsABI, viewsABIErr = abi.JSON(strings.NewReader(string(embeddedViewsABI)))
	})
	if viewsABIErr != nil {
		return nil, viewsABIErr
	}
	return &viewsABI, nil
}
