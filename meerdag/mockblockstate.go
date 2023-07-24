package meerdag

import (
	"bytes"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
)

type mockBlockState struct {
	id     uint64
	order  uint64
	weight uint64
	status model.BlockStatus
}

func (b *mockBlockState) GetID() uint64 {
	return b.id
}

func (b *mockBlockState) SetOrder(o uint64) {
	b.order = o
}

func (b *mockBlockState) GetOrder() uint64 {
	return b.order
}

func (b *mockBlockState) IsOrdered() bool {
	return b.GetOrder() != uint64(MaxBlockOrder)
}

func (b *mockBlockState) SetWeight(weight uint64) {
	b.weight = weight
}

func (b *mockBlockState) GetWeight() uint64 {
	return b.weight
}

func (b *mockBlockState) SetStatusFlags(flags model.BlockStatus) {
	b.status |= flags
}

func (b *mockBlockState) UnsetStatusFlags(flags model.BlockStatus) {
	b.status &^= flags
}

func (b *mockBlockState) Valid() {
	b.UnsetStatusFlags(model.StatusInvalid)
}

func (b *mockBlockState) Invalid() {
	b.SetStatusFlags(model.StatusInvalid)
}

func (b *mockBlockState) GetStatus() model.BlockStatus {
	return b.status
}

func (b *mockBlockState) EncodeRLP(_w io.Writer) error {
	w := rlp.NewEncoderBuffer(_w)
	_tmp0 := w.List()
	w.WriteUint64(b.id)
	w.WriteUint64(b.order)
	w.WriteUint64(b.weight)
	w.WriteUint64(uint64(b.status))
	w.ListEnd(_tmp0)
	return w.Flush()
}

func (b *mockBlockState) DecodeRLP(dec *rlp.Stream) error {
	var _tmp0 mockBlockState
	{
		if _, err := dec.List(); err != nil {
			return err
		}
		// Id:
		_tmp1, err := dec.Uint64()
		if err != nil {
			return err
		}
		_tmp0.id = _tmp1
		// Order:
		_tmp2, err := dec.Uint64()
		if err != nil {
			return err
		}
		_tmp0.order = _tmp2
		// Weight:
		_tmp3, err := dec.Uint64()
		if err != nil {
			return err
		}
		_tmp0.weight = _tmp3
		// Status:
		_tmp4, err := dec.Uint8()
		if err != nil {
			return err
		}
		_tmp0.status = model.BlockStatus(_tmp4)
		if err := dec.ListEnd(); err != nil {
			return err
		}
	}
	*b = _tmp0
	return nil
}

func (b *mockBlockState) Bytes() ([]byte, error) {
	return rlp.EncodeToBytes(b)
}

func (b *mockBlockState) Root() *hash.Hash {
	return nil
}

func (b *mockBlockState) GetEVMRoot() common.Hash {
	return common.Hash{}
}
func (b *mockBlockState) GetEVMHash() common.Hash {
	return common.Hash{}
}
func (b *mockBlockState) GetEVMNumber() uint64 {
	return 0
}
func (b *mockBlockState) SetEVM(header *etypes.Header) {

}
func (b *mockBlockState) GetDuplicateTxs() []int {
	return nil
}

func (b *mockBlockState) Update(block *types.SerializedBlock, prev model.BlockState, header *etypes.Header) {

}

func CreateMockBlockState(id uint64) model.BlockState {
	return &mockBlockState{id: id, status: model.StatusNone, order: uint64(MaxBlockOrder)}
}

func CreateMockBlockStateFromBytes(data []byte) (model.BlockState, error) {
	var bs mockBlockState
	err := rlp.Decode(bytes.NewReader(data), &bs)
	if err != nil {
		return nil, err
	}
	return &bs, nil
}
