package state

import "github.com/Qitmeer/qng/consensus/model"

func CreateBlockState(id uint64) model.BlockState {
	return NewBlockState(id)
}

func CreateBlockStateFromBytes(data []byte) (model.BlockState, error) {
	return NewBlockStateFromBytes(data)
}
