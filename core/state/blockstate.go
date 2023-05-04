package state

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/merkle"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/common"
)

const MaxBlockOrder = uint64(^uint32(0))

type BlockState struct {
	id           uint64
	order        uint64
	weight       uint64
	status       model.BlockStatus
	duplicateTxs []int
	evmRoot      common.Hash
	root         hash.Hash
}

func (b *BlockState) SetWeight(weight uint64) {
	b.weight = weight
}

func (b *BlockState) GetWeight() uint64 {
	return b.weight
}

func (b *BlockState) SetStatusFlags(flags model.BlockStatus) {
	b.status |= flags
}

func (b *BlockState) UnsetStatusFlags(flags model.BlockStatus) {
	b.status &^= flags
}

func (b *BlockState) Valid() {
	b.UnsetStatusFlags(model.StatusInvalid)
}

func (b *BlockState) Invalid() {
	b.SetStatusFlags(model.StatusInvalid)
}

func (b *BlockState) SetOrder(o uint64) {
	b.order = o

	if !b.IsOrdered() {
		b.Reset()
	}
}

func (b *BlockState) GetOrder() uint64 {
	return b.order
}

func (b *BlockState) IsOrdered() bool {
	return b.GetOrder() != MaxBlockOrder
}

func (b *BlockState) Root() *hash.Hash {
	return &b.root
}

func (b *BlockState) Reset() {
	b.root = hash.ZeroHash
}

func (b *BlockState) Update(block *types.SerializedBlock, evmRoot common.Hash) {
	defer func() {
		log.Trace("Update block state", "id", b.id, "order", b.order, "root", b.root.String())
	}()
	b.evmRoot = evmRoot
	b.root = hash.ZeroHash
	if b.status.KnownInvalid() ||
		!b.IsOrdered() {
		return
	}
	b.duplicateTxs = []int{}
	txs := []*types.Tx{}
	txRoot := block.Block().Header.TxRoot
	for _, tx := range block.Transactions() {
		if tx.IsDuplicate {
			b.duplicateTxs = append(b.duplicateTxs, tx.Index())
		} else {
			txs = append(txs, tx)
		}
	}
	if len(b.duplicateTxs) > 0 {
		merkles := merkle.BuildMerkleTreeStore(txs, false)
		txRoot = *merkles[len(merkles)-1]
	}
	//
	data := serialization.SerializeUint64(b.order)
	data = append(data, serialization.SerializeUint64(b.weight)...)
	data = append(data, byte(b.status))
	data = append(data, txRoot.Bytes()...)
	data = append(data, b.evmRoot.Bytes()...)
	b.root = hash.DoubleHashH(data)
}

func NewBlockState(id uint64) *BlockState {
	return &BlockState{id: id, status: model.StatusNone, root: hash.ZeroHash, order: MaxBlockOrder}
}
