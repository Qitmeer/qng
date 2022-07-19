package blockchain

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"sync"
)

func OutpointKey(outpoint types.TxOutPoint) *[]byte {
	// A VLQ employs an MSB encoding, so they are useful not only to reduce
	// the amount of storage space, but also so iteration of utxos when
	// doing byte-wise comparisons will produce them in order.
	key := outpointKeyPool.Get().(*[]byte)
	idx := uint64(outpoint.OutIndex)
	*key = (*key)[:hash.HashSize+serialization.SerializeSizeVLQ(idx)]
	copy(*key, outpoint.Hash[:])
	serialization.PutVLQ((*key)[hash.HashSize:], idx)
	return key
}

var outpointKeyPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, hash.HashSize+serialization.MaxUint32VLQSerializeSize)
		return &b // Pointer to slice to avoid boxing alloc.
	},
}

func RecycleOutpointKey(key *[]byte) {
	outpointKeyPool.Put(key)
}
