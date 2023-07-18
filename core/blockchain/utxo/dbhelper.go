package utxo

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
)

const UtxoEntryAmountCoinIDSize = 2

func DBFetchUtxoEntry(db model.DataBase, outpoint types.TxOutPoint) (*UtxoEntry, error) {
	return dbFetchUtxoEntry(db, outpoint)
}

// dbFetchUtxoEntry uses an existing database transaction to fetch all unspent
// outputs for the provided Bitcoin transaction hash from the utxo set.
//
// When there is no entry for the provided hash, nil will be returned for the
// both the entry and the error.
func dbFetchUtxoEntry(db model.DataBase, outpoint types.TxOutPoint) (*UtxoEntry, error) {
	// Fetch the unspent transaction output information for the passed
	// transaction output.  Return now when there is no entry.
	key := OutpointKey(outpoint)
	serializedUtxo, err := db.GetUtxo(*key)
	if err != nil {
		return nil, err
	}
	RecycleOutpointKey(key)
	if serializedUtxo == nil {
		return nil, nil
	}

	// A non-nil zero-length entry means there is an entry in the database
	// for a spent transaction output which should never be the case.
	if len(serializedUtxo) == 0 {
		return nil, fmt.Errorf(fmt.Sprintf("database contains entry "+
			"for spent tx output %v", outpoint))
	}

	// Deserialize the utxo entry and return it.
	entry, err := DeserializeUtxoEntry(serializedUtxo)
	if err != nil {
		// Ensure any deserialization errors are returned as database
		// corruption errors.
		if model.IsDeserializeErr(err) {
			return nil, legacydb.Error{
				ErrorCode: legacydb.ErrCorruption,
				Description: fmt.Sprintf("corrupt utxo entry "+
					"for %v: %v", outpoint, err),
			}
		}

		return nil, err
	}

	return entry, nil
}

// deserializeUtxoEntry decodes a utxo entry from the passed serialized byte
// slice into a new UtxoEntry using a format that is suitable for long-term
// storage.  The format is described in detail above.
func DeserializeUtxoEntry(serialized []byte) (*UtxoEntry, error) {
	// Deserialize the header code.
	code, offset := serialization.DeserializeVLQ(serialized)
	if offset >= len(serialized) {
		return nil, model.ErrDeserialize("unexpected end of data after header")
	}

	// Decode the header code.
	//
	// Bit 0 indicates whether the containing transaction is a coinbase.
	// Bits 1-x encode id of containing transaction.
	isCoinBase := code&0x01 != 0

	blockHash, err := hash.NewHash(serialized[offset : offset+hash.HashSize])
	if err != nil {
		return nil, model.ErrDeserialize(fmt.Sprintf("unable to decode "+
			"utxo: %v", err))
	}
	offset += hash.HashSize
	// Decode amount coinId
	// Decode amount coinId
	amountCoinId := byteOrder.Uint16(serialized[offset : offset+SpentTxoutAmountCoinIDSize])
	offset += SpentTxoutAmountCoinIDSize
	// Decode the compressed unspent transaction output.
	amount, pkScript, _, err := decodeCompressedTxOut(serialized[offset:])
	if err != nil {
		return nil, model.ErrDeserialize(fmt.Sprintf("unable to decode "+
			"utxo: %v", err))
	}
	return NewUtxoEntry(types.Amount{Value: int64(amount), Id: types.CoinID(amountCoinId)}, pkScript, blockHash, isCoinBase), nil
}

func SerializeUtxoEntry(entry *UtxoEntry) ([]byte, error) {
	// Spent outputs have no serialization.
	if entry.IsSpent() {
		return nil, nil
	}

	// Encode the header code.
	headerCode, err := utxoEntryHeaderCode(entry)
	if err != nil {
		return nil, err
	}

	// Calculate the size needed to serialize the entry.
	size := serialization.SerializeSizeVLQ(headerCode) + hash.HashSize + UtxoEntryAmountCoinIDSize +
		compressedTxOutSize(uint64(entry.Amount().Value), entry.PkScript())

	// Serialize the header code followed by the compressed unspent
	// transaction output.
	serialized := make([]byte, size)
	offset := serialization.PutVLQ(serialized, headerCode)
	copy(serialized[offset:offset+hash.HashSize], entry.BlockHash().Bytes())
	offset += hash.HashSize
	// add Amount coinId
	byteOrder.PutUint16(serialized[offset:], uint16(entry.Amount().Id))
	offset += SpentTxoutAmountCoinIDSize
	putCompressedTxOut(serialized[offset:], uint64(entry.Amount().Value),
		entry.PkScript())

	return serialized, nil
}

func utxoEntryHeaderCode(entry *UtxoEntry) (uint64, error) {
	if entry.IsSpent() {
		return 0, model.AssertError("attempt to serialize spent utxo header")
	}

	// As described in the serialization format comments, the header code
	// encodes the height shifted over one bit and the coinbase flag in the
	// lowest bit.
	headerCode := uint64(0)
	if entry.IsCoinBase() {
		headerCode |= 0x01
	}

	return headerCode, nil
}
