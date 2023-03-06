package wtxmgr

import (
	"encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/common/math"
	corejson "github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/services/hotwallet/utils"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
	"time"

	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
)

// Block contains the minimum amount of data to uniquely identify any block on
// either the best or side chain.
type Block struct {
	Hash  hash.Hash
	Order int32
}

// BlockMeta contains the unique identification for a block and any metadata
// pertaining to the block.  At the moment, this additional metadata only
// includes the block time from the block header.
type BlockMeta struct {
	Block
	Time time.Time
}

// blockRecord is an in-memory representation of the block record saved in the
// database.
type blockRecord struct {
	Block
	Time         time.Time
	transactions []hash.Hash
}

type SpendTo struct {
	Index uint32
	TxId  hash.Hash
	//Block  Block
}

type TxInputPoint struct {
	TxOutPoint types.TxOutPoint
	SpendTo    SpendTo
}

type AddrTxOutput struct {
	Address  string
	TxId     hash.Hash
	Index    uint32
	Amount   types.Amount
	Block    Block
	Spend    SpendStatus
	SpendTo  *SpendTo
	Status   TxStatus
	Locked   uint32
	IsBlue   bool
	PkScript string
}

func NewAddrTxOutput() *AddrTxOutput {
	return &AddrTxOutput{SpendTo: &SpendTo{}}
}

func (a *AddrTxOutput) Encode() []byte {
	bytes, _ := utils.Encode(a)
	return bytes
}

func DecodeAddrTxOutput(bytes []byte) (*AddrTxOutput, error) {
	output := &AddrTxOutput{}
	return output, utils.Decode(bytes, output)
}

type AddrTxOutputs []AddrTxOutput

func (s AddrTxOutputs) Len() int { return len(s) }

func (s AddrTxOutputs) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s AddrTxOutputs) Less(i, j int) bool {
	return s[i].Block.Order < s[j].Block.Order
}

type TxConfirmed struct {
	TxId          string
	Confirmations uint32
	TxStatus
}

type SpendStatus uint16

const (
	SpendStatusUnspent SpendStatus = 0
	SpendStatusSpend   SpendStatus = 1
)

type TxStatus uint16

const (
	TxStatusMemPool     TxStatus = 0
	TxStatusUnConfirmed TxStatus = 1
	TxStatusConfirmed   TxStatus = 2
	TxStatusFailed      TxStatus = 3
	TxStatusRead        TxStatus = 4
)

type UnconfirmTx struct {
	Order         uint32
	Confirmations uint32
}

func (u *UnconfirmTx) Marshal() []byte {
	bytes, _ := json.Marshal(u)
	return bytes
}

func UnMarshalUnconfirmTx(bytes []byte) (*UnconfirmTx, error) {
	u := &UnconfirmTx{}
	err := json.Unmarshal(bytes, u)
	return u, err
}

type UTxo struct {
	Address string
	TxId    string
	Index   uint32
	Amount  types.Amount
}

// incidence records the block hash and blockchain height of a mined transaction.
// Since a transaction hash alone is not enough to uniquely identify a mined
// transaction (duplicate transaction hashes are allowed), the incidence is used
// instead.
type incidence struct {
	txHash hash.Hash
	block  Block
}

// indexedIncidence records the transaction incidence and an input or output
// index.
type indexedIncidence struct {
	incidence
	index uint32
}

// credit describes a transaction output which was or is spendable by wallet.
type credit struct {
	outPoint types.TxOutPoint
	block    Block
	amount   types.Amount
	change   bool
	spentBy  indexedIncidence // Index == ^uint32(0) if unspent
}

// TxRecord represents a transaction managed by the Store.
type TxRecord struct {
	MsgTx        types.Transaction
	Hash         hash.Hash
	Received     time.Time
	SerializedTx []byte // Optional: may be nil
}

// Credit is the type representing a transaction output which was spent or
// is still spendable by wallet.  A UTXO is an unspent Credit, but not all
// Credits are UTXOs.
type Credit struct {
	types.TxOutPoint
	BlockMeta
	Amount       types.Amount
	PkScript     []byte
	Received     time.Time
	FromCoinBase bool
}

// Store implements a transaction store for storing and managing wallet
// transactions.
type Store struct {
	chainParams *params.Params

	// Event callbacks.  These execute in the same goroutine as the wtxmgr
	// caller.
	NotifyUnspent func(hash *hash.Hash, index uint32)
}

// Open opens the wallet transaction store from a walletdb namespace.  If the
// store does not exist, ErrNoExist is returned.
func Open(ns walletdb.ReadBucket, chainParams *params.Params) (*Store, error) {
	// Open the store.
	err := openStore(ns)
	if err != nil {
		return nil, err
	}
	s := &Store{chainParams, nil} // TODO: set callbacks
	return s, nil
}

// Create creates a new persistent transaction store in the walletdb namespace.
// Creating the store when one already exists in this namespace will error with
// ErrAlreadyExists.
func Create(ns walletdb.ReadWriteBucket) error {
	return createStore(ns)
}

func (s *Store) UpdateAddrTxIn(ns walletdb.ReadWriteBucket, addr string, outPoint *types.TxOutPoint) error {
	inRw, err := ns.CreateBucketIfNotExists([]byte(addr))
	if err != nil {
		return err
	} else {
		v := canonicalOutPoint(&outPoint.Hash, outPoint.OutIndex)
		err := inRw.Put(v, v)
		return err
	}
}

func (s *Store) InsertAddrTxOut(ns walletdb.ReadWriteBucket, txOut *AddrTxOutput) error {
	outRw, err := ns.CreateBucketIfNotExists([]byte(txOut.Address))
	if err != nil {
		return err
	} else {
		k := canonicalOutPoint(&txOut.TxId, txOut.Index)
		v := txOut.Encode()
		oldTxOut := outRw.Get(k)
		if oldTxOut == nil || len(oldTxOut) == 0 {
			err := outRw.Put(k, v)
			return err
		} else {
			addTxOutPut := NewAddrTxOutput()
			addTxOutPut, err := DecodeAddrTxOutput(oldTxOut)
			if err != nil {
				return err
			}
			if addTxOutPut.Spend != SpendStatusSpend {
				err := outRw.Put(k, v)
				return err
			} else {
				txOut.SpendTo = addTxOutPut.SpendTo
				txOut.Spend = addTxOutPut.Spend
				v := txOut.Encode()
				err := outRw.Put(k, v)
				return err
			}
		}
	}
}
func (s *Store) UpdateAddrTxOut(ns walletdb.ReadWriteBucket, txOut *AddrTxOutput) error {
	outRw, err := ns.CreateBucketIfNotExists([]byte(txOut.Address))
	if err != nil {
		return err
	} else {
		k := canonicalOutPoint(&txOut.TxId, txOut.Index)
		v := txOut.Encode()
		err := outRw.Put(k, v)
		return err
	}
}
func (s *Store) GetAddrTxOut(ns walletdb.ReadWriteBucket, address string, point types.TxOutPoint) (*AddrTxOutput, error) {
	outRw := ns.NestedReadWriteBucket([]byte(address))
	k := canonicalOutPoint(&point.Hash, point.OutIndex)
	txOut := outRw.Get(k)
	addTxOutPut := NewAddrTxOutput()
	addTxOutPut, err := DecodeAddrTxOutput(txOut)
	if err != nil {
		return nil, err
	}
	return addTxOutPut, nil
}

// updateMinedBalance updates the mined balance within the store, if changed,
// after processing the given transaction record.
func (s *Store) updateMinedBalance(ns walletdb.ReadWriteBucket, rec *TxRecord,
	block *BlockMeta) error {

	// Fetch the mined balance in case we need to update it.
	minedBalance, err := fetchMinedBalance(ns, types.MEERA)
	if err != nil {
		return err
	}

	// Add a debit record for each unspent credit spent by this transaction.
	// The index is set in each iteration below.
	spender := indexedIncidence{
		incidence: incidence{
			txHash: rec.Hash,
			block:  block.Block,
		},
	}

	newMinedBalance := minedBalance
	for i, input := range rec.MsgTx.TxIn {
		unspentKey, credKey := existsUnspent(ns, &input.PreviousOut)
		if credKey == nil {
			continue
		}

		// If this output is relevant to us, we'll mark the it as spent
		// and remove its amount from the store.
		spender.index = uint32(i)
		amt, err := spendCredit(ns, credKey, &spender)
		if err != nil {
			return err
		}
		err = putDebit(
			ns, &rec.Hash, uint32(i), amt, &block.Block, credKey,
		)
		if err != nil {
			return err
		}
		if err := deleteRawUnspent(ns, unspentKey); err != nil {
			return err
		}

		newMinedBalance.Value -= amt.Value
	}

	// For each output of the record that is marked as a credit, if the
	// output is marked as a credit by the unconfirmed store, remove the
	// marker and mark the output as a credit in the db.
	//
	// Moved credits are added as unspents, even if there is another
	// unconfirmed transaction which spends them.
	cred := credit{
		outPoint: types.TxOutPoint{Hash: rec.Hash},
		block:    block.Block,
		spentBy:  indexedIncidence{index: ^uint32(0)},
	}

	it := makeUnMinedCreditIterator(ns, &rec.Hash)
	for it.next() {
		// can be moved from unMined directly to the credits bucket.
		// The key needs a modification to include the block
		// height/hash.
		index, err := fetchRawUnMinedCreditIndex(it.ck)
		if err != nil {
			return err
		}
		amount, change, err := fetchRawUnMinedCreditAmountChange(it.cv)
		if err != nil {
			return err
		}

		cred.outPoint.OutIndex = index
		cred.amount = amount
		cred.change = change

		if err := putUnspentCredit(ns, &cred); err != nil {
			return err
		}
		err = putUnspent(ns, &cred.outPoint, &block.Block)
		if err != nil {
			return err
		}

		newMinedBalance.Value += amount.Value
	}
	if it.err != nil {
		return it.err
	}

	// Update the balance if it has changed.
	if newMinedBalance != minedBalance {
		return putMinedBalance(ns, newMinedBalance)
	}

	return nil
}

// deleteUnMinedTx deletes an unMined transaction from the store.
//
// NOTE: This should only be used once the transaction has been mined.
func (s *Store) deleteUnMinedTx(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		if err := deleteRawUnMinedCredit(ns, k); err != nil {
			return err
		}
	}

	return deleteRawUnmMined(ns, rec.Hash[:])
}

// InsertTx records a transaction as belonging to a wallet's transaction
// history.  If block is nil, the transaction is considered unspent, and the
// transaction's index must be unset.
func (s *Store) InsertTx(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta) error {
	if block == nil {
		return s.insertMemPoolTx(ns, rec)
	}
	return s.insertMinedTx(ns, rec, block)
}

// RemoveUnMinedTx attempts to remove an unMined transaction from the
// transaction store. This is to be used in the scenario that a transaction
// that we attempt to rebroadcast, turns out to double spend one of our
// existing inputs. This function we remove the conflicting transaction
// identified by the tx record, and also recursively remove all transactions
// that depend on it.
func (s *Store) RemoveUnMinedTx(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
	// As we already have a tx record, we can directly call the
	// removeConflict method. This will do the job of recursively removing
	// this unMined transaction, and any transactions that depend on it.
	return s.removeConflict(ns, rec)
}

// insertMinedTx inserts a new transaction record for a mined transaction into
// the database under the confirmed bucket. It guarantees that, if the
// tranasction was previously unconfirmed, then it will take care of cleaning up
// the unconfirmed state. All other unconfirmed double spend attempts will be
// removed as well.
func (s *Store) insertMinedTx(ns walletdb.ReadWriteBucket, rec *TxRecord,
	block *BlockMeta) error {

	// If a transaction record for this hash and block already exists, we
	// can exit early.
	if _, v := existsTxRecord(ns, &rec.Hash, &block.Block); v != nil {
		return nil
	}

	// If a block record does not yet exist for any transactions from this
	// block, insert a block record first. Otherwise, update it by adding
	// the transaction hash to the set of transactions from this block.
	var err error
	blockKey, blockValue := existsBlockRecord(ns, uint32(block.Order))
	if blockValue == nil {
		err = putBlockRecord(ns, block, &rec.Hash)
	} else {
		blockValue, err = appendRawBlockRecord(blockValue, &rec.Hash)
		if err != nil {
			return err
		}
		err = putRawBlockRecord(ns, blockKey, blockValue)
	}
	if err != nil {
		return err
	}
	if err := putTxRecord(ns, rec, &block.Block); err != nil {
		return err
	}

	// Determine if this transaction has affected our balance, and if so,
	// update it.
	if err := s.updateMinedBalance(ns, rec, block); err != nil {
		return err
	}

	// If this transaction previously existed within the store as unMined,
	// we'll need to remove it from the unMined bucket.
	if v := existsRawUnMined(ns, rec.Hash[:]); v != nil {
		log.Info(fmt.Sprintf("Marking unconfirmed transaction %v mined in block %d",
			&rec.Hash, block.Order))

		if err := s.deleteUnMinedTx(ns, rec); err != nil {
			return err
		}
	}

	// As there may be unconfirmed transactions that are invalidated by this
	// transaction (either being duplicates, or double spends), remove them
	// from the unconfirmed set.  This also handles removing unconfirmed
	// transaction spend chains if any other unconfirmed transactions spend
	// outputs of the removed double spend.
	return s.removeDoubleSpends(ns, rec)
}

// AddCredit marks a transaction record as containing a transaction output
// spendable by wallet.  The output is added unspent, and is marked spent
// when a new transaction spending the output is inserted into the store.
func (s *Store) AddCredit(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32, change bool) error {
	if int(index) >= len(rec.MsgTx.TxOut) {
		str := "transaction output does not exist"
		return storeError(ErrInput, str, nil)
	}

	isNew, err := s.addCredit(ns, rec, block, index, change)
	if err == nil && isNew && s.NotifyUnspent != nil {
		s.NotifyUnspent(&rec.Hash, index)
	}
	return err
}

// addCredit is an AddCredit helper that runs in an update transaction.  The
// bool return specifies whether the unspent output is newly added (true) or a
// duplicate (false).
func (s *Store) addCredit(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32, change bool) (bool, error) {
	if block == nil {

		k := canonicalOutPoint(&rec.Hash, index)
		if existsRawUnMinedCredit(ns, k) != nil {
			return false, nil
		}
		if existsRawUnspent(ns, k) != nil {
			return false, nil
		}
		v := valueUnMinedCredit(rec.MsgTx.TxOut[index].Amount, change)
		return true, putRawUnMinedCredit(ns, k, v)
	}

	k, v := existsCredit(ns, &rec.Hash, index, &block.Block)
	if v != nil {
		return false, nil
	}

	txOutAmt := rec.MsgTx.TxOut[index].Amount
	log.Debug("Marking transaction %v output %d (%v) spendable",
		rec.Hash, index, txOutAmt)

	cred := credit{
		outPoint: types.TxOutPoint{
			Hash:     rec.Hash,
			OutIndex: index,
		},
		block:   block.Block,
		amount:  txOutAmt,
		change:  change,
		spentBy: indexedIncidence{index: ^uint32(0)},
	}
	v = valueUnspentCredit(&cred)
	err := putRawCredit(ns, k, v)
	if err != nil {
		return false, err
	}

	minedBalance, err := fetchMinedBalance(ns, txOutAmt.Id)
	if err != nil {
		return false, err
	}
	a := types.Amount{Value: minedBalance.Value + txOutAmt.Value, Id: txOutAmt.Id}
	err = putMinedBalance(ns, a)
	if err != nil {
		return false, err
	}

	return true, putUnspent(ns, &cred.outPoint, &block.Block)
}

// Rollback removes all blocks at height onwards, moving any transactions within
// each block to the unconfirmed pool.
func (s *Store) Rollback(ns walletdb.ReadWriteBucket, height int32) error {
	return s.rollback(ns, height)
}

var (
	// zeroHash is the zero value for a hash.Hash and is defined as a
	// package level variable to avoid the need to create a new instance
	// every time a check is needed.
	zeroHash = &hash.ZeroHash
)

func IsCoinBaseTx(tx *types.Transaction) bool {
	// A coin base must only have one transaction input.
	if len(tx.TxIn) != 1 {
		return false
	}
	// The previous output of a coin base must have a max value index and a
	// zero hash.
	prevOut := &tx.TxIn[0].PreviousOut
	if prevOut.OutIndex != math.MaxUint32 || !prevOut.Hash.IsEqual(zeroHash) {
		return false
	}
	return true
}

func TxRawIsCoinBase(tx corejson.TxRawResult) bool {
	for _, vin := range tx.Vin {
		if vin.Coinbase != "" {
			return true
		}
	}
	return false
}

func (s *Store) rollback(ns walletdb.ReadWriteBucket, height int32) error {
	minedBalance, err := fetchMinedBalance(ns, types.MEERA)
	if err != nil {
		return err
	}

	// Keep track of all credits that were removed from coinbase
	// transactions.  After detaching all blocks, if any transaction record
	// exists in unMined that spends these outputs, remove them and their
	// spend chains.
	//
	// It is necessary to keep these in memory and fix the unMined
	// transactions later since blocks are removed in increasing order.
	var coinBaseCredits []types.TxOutPoint
	var heightsToRemove []int32

	it := makeReverseBlockIterator(ns)
	for it.prev() {
		b := &it.elem
		if it.elem.Order < height {
			break
		}

		heightsToRemove = append(heightsToRemove, it.elem.Order)

		log.Info("Rolling back transactions", "transactions", len(b.transactions), "block", b.Hash, "height", b.Order)

		for i := range b.transactions {
			txHash := &b.transactions[i]

			recKey := keyTxRecord(txHash, &b.Block)
			recVal := existsRawTxRecord(ns, recKey)
			var rec TxRecord
			err = readRawTxRecord(txHash, recVal, &rec)
			if err != nil {
				return err
			}

			err = deleteTxRecord(ns, txHash, &b.Block)
			if err != nil {
				return err
			}

			// Handle coinbase transactions specially since they are
			// not moved to the unconfirmed store.  A coinbase cannot
			// contain any debits, but all credits should be removed
			// and the mined balance decremented.
			if IsCoinBaseTx(&rec.MsgTx) {
				op := types.TxOutPoint{Hash: rec.Hash}
				for i, output := range rec.MsgTx.TxOut {
					k, v := existsCredit(ns, &rec.Hash,
						uint32(i), &b.Block)
					if v == nil {
						continue
					}
					op.OutIndex = uint32(i)

					coinBaseCredits = append(coinBaseCredits, op)

					unspentKey, credKey := existsUnspent(ns, &op)
					if credKey != nil {
						minedBalance.Value -= output.Amount.Value
						err = deleteRawUnspent(ns, unspentKey)
						if err != nil {
							return err
						}
					}
					err = deleteRawCredit(ns, k)
					if err != nil {
						return err
					}
				}

				continue
			}

			err = putRawUnMined(ns, txHash[:], recVal)
			if err != nil {
				return err
			}

			// For each debit recorded for this transaction, mark
			// the credit it spends as unspent (as long as it still
			// exists) and delete the debit.  The previous output is
			// recorded in the unconfirmed store for every previous
			// output, not just debits.
			for i, input := range rec.MsgTx.TxIn {
				prevOut := &input.PreviousOut
				prevOutKey := canonicalOutPoint(&prevOut.Hash,
					prevOut.OutIndex)
				err = putRawUnMinedInput(ns, prevOutKey, rec.Hash[:])
				if err != nil {
					return err
				}

				// If this input is a debit, remove the debit
				// record and mark the credit that it spent as
				// unspent, incrementing the mined balance.
				debKey, credKey, err := existsDebit(ns,
					&rec.Hash, uint32(i), &b.Block)
				if err != nil {
					return err
				}
				if debKey == nil {
					continue
				}

				// unspendRawCredit does not error in case the
				// no credit exists for this key, but this
				// behavior is correct.  Since blocks are
				// removed in increasing order, this credit
				// may have already been removed from a
				// previously removed transaction record in
				// this rollback.
				var amt types.Amount
				amt, err = unspendRawCredit(ns, credKey)
				if err != nil {
					return err
				}
				err = deleteRawDebit(ns, debKey)
				if err != nil {
					return err
				}

				// If the credit was previously removed in the
				// rollback, the credit amount is zero.  Only
				// mark the previously spent credit as unspent
				// if it still exists.
				if amt.Value == 0 {
					continue
				}
				unspentVal, err := fetchRawCreditUnspentValue(credKey)
				if err != nil {
					return err
				}
				minedBalance.Value += amt.Value
				err = putRawUnspent(ns, prevOutKey, unspentVal)
				if err != nil {
					return err
				}
			}

			// For each detached non-coinbase credit, move the
			// credit output to unMined.  If the credit is marked
			// unspent, it is removed from the utxo set and the
			// mined balance is decremented.
			//
			// TODO: use a credit iterator
			for i, output := range rec.MsgTx.TxOut {
				k, v := existsCredit(ns, &rec.Hash, uint32(i),
					&b.Block)
				if v == nil {
					continue
				}

				amt, change, err := fetchRawCreditAmountChange(v)
				if err != nil {
					return err
				}
				outPointKey := canonicalOutPoint(&rec.Hash, uint32(i))
				unMinedCredVal := valueUnMinedCredit(amt, change)
				err = putRawUnMinedCredit(ns, outPointKey, unMinedCredVal)
				if err != nil {
					return err
				}

				err = deleteRawCredit(ns, k)
				if err != nil {
					return err
				}

				credKey := existsRawUnspent(ns, outPointKey)
				if credKey != nil {
					minedBalance.Value -= output.Amount.Value
					err = deleteRawUnspent(ns, outPointKey)
					if err != nil {
						return err
					}
				}
			}
		}

		it.reposition(uint32(it.elem.Order))

	}
	if it.err != nil {
		return it.err
	}

	// Delete the block records outside of the iteration since cursor deletion
	// is broken.
	for _, h := range heightsToRemove {
		err = deleteBlockRecord(ns, uint32(h))
		if err != nil {
			return err
		}
	}

	for _, op := range coinBaseCredits {
		opKey := canonicalOutPoint(&op.Hash, op.OutIndex)
		unMinedSpendTxHashKeys := fetchUnMinedInputSpendTxHashes(ns, opKey)
		for _, unMinedSpendTxHashKey := range unMinedSpendTxHashKeys {
			unMinedVal := existsRawUnMined(ns, unMinedSpendTxHashKey[:])

			if unMinedVal == nil {
				continue
			}

			var unMinedRec TxRecord
			unMinedRec.Hash = unMinedSpendTxHashKey
			err = readRawTxRecord(&unMinedRec.Hash, unMinedVal, &unMinedRec)
			if err != nil {
				return err
			}

			log.Debug("Transaction %v spends a removed coinbase "+
				"output -- removing as well", unMinedRec.Hash)
			err = s.removeConflict(ns, &unMinedRec)
			if err != nil {
				return err
			}
		}
	}

	return putMinedBalance(ns, minedBalance)
}

// UnspentOutputs returns all unspent received transaction outputs.
// The order is undefined.
func (s *Store) UnspentOutputs(ns walletdb.ReadBucket) ([]Credit, error) {
	var unspent []Credit

	var op types.TxOutPoint
	var block Block
	err := ns.NestedReadBucket(bucketUnspent).ForEach(func(k, v []byte) error {
		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}
		if existsRawUnMinedInput(ns, k) != nil {
			// Output is spent by an unMined transaction.
			// Skip this k/v pair.
			return nil
		}
		err = readUnspentBlock(v, &block)
		if err != nil {
			return err
		}

		blockTime, err := fetchBlockTime(ns, uint32(block.Order))
		if err != nil {
			return err
		}
		// TODO(jrick): reading the entire transaction should
		// be avoidable.  Creating the credit only requires the
		// output amount and pkScript.
		rec, err := fetchTxRecord(ns, &op.Hash, &block)
		if err != nil {
			return err
		}
		txOut := rec.MsgTx.TxOut[op.OutIndex]
		cred := Credit{
			TxOutPoint: op,
			BlockMeta: BlockMeta{
				Block: block,
				Time:  blockTime,
			},
			Amount:       types.Amount(txOut.Amount),
			PkScript:     txOut.PkScript,
			Received:     rec.Received,
			FromCoinBase: IsCoinBaseTx(&rec.MsgTx),
		}
		unspent = append(unspent, cred)
		return nil
	})
	if err != nil {
		if _, ok := err.(Error); ok {
			return nil, err
		}
		str := "failed iterating unspent bucket"
		return nil, storeError(ErrDatabase, str, err)
	}

	err = ns.NestedReadBucket(bucketUnMinedCredits).ForEach(func(k, v []byte) error {
		if existsRawUnMinedInput(ns, k) != nil {
			// Output is spent by an unMined transaction.
			// Skip to next unMined credit.
			return nil
		}

		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}

		// just for the output amount and script can be avoided.
		recVal := existsRawUnMined(ns, op.Hash[:])
		var rec TxRecord
		err = readRawTxRecord(&op.Hash, recVal, &rec)
		if err != nil {
			return err
		}

		txOut := rec.MsgTx.TxOut[op.OutIndex]
		cred := Credit{
			TxOutPoint: op,
			BlockMeta: BlockMeta{
				Block: Block{Order: -1},
			},
			Amount:       types.Amount(txOut.Amount),
			PkScript:     txOut.PkScript,
			Received:     rec.Received,
			FromCoinBase: IsCoinBaseTx(&rec.MsgTx),
		}
		unspent = append(unspent, cred)
		return nil
	})
	if err != nil {
		if _, ok := err.(Error); ok {
			return nil, err
		}
		str := "failed iterating unMined credits bucket"
		return nil, storeError(ErrDatabase, str, err)
	}

	return unspent, nil
}

// Balance returns the spendable wallet balance (total value of all unspent
// transaction outputs) given a minimum of minConf confirmations, calculated
// at a current chain height of curHeight.  Coinbase outputs are only included
// in the balance if maturity has been reached.
//
// Balance may return unexpected results if syncHeight is lower than the block
// height of the most recent mined transaction in the store.
func (s *Store) Balance(ns walletdb.ReadBucket, minConf int32, syncOrder int32, coinId types.CoinID) (types.Amount, error) {
	bal, err := fetchMinedBalance(ns, coinId)
	if err != nil {
		return types.Amount{}, err
	}

	// Subtract the balance for each credit that is spent by an unMined
	// transaction.
	var op types.TxOutPoint
	var block Block
	err = ns.NestedReadBucket(bucketUnspent).ForEach(func(k, v []byte) error {
		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}
		err = readUnspentBlock(v, &block)
		if err != nil {
			return err
		}
		if existsRawUnMinedInput(ns, k) != nil {
			_, v := existsCredit(ns, &op.Hash, op.OutIndex, &block)
			amt, err := fetchRawCreditAmount(v)
			if err != nil {
				return err
			}
			bal.Value -= amt.Value
		}
		return nil
	})
	if err != nil {
		if _, ok := err.(Error); ok {
			return types.Amount{}, err
		}
		str := "failed iterating unspent outputs"
		return types.Amount{}, storeError(ErrDatabase, str, err)
	}

	// Decrement the balance for any unspent credit with less than
	// minConf confirmations and any (unspent) immature coinbase credit.
	coinBaseMaturity := int32(s.chainParams.CoinbaseMaturity)
	stopConf := minConf
	if coinBaseMaturity > stopConf {
		stopConf = coinBaseMaturity
	}
	lastOrder := syncOrder - stopConf
	blockIt := makeReadReverseBlockIterator(ns)
	for blockIt.prev() {
		block := &blockIt.elem

		if block.Order < lastOrder {
			break
		}

		for i := range block.transactions {
			txHash := &block.transactions[i]
			rec, err := fetchTxRecord(ns, txHash, &block.Block)
			if err != nil {
				return types.Amount{}, err
			}
			numOuts := uint32(len(rec.MsgTx.TxOut))
			for i := uint32(0); i < numOuts; i++ {
				// Avoid double decrementing the credit amount
				// if it was already removed for being spent by
				// an unMined tx.
				opKey := canonicalOutPoint(txHash, i)
				if existsRawUnMinedInput(ns, opKey) != nil {
					continue
				}

				_, v := existsCredit(ns, txHash, i, &block.Block)
				if v == nil {
					continue
				}
				amt, spent, err := fetchRawCreditAmountSpent(v)
				if err != nil {
					return types.Amount{}, err
				}
				if spent {
					continue
				}
				confirms := syncOrder - block.Order + 1
				if confirms < minConf || (IsCoinBaseTx(&rec.MsgTx) &&
					confirms < coinBaseMaturity) {
					bal.Value -= amt.Value
				}
			}
		}
	}
	if blockIt.err != nil {
		return types.Amount{}, blockIt.err
	}

	// If unMined outputs are included, increment the balance for each
	// output that is unspent.
	if minConf == 0 {
		err = ns.NestedReadBucket(bucketUnMinedCredits).ForEach(func(k, v []byte) error {
			if existsRawUnMinedInput(ns, k) != nil {
				// Output is spent by an unMined transaction.
				// Skip to next unMined credit.
				return nil
			}

			amount, err := fetchRawUnMinedCreditAmount(v)
			if err != nil {
				return err
			}
			bal.Value += amount.Value
			return nil
		})
		if err != nil {
			if _, ok := err.(Error); ok {
				return types.Amount{}, err
			}
			str := "failed to iterate over unMined credits bucket"
			return types.Amount{}, storeError(ErrDatabase, str, err)
		}
	}

	return bal, nil
}
