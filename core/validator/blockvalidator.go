package validator

import "github.com/Qitmeer/qng/consensus/model"

type BlockValidator struct {
	bc model.BlockChain
}

func NewBlockValidator(bc model.BlockChain) *BlockValidator {
	return &BlockValidator{
		bc:bc,
	}
}