package coinbase

import (
	"encoding/binary"
	"errors"
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/engine/txscript"
)

func ExtractCoinbaseHeight(coinbaseTx *types.Transaction) (uint64, error) {
	sigScript := coinbaseTx.TxIn[0].SignScript
	if len(sigScript) < 1 {
		str := "It has not the coinbase signature script for blocks"
		return 0, errors.New(str)
	}

	// Detect the case when the block height is a small integer encoded with
	// as single byte.
	opcode := int(sigScript[0])
	if opcode == txscript.OP_0 {
		return 0, nil
	}
	if opcode >= txscript.OP_1 && opcode <= txscript.OP_16 {
		return uint64(opcode - (txscript.OP_1 - 1)), nil
	}

	// Otherwise, the opcode is the length of the following bytes which
	// encode in the block height.
	serializedLen := int(sigScript[0])
	if len(sigScript[1:]) < serializedLen {
		str := "It has not the coinbase signature script for blocks"
		return 0, errors.New(str)
	}

	serializedHeightBytes := make([]byte, 8)
	copy(serializedHeightBytes, sigScript[1:serializedLen+1])
	serializedHeight := binary.LittleEndian.Uint64(serializedHeightBytes)

	return serializedHeight, nil
}
