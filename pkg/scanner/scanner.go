package scanner

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ssvlabs/go-ssv-scanner/internal/eth"
	"github.com/ssvlabs/go-ssv-scanner/internal/service/cluster"
	"github.com/ssvlabs/go-ssv-scanner/internal/service/nonce"
	"github.com/ssvlabs/go-ssv-scanner/internal/service/operator"
)

// NewScanner initializes a Scanner using the provided config.
func NewScanner(ctx context.Context, cfg Config) (*Scanner, error) {
	if cfg.Network == "" {
		return nil, fmt.Errorf("network is required")
	}
	if cfg.NodeURL == "" {
		return nil, fmt.Errorf("node url is required")
	}

	netCfg, err := eth.GetContractSettings(cfg.Network)
	if err != nil {
		return nil, err
	}
	client, err := eth.ConnectHTTP(ctx, cfg.NodeURL)
	if err != nil {
		return nil, fmt.Errorf("connect node: %w", err)
	}
	return &Scanner{cfg: cfg, client: client, network: netCfg, logger: cfg.Logger}, nil
}

// Close closes underlying resources (currently a no-op for ethclient, but kept for API symmetry).
func (s *Scanner) Close() error { return nil }

// Scan provides a unified entrypoint using a Request. It returns a typed result based on Kind:
//
//	nonce    -> NonceResult
//	cluster  -> cluster.ClusterResult
//	operator -> OperatorResult
func (s *Scanner) Scan(ctx context.Context, req Request) (any, error) {
	switch req.Kind {
	case "nonce":
		if req.Owner == "" {
			return nil, fmt.Errorf("owner is required")
		}
		n, err := s.Nonce(ctx, req.Owner)
		if err != nil {
			return nil, err
		}
		return NonceResult{Owner: req.Owner, Nonce: n}, nil
	case "cluster":
		if req.Owner == "" {
			return nil, fmt.Errorf("owner is required")
		}
		if len(req.OperatorIDs) == 0 {
			return nil, fmt.Errorf("operator ids are required")
		}
		return s.Cluster(ctx, req.Owner, req.OperatorIDs)
	case "operator":
		if req.Owner == "" {
			return nil, fmt.Errorf("owner is required")
		}
		path, count, err := s.OperatorPubkeys(ctx, req.OutputPath)
		if err != nil {
			return nil, err
		}
		return OperatorResult{File: path, Count: count}, nil
	default:
		return nil, fmt.Errorf("unsupported kind: %s", req.Kind)
	}
}

// Nonce scans all ValidatorAdded events for the owner and returns the total count.
func (s *Scanner) Nonce(ctx context.Context, ownerHex string) (int, error) {
	owner := common.HexToAddress(ownerHex)
	evID := s.network.ABI.Events["ValidatorAdded"].ID
	prog, stop := s.maybeProgress(ctx, s.logger, "nonce")
	total, err := nonce.ScanNonce(ctx, s.client, common.HexToAddress(s.network.ContractAddress), owner, s.network.GenesisBlock, evID, prog)
	if stop != nil {
		stop()
	}
	if err != nil {
		return 0, err
	}
	return total, nil
}

// Cluster returns the latest cluster-affecting snapshot for the owner and operator IDs.
func (s *Scanner) Cluster(ctx context.Context, ownerHex string, operatorIDs []uint64) (*cluster.ClusterResult, error) {
	owner := common.HexToAddress(ownerHex)
	prog, stop := s.maybeProgress(ctx, s.logger, "cluster")
	res, err := cluster.GetLatestClusterSnapshot(ctx, s.client, *s.network, owner, operatorIDs, prog, s.logger)
	if stop != nil {
		stop()
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

// OperatorPubkeys exports all operator pubkeys and returns the file path and count.
func (s *Scanner) OperatorPubkeys(ctx context.Context, outputDir string) (string, int, error) {
	prog, stop := s.maybeProgress(ctx, s.logger, "operator")
	path, count, err := operator.ExportOperatorPubkeys(ctx, s.client, *s.network, outputDir, prog)
	if stop != nil {
		stop()
	}
	return path, count, err
}
