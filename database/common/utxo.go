package common

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
)

type UtxoOpt struct {
	Key  []byte
	Data []byte
	Add  bool
}

func ParseOutpoint(opk []byte) (*types.TxOutPoint, error) {
	txhash, err := hash.NewHash(opk[:hash.HashSize])
	if err != nil {
		return nil, err
	}
	txOutIdex, size := serialization.DeserializeVLQ(opk[hash.HashSize:])
	if size <= 0 {
		err := fmt.Errorf("DeserializeVLQ:%s %v", txhash.String(), opk[hash.HashSize:])
		return nil, err
	}
	return types.NewOutPoint(txhash, uint32(txOutIdex)), nil
}
