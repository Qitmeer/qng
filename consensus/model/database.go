package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
)

type DataBase interface {
	Name() string
	Init() error
	Close()
	Rebuild(mgr IndexManager) error
	GetInfo() (*common.DatabaseInfo, error)
	PutInfo(di *common.DatabaseInfo) error
	GetSpendJournal(bh *hash.Hash) ([]byte, error)
	PutSpendJournal(bh *hash.Hash, data []byte) error
	DeleteSpendJournal(bh *hash.Hash) error
	GetUtxo(key []byte) ([]byte, error)
	PutUtxo(key []byte, data []byte) error
	DeleteUtxo(key []byte) error
	ForeachUtxo(fn func(key []byte, data []byte) error) error
	GetTokenState(blockID uint) ([]byte, error)
	PutTokenState(blockID uint, data []byte) error
	DeleteTokenState(blockID uint) error
	GetBestChainState() ([]byte, error)
	PutBestChainState(data []byte) error
	GetBlock(hash *hash.Hash) (*types.SerializedBlock, error)
	GetBlockBytes(hash *hash.Hash) ([]byte, error)
	GetHeader(hash *hash.Hash) (*types.BlockHeader, error)
	PutBlock(block *types.SerializedBlock) error
	HasBlock(hash *hash.Hash) bool
	GetDagInfo() ([]byte, error)
	PutDagInfo(data []byte) error
	GetDAGBlock(blockID uint) ([]byte, error)
	PutDAGBlock(blockID uint, data []byte) error
	DeleteDAGBlock(blockID uint) error
	GetDAGBlockIdByHash(bh *hash.Hash) (uint, error)
	PutDAGBlockIdByHash(bh *hash.Hash, id uint) error
	DeleteDAGBlockIdByHash(bh *hash.Hash) error
	PutMainChainBlock(blockID uint) error
	HasMainChainBlock(blockID uint) bool
	DeleteMainChainBlock(blockID uint) error
	PutBlockIdByOrder(order uint, id uint) error
	GetBlockIdByOrder(order uint) (uint, error)
	PutDAGTip(id uint, isMain bool) error
	GetDAGTips() ([]uint, error)
	DeleteDAGTip(id uint) error
	PutDiffAnticone(id uint) error
	GetDiffAnticones() ([]uint, error)
	DeleteDiffAnticone(id uint) error
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	PutTxIdxEntrys(sblock *types.SerializedBlock, block Block) error
	GetTxIdxEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error)
	DeleteTxIdxEntrys(block *types.SerializedBlock) error
	PutTxHashs(block *types.SerializedBlock) error
	GetTxIdByHash(fullHash *hash.Hash) (*hash.Hash, error)
	DeleteTxHashs(block *types.SerializedBlock) error
	IsInvalidTxEmpty() bool
	GetInvalidTxTip() (uint64, *hash.Hash, error)
	PutInvalidTxTip(order uint64, bh *hash.Hash) error
	PutInvalidTxs(sblock *types.SerializedBlock, block Block) error
	DeleteInvalidTxs(sblock *types.SerializedBlock, block Block) error
	GetInvalidTx(id *hash.Hash) (*types.Transaction, error)
	GetInvalidTxIdByHash(fullHash *hash.Hash) (*hash.Hash, error)
	CleanInvalidTxs() error
	GetAddrIdxTip() (*hash.Hash, uint, error)
	PutAddrIdxTip(bh *hash.Hash, order uint) error
	PutAddrIdx(sblock *types.SerializedBlock, block Block, stxos [][]byte) error
	GetTxForAddress(addr types.Address, numToSkip, numRequested uint32, reverse bool) ([]*common.RetrievedTx, uint32, error)
	DeleteAddrIdx(sblock *types.SerializedBlock, stxos [][]byte) error
	CleanAddrIdx(finish bool) error
	IsLegacy() bool
	TryUpgrade(di *common.DatabaseInfo, interrupt <-chan struct{}) error
	GetEstimateFee() ([]byte, error)
	PutEstimateFee(data []byte) error
	DeleteEstimateFee() error
}
