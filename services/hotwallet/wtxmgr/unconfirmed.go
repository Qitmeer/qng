package wtxmgr

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
)

// insertMemPoolTx inserts the unMined transaction record.  It also marks
// previous outputs referenced by the inputs as spent.
func (s *Store) insertMemPoolTx(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
	// Check whether the transaction has already been added to the
	// unconfirmed bucket.
	if existsRawUnMined(ns, rec.Hash[:]) != nil {
		// TODO: compare serialized txs to ensure this isn't a hash
		// collision?
		return nil
	}

	// Since transaction records within the store are keyed by their
	// transaction _and_ block confirmation, we'll iterate through the
	// transaction's outputs to determine if we've already seen them to
	// prevent from adding this transaction to the unconfirmed bucket.
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		if existsRawUnspent(ns, k) != nil {
			return nil
		}
	}

	log.Trace("Inserting unconfirmed transaction ", "rec.Hash", rec.Hash)
	v, err := valueTxRecord(rec)
	if err != nil {
		return err
	}
	err = putRawUnMined(ns, rec.Hash[:], v)
	if err != nil {
		return err
	}

	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOut
		k := canonicalOutPoint(&prevOut.Hash, prevOut.OutIndex)
		err = putRawUnMinedInput(ns, k, rec.Hash[:])
		if err != nil {
			return err
		}
	}

	return nil
}

// removeDoubleSpends checks for any unMined transactions which would introduce
// a double spend if tx was added to the store (either as a confirmed or unMined
// transaction).  Each conflicting transaction and all transactions which spend
// it are recursively removed.
func (s *Store) removeDoubleSpends(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOut
		prevOutKey := canonicalOutPoint(&prevOut.Hash, prevOut.OutIndex)

		doubleSpendHashes := fetchUnMinedInputSpendTxHashes(ns, prevOutKey)
		for _, doubleSpendHash := range doubleSpendHashes {
			doubleSpendVal := existsRawUnMined(ns, doubleSpendHash[:])

			// If the spending transaction spends multiple outputs
			// from the same transaction, we'll find duplicate
			// entries within the store, so it's possible we're
			// unable to find it if the conflicts have already been
			// removed in a previous iteration.
			if doubleSpendVal == nil {
				continue
			}

			var doubleSpend TxRecord
			doubleSpend.Hash = doubleSpendHash
			err := readRawTxRecord(
				&doubleSpend.Hash, doubleSpendVal, &doubleSpend,
			)
			if err != nil {
				return err
			}

			log.Debug("Removing double spending transaction %v",
				doubleSpend.Hash)
			if err := s.removeConflict(ns, &doubleSpend); err != nil {
				return err
			}
		}
	}

	return nil
}

// removeConflict removes an unMined transaction record and all spend chains
// deriving from it from the store.  This is designed to remove transactions
// that would otherwise result in double spend conflicts if left in the store,
// and to remove transactions that spend coinbase transactions on reorgs.
func (s *Store) removeConflict(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
	// For each potential credit for this record, each spender (if any) must
	// be recursively removed as well.  Once the spenders are removed, the
	// credit is deleted.
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		spenderHashes := fetchUnMinedInputSpendTxHashes(ns, k)
		for _, spenderHash := range spenderHashes {
			spenderVal := existsRawUnMined(ns, spenderHash[:])

			// If the spending transaction spends multiple outputs
			// from the same transaction, we'll find duplicate
			// entries within the store, so it's possible we're
			// unable to find it if the conflicts have already been
			// removed in a previous iteration.
			if spenderVal == nil {
				continue
			}

			var spender TxRecord
			spender.Hash = spenderHash
			err := readRawTxRecord(&spender.Hash, spenderVal, &spender)
			if err != nil {
				return err
			}

			log.Debug("Transaction %v is part of a removed conflict "+
				"chain -- removing as well", spender.Hash)
			if err := s.removeConflict(ns, &spender); err != nil {
				return err
			}
		}
		if err := deleteRawUnMinedCredit(ns, k); err != nil {
			return err
		}
	}

	// If this tx spends any previous credits (either mined or unMined), set
	// each unspent.  Mined transactions are only marked spent by having the
	// output in the unMined inputs bucket.
	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOut
		k := canonicalOutPoint(&prevOut.Hash, prevOut.OutIndex)
		if err := deleteRawUnMinedInput(ns, k); err != nil {
			return err
		}
	}

	return deleteRawUnmMined(ns, rec.Hash[:])
}

// UnMinedTxs returns the underlying transactions for all unMined transactions
// which are not known to have been mined in a block.  Transactions are
// guaranteed to be sorted by their dependency order.
func (s *Store) UnMinedTxs(ns walletdb.ReadBucket) ([]*types.Transaction, error) {
	recSet, err := s.unMinedTxRecords(ns)
	if err != nil {
		return nil, err
	}

	txSet := make(map[hash.Hash]*types.Transaction, len(recSet))
	for txHash, txRec := range recSet {
		txSet[txHash] = &txRec.MsgTx
	}

	return DependencySort(txSet), nil
}

func (s *Store) unMinedTxRecords(ns walletdb.ReadBucket) (map[hash.Hash]*TxRecord, error) {
	unMined := make(map[hash.Hash]*TxRecord)
	err := ns.NestedReadBucket(bucketUnMined).ForEach(func(k, v []byte) error {
		var txHash hash.Hash
		err := readRawUnMinedHash(k, &txHash)
		if err != nil {
			return err
		}

		rec := new(TxRecord)
		err = readRawTxRecord(&txHash, v, rec)
		if err != nil {
			return err
		}
		unMined[rec.Hash] = rec
		return nil
	})
	return unMined, err
}

// UnMinedTxHashes returns the hashes of all transactions not known to have been
// mined in a block.
func (s *Store) UnMinedTxHashes(ns walletdb.ReadBucket) ([]*hash.Hash, error) {
	return s.unMinedTxHashes(ns)
}

func (s *Store) unMinedTxHashes(ns walletdb.ReadBucket) ([]*hash.Hash, error) {
	var hashes []*hash.Hash
	err := ns.NestedReadBucket(bucketUnMined).ForEach(func(k, v []byte) error {
		hash := new(hash.Hash)
		err := readRawUnMinedHash(k, hash)
		if err == nil {
			hashes = append(hashes, hash)
		}
		return err
	})
	return hashes, err
}
