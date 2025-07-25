package model

import (
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/consensus/model/meer"
	"github.com/Qitmeer/qng/v2/node/service"
	"github.com/Qitmeer/qng/v2/rpc/api"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

type MeerChain interface {
	service.IService
	RegisterAPIs(apis []api.API)
	GetBlockIDByTxHash(txhash *hash.Hash) uint64
	SyncMode() downloader.SyncMode
	Downloader() *downloader.Downloader
	Ether() *eth.Ethereum
	Node() *node.Node
	Server() *p2p.Server
	CurrentHeader() *types.Header
	Synced() bool
	SetSynced()
	HasState(root common.Hash) bool
	CheckState(blockNrOrHash *api.HashOrNumber) bool
	GetPivot() uint64
	RewindTo(state BlockState) error
	PrepareEnvironment(state BlockState) (*types.Header, error)
	Genesis() *hash.Hash
	GetBalance(addre string) (int64, error)
	CheckConnectBlock(block *meer.Block) error
	ConnectBlock(block *meer.Block) (uint64, error)
	VerifyTx(tx *meer.VMTx, view interface{}) (int64, error)
	CheckSanity(vt *meer.VMTx) error
	TxPool() meer.TxPool
	SetMainTxPool(tp TxPool)
	SetP2P(ser P2PService)
	GetMinerAccount() (common.Address, accounts.Wallet)
}
