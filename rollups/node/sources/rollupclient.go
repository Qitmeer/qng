package sources

import (
	"context"

	"github.com/Qitmeer/qit/common/hexutil"

	"github.com/Qitmeer/qng/rollups/node/client"
	"github.com/Qitmeer/qng/rollups/node/eth"
	"github.com/Qitmeer/qng/rollups/node/rollup"
)

type RollupClient struct {
	rpc client.RPC
}

func NewRollupClient(rpc client.RPC) *RollupClient {
	return &RollupClient{rpc}
}

func (r *RollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	var output *eth.OutputResponse
	err := r.rpc.CallContext(ctx, &output, "optimism_outputAtBlock", hexutil.Uint64(blockNum))
	return output, err
}

func (r *RollupClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	var output *eth.SyncStatus
	err := r.rpc.CallContext(ctx, &output, "optimism_syncStatus")
	return output, err
}

func (r *RollupClient) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	var output *rollup.Config
	err := r.rpc.CallContext(ctx, &output, "optimism_rollupConfig")
	return output, err
}

func (r *RollupClient) Version(ctx context.Context) (string, error) {
	var output string
	err := r.rpc.CallContext(ctx, &output, "optimism_version")
	return output, err
}
