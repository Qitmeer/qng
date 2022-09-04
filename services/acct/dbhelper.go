package acct

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/params"
	"os"
	"path/filepath"
)

const (
	dbNamePrefix = "accts"
)

func loadDB(DbType string, DataDir string, nocreate bool) (database.DB, error) {
	dbPath := getDBPath(DataDir)
	log.Trace(fmt.Sprintf("Loading acct database:%s", dbPath))
	db, err := database.Open(DbType, dbPath, params.ActiveNetParams.Net)
	if err != nil {
		if nocreate {
			// Return the error if it's not because the database doesn't
			// exist.
			if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode !=
				database.ErrDbDoesNotExist {

				return nil, err
			}
			// Create the db if it does not exist.
			err = os.MkdirAll(DataDir, 0700)
			if err != nil {
				return nil, err
			}
			db, err = database.Create(DbType, dbPath, params.ActiveNetParams.Net)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	log.Trace("Acct database loaded")
	return db, nil
}

func removeDB(dbPath string) error {
	fi, err := os.Stat(dbPath)
	if err == nil {
		log.Info(fmt.Sprintf("Removing acct database from '%s'", dbPath))
		if fi.IsDir() {
			err := os.RemoveAll(dbPath)
			if err != nil {
				return err
			}
		} else {
			err := os.Remove(dbPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getDBPath(dataDir string) string {
	return filepath.Join(dataDir, dbNamePrefix)
}

// info
func DBGetACCTInfo(dbTx database.Tx) (*AcctInfo, error) {
	meta := dbTx.Metadata()

	infoData := meta.Get(InfoBucketName)
	if infoData == nil {
		return nil, nil
	}
	info := NewAcctInfo()
	err := info.Decode(bytes.NewReader(infoData))
	if err != nil {
		return nil, err
	}
	return info, nil
}

func DBPutACCTInfo(dbTx database.Tx, ai *AcctInfo) error {
	var buff bytes.Buffer
	err := ai.Encode(&buff)
	if err != nil {
		return err
	}
	return dbTx.Metadata().Put(InfoBucketName, buff.Bytes())
}

// balance
func DBGetACCTBalance(dbTx database.Tx, address string) (*AcctBalance, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(BalanceBucketName)
	if bucket == nil {
		return nil, nil
	}
	balData := bucket.Get([]byte(address))
	if balData == nil {
		return nil, nil
	}
	balance := &AcctBalance{}
	err := balance.Decode(bytes.NewReader(balData))
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func DBPutACCTBalance(dbTx database.Tx, address string, ab *AcctBalance) error {
	var buff bytes.Buffer
	err := ab.Encode(&buff)
	if err != nil {
		return err
	}
	meta := dbTx.Metadata()
	bucket, err := meta.CreateBucketIfNotExists(BalanceBucketName)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(address), buff.Bytes())
}

func DBDelACCTBalance(dbTx database.Tx, address string) error {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(BalanceBucketName)
	if bucket == nil {
		return nil
	}
	return bucket.Delete([]byte(address))
}

// utxo
func GetACCTUTXOKey(address string) []byte {
	return []byte(fmt.Sprintf("%s%s", address, AddressUTXOsSuffix))
}

func DBPutACCTUTXO(dbTx database.Tx, address string, op *types.TxOutPoint, au *AcctUTXO) error {
	var buff bytes.Buffer
	err := au.Encode(&buff)
	if err != nil {
		return err
	}
	meta := dbTx.Metadata()
	bucket, err := meta.CreateBucketIfNotExists(BalanceBucketName)
	if err != nil {
		return err
	}
	balUTXOBucket, err := bucket.CreateBucketIfNotExists(GetACCTUTXOKey(address))
	if err != nil {
		return err
	}

	key := OutpointKey(op)
	return balUTXOBucket.Put(key, buff.Bytes())
}

func DBDelACCTUTXO(dbTx database.Tx, address string, op *types.TxOutPoint) error {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(BalanceBucketName)
	if bucket == nil {
		return nil
	}
	bkey := GetACCTUTXOKey(address)
	balUTXOBucket := bucket.Bucket(bkey)
	if balUTXOBucket == nil {
		return nil
	}

	key := OutpointKey(op)
	return balUTXOBucket.Delete(key)
}

func DBDelACCTUTXOs(dbTx database.Tx, address string) error {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(BalanceBucketName)
	if bucket == nil {
		return nil
	}
	bkey := GetACCTUTXOKey(address)
	if bucket.Bucket(bkey) == nil {
		return nil
	}
	return bucket.DeleteBucket(bkey)
}

func OutpointKey(outpoint *types.TxOutPoint) []byte {
	idx := uint64(outpoint.OutIndex)
	size := hash.HashSize + serialization.SerializeSizeVLQ(idx)
	key := make([]byte, size)
	copy(key, outpoint.Hash[:])
	serialization.PutVLQ(key[hash.HashSize:], idx)
	return key
}

func parseOutpoint(opk []byte) (*types.TxOutPoint, error) {
	txhash, err := hash.NewHash(opk[:hash.HashSize])
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	txOutIdex, size := serialization.DeserializeVLQ(opk[hash.HashSize:])
	if size <= 0 {
		err := fmt.Errorf("DeserializeVLQ:%s %v", txhash.String(), opk[hash.HashSize:])
		log.Error(err.Error())
		return nil, err
	}
	return types.NewOutPoint(txhash, uint32(txOutIdex)), nil
}
