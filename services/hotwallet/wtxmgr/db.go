package wtxmgr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
	"time"
)

// Big endian is the preferred byte order, due to cursor scans over integer
// keys iterating in order.
var byteOrder = binary.BigEndian

// This package makes assumptions that the width of a hash.Hash is always
// 32 bytes.  If this is ever changed (unlikely for bitcoin, possible for alts),
// offsets have to be rewritten.  Use a compile-time assertion that this
// assumption holds true.
var _ [32]byte = hash.Hash{}

// Bucket names
var (
	bucketBlocks         = []byte("b")
	bucketTxRecords      = []byte("t")
	bucketCredits        = []byte("c")
	bucketUnspent        = []byte("u")
	bucketDebits         = []byte("d")
	bucketUnMined        = []byte("m")
	bucketUnMinedCredits = []byte("mc")
	bucketUnMinedInputs  = []byte("mi")
	BucketUnConfirmed    = []byte("uc")
	BucketAddrtxin       = []byte("in")
	BucketAddrtxout      = []byte("out")
	BucketTxJson         = []byte("txjson")
	BucketSync           = []byte("sync")
	BucketHeight         = []byte("h")
)

// Root (namespace) bucket keys
var (
	rootCreateDate   = []byte("date")
	rootVersion      = []byte("vers")
	rootMinedBalance = []byte("bal")
)

func rootMinedBalanceKey(coinId types.CoinID) []byte {
	balSize := binary.Size(rootMinedBalance)
	idSize := binary.Size(coinId)
	n := make([]byte, balSize+idSize)
	copy(n, rootMinedBalance)
	byteOrder.PutUint16(n[balSize:balSize+idSize], uint16(coinId))
	return n
}

// The root bucket's mined balance k/v pair records the total balance for all
// unspent credits from mined transactions.  This includes immature outputs, and
// outputs spent by mempool transactions, which must be considered when
// returning the actual balance for a given number of block confirmations.  The
// value is the amount serialized as a uint64.
func fetchMinedBalance(ns walletdb.ReadBucket, coinId types.CoinID) (types.Amount, error) {
	v := ns.Get(rootMinedBalanceKey(coinId))
	if len(v) != 8 {
		str := fmt.Sprintf("balance: short read (expected 8 bytes, "+
			"read %v)", len(v))
		return types.Amount{}, storeError(ErrData, str, nil)
	}
	a := types.Amount{Value: int64(byteOrder.Uint64(v)), Id: coinId}
	return a, nil
}

func putMinedBalance(ns walletdb.ReadWriteBucket, amt types.Amount) error {
	v := make([]byte, 8)
	byteOrder.PutUint64(v, uint64(amt.Value))
	err := ns.Put(rootMinedBalanceKey(amt.Id), v)
	if err != nil {
		str := "failed to put balance"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

// Several data structures are given canonical serialization formats as either
// keys or values.  These common formats allow keys and values to be reused
// across different buckets.
//
// The canonical outpoint serialization format is:
//
//   [0:32]  Trasaction hash (32 bytes)
//   [32:36] Output index (4 bytes)
//
// The canonical transaction hash serialization is simply the hash.

func canonicalOutPoint(txHash *hash.Hash, index uint32) []byte {
	k := make([]byte, 36)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], index)
	return k
}

func readCanonicalOutPoint(k []byte, op *types.TxOutPoint) error {
	if len(k) < 36 {
		str := "short canonical outpoint"
		return storeError(ErrData, str, nil)
	}
	copy(op.Hash[:], k)
	op.OutIndex = byteOrder.Uint32(k[32:36])
	return nil
}

// Details regarding blocks are saved as k/v pairs in the blocks bucket.
// blockRecords are keyed by their height.  The value is serialized as such:
//
//   [0:32]  Hash (32 bytes)
//   [32:40] Unix time (8 bytes)
//   [40:44] Number of transaction hashes (4 bytes)
//   [44:]   For each transaction hash:
//             Hash (32 bytes)

func keyBlockRecord(order uint32) []byte {
	k := make([]byte, 4)
	byteOrder.PutUint32(k, order)
	return k
}

func valueBlockRecord(block *BlockMeta, txHash *hash.Hash) []byte {
	v := make([]byte, 76)
	copy(v, block.Hash[:])
	byteOrder.PutUint64(v[32:40], uint64(block.Time.Unix()))
	byteOrder.PutUint32(v[40:44], 1)
	copy(v[44:76], txHash[:])
	return v
}

/*
func ValueAddrTxOutput(txout *AddrTxOutput) []byte {
	var v []byte
	if txout.SpendTo == nil {
		v = make([]byte, 133)
	} else {
		v = make([]byte, 169)
	}
	copy(v, txout.TxId[:])
	byteOrder.PutUint32(v[32:36], txout.Index)
	byteOrder.PutUint64(v[36:44], uint64(txout.Amount.Value))
	byteOrder.PutUint16(v[44:46], uint16(txout.Amount.Id))
	copy(v[46:78], txout.Block.Hash[:])
	byteOrder.PutUint32(v[90:94], uint32(txout.Block.Order))
	byteOrder.PutUint32(v[94:98], uint32(txout.Spend))
	byteOrder.PutUint16(v[98:100], uint16(txout.Status))
	if txout.IsBlue {
		copy(v[100:101], []byte{1})
	} else {
		copy(v[100:101], []byte{0})
	}
	copy(v[101:133], txout.SpendTo.TxId[:])
	if len(v) == 169 {
		byteOrder.PutUint32(v[133:137], txout.SpendTo.Index)
		copy(v[137:169], txout.SpendTo.TxId[:])
	}
	return v
}

func ReadAddrTxOutput(addr string, v []byte, txout *AddrTxOutput) (err error) {
	defer func() {
		if rev := recover(); rev != nil {
			errMsg := fmt.Sprintf("ReadAddrTxOutput recover: %s", rev)
			err = errors.New(errMsg)
		}
	}()
	txId := hash.Hash{}
	txout.Address = addr
	copy(txout.TxId[:], v[0:32])
	txout.Index = byteOrder.Uint32(v[32:36])
	txout.Amount.Value = int64(byteOrder.Uint64(v[36:44]))
	txout.Amount.Id = types.CoinID(byteOrder.Uint16(v[44:46]))
	copy(txout.Block.Hash[:], v[46:78])
	txout.Block.Order = int32(byteOrder.Uint32(v[90:94]))
	txout.Spend = SpendStatus(byteOrder.Uint32(v[94:98]))
	txout.Status = TxStatus(byteOrder.Uint16(v[98:100]))
	if bytes.Compare(v[100:101], []byte{1}) == 0 {
		txout.IsBlue = true
	}
	copy(txId[:], v[101:133])
	txout.SpendTo.TxId = txId

	if len(v) == 169 {
		st := SpendTo{}
		st.Index = byteOrder.Uint32(v[133:137])
		copy(st.TxId[:], v[137:169])
		txout.SpendTo = &st
	}
	return nil
}*/

// appendRawBlockRecord returns a new block record value with a transaction
// hash appended to the end and an incremented number of transactions.
func appendRawBlockRecord(v []byte, txHash *hash.Hash) ([]byte, error) {
	if len(v) < 44 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
		return nil, storeError(ErrData, str, nil)
	}
	newv := append(v[:len(v):len(v)], txHash[:]...)
	n := byteOrder.Uint32(newv[40:44])
	byteOrder.PutUint32(newv[40:44], n+1)
	return newv, nil
}

func putRawBlockRecord(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketBlocks).Put(k, v)
	if err != nil {
		str := "failed to store block"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func putBlockRecord(ns walletdb.ReadWriteBucket, block *BlockMeta, txHash *hash.Hash) error {
	k := keyBlockRecord(uint32(block.Order))
	v := valueBlockRecord(block, txHash)
	return putRawBlockRecord(ns, k, v)
}

func fetchBlockTime(ns walletdb.ReadBucket, order uint32) (time.Time, error) {
	k := keyBlockRecord(order)
	v := ns.NestedReadBucket(bucketBlocks).Get(k)
	if len(v) < 44 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
		return time.Time{}, storeError(ErrData, str, nil)
	}
	return time.Unix(int64(byteOrder.Uint64(v[32:40])), 0), nil
}

func existsBlockRecord(ns walletdb.ReadBucket, order uint32) (k, v []byte) {
	k = keyBlockRecord(order)
	v = ns.NestedReadBucket(bucketBlocks).Get(k)
	return
}

func readRawBlockRecord(k, v []byte, block *blockRecord) error {
	if len(k) < 4 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketBlocks, 4, len(k))
		return storeError(ErrData, str, nil)
	}
	if len(v) < 44 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
		return storeError(ErrData, str, nil)
	}
	numTransactions := int(byteOrder.Uint32(v[40:44]))
	expectedLen := 44 + hash.HashSize*numTransactions
	if len(v) < expectedLen {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, expectedLen, len(v))
		return storeError(ErrData, str, nil)
	}

	block.Order = int32(byteOrder.Uint32(k))
	copy(block.Hash[:], v)
	block.Time = time.Unix(int64(byteOrder.Uint64(v[32:40])), 0)
	block.transactions = make([]hash.Hash, numTransactions)
	off := 44
	for i := range block.transactions {
		copy(block.transactions[i][:], v[off:])
		off += hash.HashSize
	}

	return nil
}

type blockIterator struct {
	c    walletdb.ReadWriteCursor
	seek []byte
	ck   []byte
	cv   []byte
	elem blockRecord
	err  error
}

func makeReadBlockIterator(ns walletdb.ReadBucket, height int32) blockIterator {
	seek := make([]byte, 4)
	byteOrder.PutUint32(seek, uint32(height))
	c := ns.NestedReadBucket(bucketBlocks).ReadCursor()
	return blockIterator{c: readCursor{c}, seek: seek}
}

// Works just like makeBlockIterator but will initially position the cursor at
// the last k/v pair.  Use this with blockIterator.prev.
func makeReverseBlockIterator(ns walletdb.ReadWriteBucket) blockIterator {
	seek := make([]byte, 4)
	byteOrder.PutUint32(seek, ^uint32(0))
	c := ns.NestedReadWriteBucket(bucketBlocks).ReadWriteCursor()
	return blockIterator{c: c, seek: seek}
}

func makeReadReverseBlockIterator(ns walletdb.ReadBucket) blockIterator {
	seek := make([]byte, 4)
	byteOrder.PutUint32(seek, ^uint32(0))
	c := ns.NestedReadBucket(bucketBlocks).ReadCursor()
	return blockIterator{c: readCursor{c}, seek: seek}
}

func (it *blockIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.seek)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if it.ck == nil {
		it.c = nil
		return false
	}

	err := readRawBlockRecord(it.ck, it.cv, &it.elem)
	if err != nil {
		it.c = nil
		it.err = err
		return false
	}

	return true
}

func (it *blockIterator) prev() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.seek)
		// Seek positions the cursor at the next k/v pair if one with
		// this prefix was not found.  If this happened (the prefixes
		// won't match in this case) move the cursor backward.
		//
		// This technically does not correct for multiple keys with
		// matching prefixes by moving the cursor to the last matching
		// key, but this doesn't need to be considered when dealing with
		// block records since the key (and seek prefix) is just the
		// block height.
		if !bytes.HasPrefix(it.ck, it.seek) {
			it.ck, it.cv = it.c.Prev()
		}
	} else {
		it.ck, it.cv = it.c.Prev()
	}
	if it.ck == nil {
		it.c = nil
		return false
	}

	err := readRawBlockRecord(it.ck, it.cv, &it.elem)
	if err != nil {
		it.c = nil
		it.err = err
		return false
	}

	return true
}

func (it *blockIterator) reposition(order uint32) {
	it.c.Seek(keyBlockRecord(order))
}

func deleteBlockRecord(ns walletdb.ReadWriteBucket, order uint32) error {
	k := keyBlockRecord(order)
	return ns.NestedReadWriteBucket(bucketBlocks).Delete(k)
}

// Transaction records are keyed as such:
//
//   [0:32]  Transaction hash (32 bytes)
//   [32:36] Block height (4 bytes)
//   [36:68] Block hash (32 bytes)
//
// The leading transaction hash allows to prefix filter for all records with
// a matching hash.  The block height and hash records a particular incidence
// of the transaction in the blockchain.
//
// The record value is serialized as such:
//
//   [0:8]   Received time (8 bytes)
//   [8:]    Serialized transaction (varies)

func keyTxRecord(txHash *hash.Hash, block *Block) []byte {
	k := make([]byte, 68)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], uint32(block.Order))
	copy(k[36:68], block.Hash[:])
	return k
}

func valueTxRecord(rec *TxRecord) ([]byte, error) {
	var v []byte
	if rec.SerializedTx == nil {
		txSize := rec.MsgTx.SerializeSize()
		v = make([]byte, 8, 8+txSize)
		bu, err := rec.MsgTx.Serialize()
		if err != nil {
			str := fmt.Sprintf("unable to serialize transaction %v", rec.Hash)
			return nil, storeError(ErrInput, str, err)
		}
		copy(v[8:], bu)
		v = v[:cap(v)]
	} else {
		v = make([]byte, 8+len(rec.SerializedTx))
		copy(v[8:], rec.SerializedTx)
	}
	byteOrder.PutUint64(v, uint64(rec.Received.Unix()))
	return v, nil
}

func putTxRecord(ns walletdb.ReadWriteBucket, rec *TxRecord, block *Block) error {
	k := keyTxRecord(&rec.Hash, block)
	v, err := valueTxRecord(rec)
	if err != nil {
		return err
	}
	err = ns.NestedReadWriteBucket(bucketTxRecords).Put(k, v)
	if err != nil {
		str := fmt.Sprintf("%s: put failed for %v", bucketTxRecords, rec.Hash)
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func readRawTxRecord(txHash *hash.Hash, v []byte, rec *TxRecord) error {
	if len(v) < 8 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketTxRecords, 8, len(v))
		return storeError(ErrData, str, nil)
	}
	rec.Hash = *txHash
	rec.Received = time.Unix(int64(byteOrder.Uint64(v)), 0)
	err := rec.MsgTx.Deserialize(bytes.NewReader(v[8:]))
	if err != nil {
		str := fmt.Sprintf("%s: failed to deserialize transaction %v",
			bucketTxRecords, txHash)
		return storeError(ErrData, str, err)
	}
	return nil
}

func readRawTxRecordBlock(k []byte, block *Block) error {
	if len(k) < 68 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketTxRecords, 68, len(k))
		return storeError(ErrData, str, nil)
	}
	block.Order = int32(byteOrder.Uint32(k[32:36]))
	copy(block.Hash[:], k[36:68])
	return nil
}

func fetchTxRecord(ns walletdb.ReadBucket, txHash *hash.Hash, block *Block) (*TxRecord, error) {
	k := keyTxRecord(txHash, block)
	v := ns.NestedReadBucket(bucketTxRecords).Get(k)

	rec := new(TxRecord)
	err := readRawTxRecord(txHash, v, rec)
	return rec, err
}

// avoid the wire.MsgTx deserialization.
func fetchRawTxRecordPkScript(k, v []byte, index uint32) ([]byte, error) {
	var rec TxRecord
	copy(rec.Hash[:], k) // Silly but need an array
	err := readRawTxRecord(&rec.Hash, v, &rec)
	if err != nil {
		return nil, err
	}
	if int(index) >= len(rec.MsgTx.TxOut) {
		str := "missing transaction output for credit index"
		return nil, storeError(ErrData, str, nil)
	}
	return rec.MsgTx.TxOut[index].PkScript, nil
}

func existsTxRecord(ns walletdb.ReadBucket, txHash *hash.Hash, block *Block) (k, v []byte) {
	k = keyTxRecord(txHash, block)
	v = ns.NestedReadBucket(bucketTxRecords).Get(k)
	return
}

func existsRawTxRecord(ns walletdb.ReadBucket, k []byte) (v []byte) {
	return ns.NestedReadBucket(bucketTxRecords).Get(k)
}

func deleteTxRecord(ns walletdb.ReadWriteBucket, txHash *hash.Hash, block *Block) error {
	k := keyTxRecord(txHash, block)
	return ns.NestedReadWriteBucket(bucketTxRecords).Delete(k)
}

// latestTxRecord searches for the newest recorded mined transaction record with
// a matching hash.  In case of a hash collision, the record from the newest
// block is returned.  Returns (nil, nil) if no matching transactions are found.
func latestTxRecord(ns walletdb.ReadBucket, txHash *hash.Hash) (k, v []byte) {
	prefix := txHash[:]
	c := ns.NestedReadBucket(bucketTxRecords).ReadCursor()
	ck, cv := c.Seek(prefix)
	var lastKey, lastVal []byte
	for bytes.HasPrefix(ck, prefix) {
		lastKey, lastVal = ck, cv
		ck, cv = c.Next()
	}
	return lastKey, lastVal
}

// All transaction credits (outputs) are keyed as such:
//
//   [0:32]  Transaction hash (32 bytes)
//   [32:36] Block height (4 bytes)
//   [36:68] Block hash (32 bytes)
//   [68:72] Output index (4 bytes)
//
// The first 68 bytes match the key for the transaction record and may be used
// as a prefix filter to iterate through all credits in order.
//
// The credit value is serialized as such:
//
//   [0:8]   Amount (8 bytes)
//   [8]     Flags (1 byte)
//             0x01: Spent
//             0x02: Change
//   [9:81]  OPTIONAL Debit bucket key (72 bytes)
//             [9:41]  Spender transaction hash (32 bytes)
//             [41:45] Spender block height (4 bytes)
//             [45:77] Spender block hash (32 bytes)
//             [77:81] Spender transaction input index (4 bytes)
//
// The optional debits key is only included if the credit is spent by another
// mined debit.

func keyCredit(txHash *hash.Hash, index uint32, block *Block) []byte {
	k := make([]byte, 72)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], uint32(block.Order))
	copy(k[36:68], block.Hash[:])
	byteOrder.PutUint32(k[68:72], index)
	return k
}

// valueUnspentCredit creates a new credit value for an unspent credit.  All
// credits are created unspent, and are only marked spent later, so there is no
// value function to create either spent or unspent credits.
func valueUnspentCredit(cred *credit) []byte {
	v := make([]byte, 9+2)
	byteOrder.PutUint64(v, uint64(cred.amount.Value))
	byteOrder.PutUint16(v, uint16(cred.amount.Id))
	if cred.change {
		v[8] |= 1 << 1
	}
	return v
}

func putRawCredit(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketCredits).Put(k, v)
	if err != nil {
		str := "failed to put credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

// putUnspentCredit puts a credit record for an unspent credit.  It may only be
// used when the credit is already know to be unspent, or spent by an
// unconfirmed transaction.
func putUnspentCredit(ns walletdb.ReadWriteBucket, cred *credit) error {
	k := keyCredit(&cred.outPoint.Hash, cred.outPoint.OutIndex, &cred.block)
	v := valueUnspentCredit(cred)
	return putRawCredit(ns, k, v)
}

func extractRawCreditTxRecordKey(k []byte) []byte {
	return k[0:68]
}

func extractRawCreditIndex(k []byte) uint32 {
	return byteOrder.Uint32(k[68:72])
}

// fetchRawCreditAmount returns the amount of the credit.
func fetchRawCreditAmount(v []byte) (types.Amount, error) {
	if len(v) < 9+2 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9+2, len(v))
		return types.Amount{}, storeError(ErrData, str, nil)
	}
	a := types.Amount{Value: int64(byteOrder.Uint64(v[0:8])), Id: types.CoinID(byteOrder.Uint16(v[8:10]))}
	return a, nil
}

// fetchRawCreditAmountSpent returns the amount of the credit and whether the
// credit is spent.
func fetchRawCreditAmountSpent(v []byte) (types.Amount, bool, error) {
	if len(v) < 9+2 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9+2, len(v))
		return types.Amount{}, false, storeError(ErrData, str, nil)
	}
	a := types.Amount{Value: int64(byteOrder.Uint64(v[0:8])), Id: types.CoinID(byteOrder.Uint16(v[8:10]))}
	return a, v[8+2]&(1<<0) != 0, nil
}

// fetchRawCreditAmountChange returns the amount of the credit and whether the
// credit is marked as change.
func fetchRawCreditAmountChange(v []byte) (types.Amount, bool, error) {
	if len(v) < 9+2 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9+2, len(v))
		return types.Amount{}, false, storeError(ErrData, str, nil)
	}
	a := types.Amount{Value: int64(byteOrder.Uint64(v)), Id: types.CoinID(byteOrder.Uint16(v[8:10]))}
	return a, v[8+2]&(1<<1) != 0, nil
}

// fetchRawCreditUnspentValue returns the unspent value for a raw credit key.
// This may be used to mark a credit as unspent.
func fetchRawCreditUnspentValue(k []byte) ([]byte, error) {
	if len(k) < 72+2 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketCredits, 72+2, len(k))
		return nil, storeError(ErrData, str, nil)
	}
	return k[32+2 : 68+2], nil
}

// spendRawCredit marks the credit with a given key as mined at some particular
// block as spent by the input at some transaction incidence.  The debited
// amount is returned.
func spendCredit(ns walletdb.ReadWriteBucket, k []byte, spender *indexedIncidence) (types.Amount, error) {
	v := ns.NestedReadBucket(bucketCredits).Get(k)
	newv := make([]byte, 81+2)
	copy(newv, v)
	v = newv
	v[8+4] |= 1 << 0
	copy(v[9+2:41+2], spender.txHash[:])
	byteOrder.PutUint32(v[41+2:45+2], uint32(spender.block.Order))
	copy(v[45+2:77+2], spender.block.Hash[:])
	byteOrder.PutUint32(v[77+2:81+2], spender.index)

	a := types.Amount{Value: int64(byteOrder.Uint64(v[0:8])), Id: types.CoinID(byteOrder.Uint16(v[8:10]))}
	return a, putRawCredit(ns, k, v)
}

// unspendRawCredit rewrites the credit for the given key as unspent.  The
// output amount of the credit is returned.  It returns without error if no
// credit exists for the key.
func unspendRawCredit(ns walletdb.ReadWriteBucket, k []byte) (types.Amount, error) {
	b := ns.NestedReadWriteBucket(bucketCredits)
	v := b.Get(k)
	if v == nil {
		return types.Amount{}, nil
	}
	newv := make([]byte, 9+2)
	copy(newv, v)
	newv[8+2] &^= 1 << 0

	err := b.Put(k, newv)
	if err != nil {
		str := "failed to put credit"
		return types.Amount{}, storeError(ErrDatabase, str, err)
	}

	a := types.Amount{Value: int64(byteOrder.Uint64(v[0:8])), Id: types.CoinID(byteOrder.Uint16(v[8:10]))}
	return a, nil
}

func existsCredit(ns walletdb.ReadBucket, txHash *hash.Hash, index uint32, block *Block) (k, v []byte) {
	k = keyCredit(txHash, index, block)
	v = ns.NestedReadBucket(bucketCredits).Get(k)
	return
}

func existsRawCredit(ns walletdb.ReadBucket, k []byte) []byte {
	return ns.NestedReadBucket(bucketCredits).Get(k)
}

func deleteRawCredit(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketCredits).Delete(k)
	if err != nil {
		str := "failed to delete credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

type creditIterator struct {
	c      walletdb.ReadWriteCursor // Set to nil after final iteration
	prefix []byte
	ck     []byte
	cv     []byte
	elem   CreditRecord
	err    error
}

func makeReadCreditIterator(ns walletdb.ReadBucket, prefix []byte) creditIterator {
	c := ns.NestedReadBucket(bucketCredits).ReadCursor()
	return creditIterator{c: readCursor{c}, prefix: prefix}
}

func (it *creditIterator) readElem() error {
	if len(it.ck) < 72+2 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketCredits, 72+2, len(it.ck))
		return storeError(ErrData, str, nil)
	}
	if len(it.cv) < 9+2 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9+2, len(it.cv))
		return storeError(ErrData, str, nil)
	}
	it.elem.Index = byteOrder.Uint32(it.ck[68+2 : 72+2])
	it.elem.Amount = types.Amount{Value: int64(byteOrder.Uint64(it.cv[0:8])), Id: types.CoinID(byteOrder.Uint16(it.cv[8:10]))}
	it.elem.Spent = it.cv[8+2]&(1<<0) != 0
	it.elem.Change = it.cv[8+2]&(1<<1) != 0
	return nil
}

func (it *creditIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.prefix)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if !bytes.HasPrefix(it.ck, it.prefix) {
		it.c = nil
		return false
	}

	err := it.readElem()
	if err != nil {
		it.err = err
		return false
	}
	return true
}

// The unspent index records all outpoints for mined credits which are not spent
// by any other mined transaction records (but may be spent by a mempool
// transaction).
//
// Keys are use the canonical outpoint serialization:
//
//   [0:32]  Transaction hash (32 bytes)
//   [32:36] Output index (4 bytes)
//
// Values are serialized as such:
//
//   [0:4]   Block height (4 bytes)
//   [4:36]  Block hash (32 bytes)

func valueUnspent(block *Block) []byte {
	v := make([]byte, 36)
	byteOrder.PutUint32(v, uint32(block.Order))
	copy(v[4:36], block.Hash[:])
	return v
}

func putUnspent(ns walletdb.ReadWriteBucket, outPoint *types.TxOutPoint, block *Block) error {
	k := canonicalOutPoint(&outPoint.Hash, outPoint.OutIndex)
	v := valueUnspent(block)
	err := ns.NestedReadWriteBucket(bucketUnspent).Put(k, v)
	if err != nil {
		str := "cannot put unspent"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func putRawUnspent(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnspent).Put(k, v)
	if err != nil {
		str := "cannot put unspent"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func readUnspentBlock(v []byte, block *Block) error {
	if len(v) < 36 {
		str := "short unspent value"
		return storeError(ErrData, str, nil)
	}
	block.Order = int32(byteOrder.Uint32(v))
	copy(block.Hash[:], v[4:36])
	return nil
}

// existsUnspent returns the key for the unspent output and the corresponding
// key for the credits bucket.  If there is no unspent output recorded, the
// credit key is nil.
func existsUnspent(ns walletdb.ReadBucket, outPoint *types.TxOutPoint) (k, credKey []byte) {
	k = canonicalOutPoint(&outPoint.Hash, outPoint.OutIndex)
	credKey = existsRawUnspent(ns, k)
	return k, credKey
}

// existsRawUnspent returns the credit key if there exists an output recorded
// for the raw unspent key.  It returns nil if the k/v pair does not exist.
func existsRawUnspent(ns walletdb.ReadBucket, k []byte) (credKey []byte) {
	if len(k) < 36 {
		return nil
	}
	v := ns.NestedReadBucket(bucketUnspent).Get(k)
	if len(v) < 36 {
		return nil
	}
	credKey = make([]byte, 72)
	copy(credKey, k[:32])
	copy(credKey[32:68], v)
	copy(credKey[68:72], k[32:36])
	return credKey
}

func deleteRawUnspent(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnspent).Delete(k)
	if err != nil {
		str := "failed to delete unspent"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

// All transaction debits (inputs which spend credits) are keyed as such:
//
//   [0:32]  Transaction hash (32 bytes)
//   [32:36] Block height (4 bytes)
//   [36:68] Block hash (32 bytes)
//   [68:72] Input index (4 bytes)
//
// The first 68 bytes match the key for the transaction record and may be used
// as a prefix filter to iterate through all debits in order.
//
// The debit value is serialized as such:
//
//   [0:8]   Amount (8 bytes)
//   [8:80]  Credits bucket key (72 bytes)
//             [8:40]  Transaction hash (32 bytes)
//             [40:44] Block height (4 bytes)
//             [44:76] Block hash (32 bytes)
//             [76:80] Output index (4 bytes)

func keyDebit(txHash *hash.Hash, index uint32, block *Block) []byte {
	k := make([]byte, 72)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], uint32(block.Order))
	copy(k[36:68], block.Hash[:])
	byteOrder.PutUint32(k[68:72], index)
	return k
}

func putDebit(ns walletdb.ReadWriteBucket, txHash *hash.Hash, index uint32, amount types.Amount, block *Block, credKey []byte) error {
	k := keyDebit(txHash, index, block)

	v := make([]byte, 80)
	byteOrder.PutUint64(v, uint64(amount.Value))
	copy(v[8:80], credKey)

	err := ns.NestedReadWriteBucket(bucketDebits).Put(k, v)
	if err != nil {
		str := fmt.Sprintf("failed to update debit %s input %d",
			txHash, index)
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func extractRawDebitCreditKey(v []byte) []byte {
	return v[8:80]
}

// existsDebit checks for the existance of a debit.  If found, the debit and
// previous credit keys are returned.  If the debit does not exist, both keys
// are nil.
func existsDebit(ns walletdb.ReadBucket, txHash *hash.Hash, index uint32, block *Block) (k, credKey []byte, err error) {
	k = keyDebit(txHash, index, block)
	v := ns.NestedReadBucket(bucketDebits).Get(k)
	if v == nil {
		return nil, nil, nil
	}
	if len(v) < 80 {
		str := fmt.Sprintf("%s: short read (expected 80 bytes, read %v)",
			bucketDebits, len(v))
		return nil, nil, storeError(ErrData, str, nil)
	}
	return k, v[8:80], nil
}

func deleteRawDebit(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketDebits).Delete(k)
	if err != nil {
		str := "failed to delete debit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

type debitIterator struct {
	c      walletdb.ReadWriteCursor // Set to nil after final iteration
	prefix []byte
	ck     []byte
	cv     []byte
	elem   DebitRecord
	err    error
}

func makeReadDebitIterator(ns walletdb.ReadBucket, prefix []byte) debitIterator {
	c := ns.NestedReadBucket(bucketDebits).ReadCursor()
	return debitIterator{c: readCursor{c}, prefix: prefix}
}

func (it *debitIterator) readElem() error {
	if len(it.ck) < 72+2 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketDebits, 72, len(it.ck))
		return storeError(ErrData, str, nil)
	}
	if len(it.cv) < 80+2 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketDebits, 80+2, len(it.cv))
		return storeError(ErrData, str, nil)
	}
	it.elem.Index = byteOrder.Uint32(it.ck[68+2 : 72+2])
	it.elem.Amount = types.Amount{Value: int64(byteOrder.Uint64(it.cv[0:8])), Id: types.CoinID(byteOrder.Uint16(it.cv[8:10]))}
	return nil
}

func (it *debitIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.prefix)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if !bytes.HasPrefix(it.ck, it.prefix) {
		it.c = nil
		return false
	}

	err := it.readElem()
	if err != nil {
		it.err = err
		return false
	}
	return true
}

// All unMined transactions are saved in the unMined bucket keyed by the
// transaction hash.  The value matches that of mined transaction records:
//
//   [0:8]   Received time (8 bytes)
//   [8:]    Serialized transaction (varies)

func putRawUnMined(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnMined).Put(k, v)
	if err != nil {
		str := "failed to put unMined record"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func readRawUnMinedHash(k []byte, txHash *hash.Hash) error {
	if len(k) < 32 {
		str := "short unMined key"
		return storeError(ErrData, str, nil)
	}
	copy(txHash[:], k)
	return nil
}

func existsRawUnMined(ns walletdb.ReadBucket, k []byte) (v []byte) {
	return ns.NestedReadBucket(bucketUnMined).Get(k)
}

func deleteRawUnmMined(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnMined).Delete(k)
	if err != nil {
		str := "failed to delete unMined record"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

// UnMined transaction credits use the canonical serialization format:
//
//  [0:32]   Transaction hash (32 bytes)
//  [32:36]  Output index (4 bytes)
//
// The value matches the format used by mined credits, but the spent flag is
// never set and the optional debit record is never included.  The simplified
// format is thus:
//
//   [0:8]   Amount (8 bytes)
//   [8:10]  Coin ID (2 bytes)
//   [8+2]     Flags (1 byte)
//             0x02: Change

func valueUnMinedCredit(amount types.Amount, change bool) []byte {
	v := make([]byte, 9+2)
	byteOrder.PutUint64(v, uint64(amount.Value))
	byteOrder.PutUint16(v, uint16(amount.Id))
	if change {
		v[8+2] = 1 << 1
	}
	return v
}

func putRawUnMinedCredit(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnMinedCredits).Put(k, v)
	if err != nil {
		str := "cannot put unMined credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func fetchRawUnMinedCreditIndex(k []byte) (uint32, error) {
	if len(k) < 36 {
		str := "short unMined credit key"
		return 0, storeError(ErrData, str, nil)
	}
	return byteOrder.Uint32(k[32:36]), nil
}

func fetchRawUnMinedCreditAmount(v []byte) (types.Amount, error) {
	if len(v) < 9+2 {
		str := "short unmMined credit value"
		return types.Amount{}, storeError(ErrData, str, nil)
	}
	a := types.Amount{Value: int64(byteOrder.Uint64(v[0:8])), Id: types.CoinID(byteOrder.Uint16(v[8:10]))}
	return a, nil
}

func fetchRawUnMinedCreditAmountChange(v []byte) (types.Amount, bool, error) {
	if len(v) < 9+2 {
		str := "short unmMined credit value"
		return types.Amount{}, false, storeError(ErrData, str, nil)
	}
	amt := types.Amount{Value: int64(byteOrder.Uint64(v)), Id: types.CoinID(byteOrder.Uint16(v[8:10]))}
	change := v[8+2]&(1<<1) != 0
	return amt, change, nil
}

func existsRawUnMinedCredit(ns walletdb.ReadBucket, k []byte) []byte {
	return ns.NestedReadBucket(bucketUnMinedCredits).Get(k)
}

func deleteRawUnMinedCredit(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnMinedCredits).Delete(k)
	if err != nil {
		str := "failed to delete unmMined credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

type unMinedCreditIterator struct {
	c      walletdb.ReadWriteCursor
	prefix []byte
	ck     []byte
	cv     []byte
	elem   CreditRecord
	err    error
}

type readCursor struct {
	walletdb.ReadCursor
}

func (r readCursor) Delete() error {
	str := "failed to delete current cursor item from read-only cursor"
	return storeError(ErrDatabase, str, walletdb.ErrTxNotWritable)
}

func makeUnMinedCreditIterator(ns walletdb.ReadWriteBucket, txHash *hash.Hash) unMinedCreditIterator {
	c := ns.NestedReadWriteBucket(bucketUnMinedCredits).ReadWriteCursor()
	return unMinedCreditIterator{c: c, prefix: txHash[:]}
}

func makeReadUnMinedCreditIterator(ns walletdb.ReadBucket, txHash *hash.Hash) unMinedCreditIterator {
	c := ns.NestedReadBucket(bucketUnMinedCredits).ReadCursor()
	return unMinedCreditIterator{c: readCursor{c}, prefix: txHash[:]}
}

func (it *unMinedCreditIterator) readElem() error {
	index, err := fetchRawUnMinedCreditIndex(it.ck)
	if err != nil {
		return err
	}
	amount, change, err := fetchRawUnMinedCreditAmountChange(it.cv)
	if err != nil {
		return err
	}

	it.elem.Index = index
	it.elem.Amount = amount
	it.elem.Change = change

	return nil
}

func (it *unMinedCreditIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.prefix)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if !bytes.HasPrefix(it.ck, it.prefix) {
		it.c = nil
		return false
	}

	err := it.readElem()
	if err != nil {
		it.err = err
		return false
	}
	return true
}

func (it *unMinedCreditIterator) reposition(txHash *hash.Hash, index uint32) {
	it.c.Seek(canonicalOutPoint(txHash, index))
}

// putRawUnMinedInput maintains a list of unMined transaction hashes that have
// spent an outpoint. Each entry in the bucket is keyed by the outpoint being
// spent.
func putRawUnMinedInput(ns walletdb.ReadWriteBucket, k, v []byte) error {
	spendTxHashes := ns.NestedReadBucket(bucketUnMinedInputs).Get(k)
	spendTxHashes = append(spendTxHashes, v...)
	err := ns.NestedReadWriteBucket(bucketUnMinedInputs).Put(k, spendTxHashes)
	if err != nil {
		str := "failed to put unMined input"
		return storeError(ErrDatabase, str, err)
	}

	return nil
}

func existsRawUnMinedInput(ns walletdb.ReadBucket, k []byte) (v []byte) {
	return ns.NestedReadBucket(bucketUnMinedInputs).Get(k)
}

// fetchUnMinedInputSpendTxHashes fetches the list of unMined transactions that
// spend the serialized outpoint.
func fetchUnMinedInputSpendTxHashes(ns walletdb.ReadBucket, k []byte) []hash.Hash {
	rawSpendTxHashes := ns.NestedReadBucket(bucketUnMinedInputs).Get(k)
	if rawSpendTxHashes == nil {
		return nil
	}

	// Each transaction hash is 32 bytes.
	spendTxHashes := make([]hash.Hash, 0, len(rawSpendTxHashes)/32)
	for len(rawSpendTxHashes) > 0 {
		var spendTxHash hash.Hash
		copy(spendTxHash[:], rawSpendTxHashes[:32])
		spendTxHashes = append(spendTxHashes, spendTxHash)
		rawSpendTxHashes = rawSpendTxHashes[32:]
	}

	return spendTxHashes
}

func deleteRawUnMinedInput(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnMinedInputs).Delete(k)
	if err != nil {
		str := "failed to delete unMined input"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

// openStore opens an existing transaction store from the passed namespace.
func openStore(ns walletdb.ReadBucket) error {
	version, err := fetchVersion(ns)
	if err != nil {
		return err
	}

	latestVersion := getLatestVersion()
	if version < latestVersion {
		str := fmt.Sprintf("a database upgrade is required to upgrade "+
			"wtxmgr from recorded version %d to the latest version %d",
			version, latestVersion)
		return storeError(ErrNeedsUpgrade, str, nil)
	}

	if version > latestVersion {
		str := fmt.Sprintf("version recorded version %d is newer that "+
			"latest understood version %d", version, latestVersion)
		return storeError(ErrUnknownVersion, str, nil)
	}

	return nil
}

// createStore creates the tx store (with the latest db version) in the passed
// namespace.  If a store already exists, ErrAlreadyExists is returned.
func createStore(ns walletdb.ReadWriteBucket) error {
	// Ensure that nothing currently exists in the namespace bucket.
	ck, cv := ns.ReadCursor().First()
	if ck != nil || cv != nil {
		const str = "namespace is not empty"
		return storeError(ErrAlreadyExists, str, nil)
	}

	// Write the latest store version.
	if err := putVersion(ns, getLatestVersion()); err != nil {
		return err
	}

	// Save the creation date of the store.
	var v [8]byte
	byteOrder.PutUint64(v[:], uint64(time.Now().Unix()))
	err := ns.Put(rootCreateDate, v[:])
	if err != nil {
		str := "failed to store database creation time"
		return storeError(ErrDatabase, str, err)
	}

	// Write a zero balance.
	byteOrder.PutUint64(v[:], 0)
	err = ns.Put(rootMinedBalanceKey(types.MEERA), v[:])
	if err != nil {
		str := "failed to write zero balance"
		return storeError(ErrDatabase, str, err)
	}

	// Finally, create all of our required descendant buckets.
	return createBuckets(ns)
}

// createBuckets creates all of the descendants buckets required for the
// transaction store to properly carry its duties.
func createBuckets(ns walletdb.ReadWriteBucket) error {
	if _, err := ns.CreateBucket(bucketBlocks); err != nil {
		str := "failed to create blocks bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketTxRecords); err != nil {
		str := "failed to create tx records bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketCredits); err != nil {
		str := "failed to create credits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketDebits); err != nil {
		str := "failed to create debits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnspent); err != nil {
		str := "failed to create unspent bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnMined); err != nil {
		str := "failed to create unMined bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnMinedCredits); err != nil {
		str := "failed to create unMined credits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnMinedInputs); err != nil {
		str := "failed to create unMined inputs bucket"
		return storeError(ErrDatabase, str, err)
	}

	for _, id := range types.CoinIDList {
		if _, err := ns.CreateBucket(CoinBucket(BucketAddrtxin, id)); err != nil {
			str := fmt.Sprintf("failed to create unMined %s addrtxin bucket", id.Name())
			return storeError(ErrDatabase, str, err)
		}
		if _, err := ns.CreateBucket(CoinBucket(BucketAddrtxout, id)); err != nil {
			str := fmt.Sprintf("failed to create unMined %s addrtxout bucket", id.Name())
			return storeError(ErrDatabase, str, err)
		}
	}

	if _, err := ns.CreateBucket(BucketTxJson); err != nil {
		str := "failed to create unMined BucketTxJson bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(BucketUnConfirmed); err != nil {
		str := "failed to create unconfirmed bucket"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

// deleteBuckets deletes all of the descendants buckets required for the
// transaction store to properly carry its duties.
func deleteBuckets(ns walletdb.ReadWriteBucket) error {
	ns.DeleteNestedBucket(bucketBlocks)
	ns.DeleteNestedBucket(bucketTxRecords)
	ns.DeleteNestedBucket(bucketCredits)
	ns.DeleteNestedBucket(bucketDebits)
	ns.DeleteNestedBucket(bucketUnspent)
	ns.DeleteNestedBucket(bucketUnMined)
	ns.DeleteNestedBucket(bucketUnMinedCredits)
	ns.DeleteNestedBucket(bucketUnMinedInputs)

	for _, id := range types.CoinIDList {
		err := ns.DeleteNestedBucket(CoinBucket(BucketAddrtxin, id))
		fmt.Println(err)
		err = ns.DeleteNestedBucket(CoinBucket(BucketAddrtxout, id))
		fmt.Println(err)
	}

	ns.DeleteNestedBucket(BucketTxJson)
	ns.DeleteNestedBucket(BucketUnConfirmed)
	return nil
}

func DropTransactionHistory(ns walletdb.ReadWriteBucket) error {

	// To drop the store's transaction history, we'll need to remove all of
	// the relevant descendant buckets and key/value pairs.
	if err := deleteBuckets(ns); err != nil {
		return err
	}
	//ns.Delete(rootMinedBalance)

	// With everything removed, we'll now recreate our buckets.
	///if err := createBuckets(ns); err != nil {
	//return err
	//}

	// Finally, we'll insert a 0 value for our mined balance.
	return nil
}

func CoinBucket(bucket []byte, coin types.CoinID) []byte {
	return append([]byte{byte(coin)}, bucket...)
}

// putVersion modifies the version of the store to reflect the given version
// number.
func putVersion(ns walletdb.ReadWriteBucket, version uint32) error {
	var v [4]byte
	byteOrder.PutUint32(v[:], version)
	if err := ns.Put(rootVersion, v[:]); err != nil {
		str := "failed to store database version"
		return storeError(ErrDatabase, str, err)
	}

	return nil
}

// fetchVersion fetches the current version of the store.
func fetchVersion(ns walletdb.ReadBucket) (uint32, error) {
	v := ns.Get(rootVersion)
	if len(v) != 4 {
		str := "no transaction store exists in namespace"
		return 0, storeError(ErrNoExists, str, nil)
	}

	return byteOrder.Uint32(v), nil
}

func Uint64ToBytes(value uint64) []byte {
	var bigE = binary.BigEndian
	valueBytes := make([]byte, 8)
	bigE.PutUint64(valueBytes, value)
	return valueBytes
}

func BytesToUin64(bytes []byte) uint64 {
	var bigE = binary.BigEndian
	value := bigE.Uint64(bytes)
	return value
}
