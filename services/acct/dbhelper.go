package acct

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/params"
	"os"
	"path/filepath"
)

const (
	dbNamePrefix = "accts"
)

func loadDB(DbType string, DataDir string, nocreate bool) (legacydb.DB, error) {
	dbPath := getDBPath(DataDir)
	log.Trace(fmt.Sprintf("Loading acct database:%s", dbPath))
	db, err := legacydb.Open(DbType, dbPath, params.ActiveNetParams.Net)
	if err != nil {
		if nocreate {
			// Return the error if it's not because the database doesn't
			// exist.
			if dbErr, ok := err.(legacydb.Error); !ok || dbErr.ErrorCode !=
				legacydb.ErrDbDoesNotExist {

				return nil, err
			}
			// Create the db if it does not exist.
			err = os.MkdirAll(DataDir, 0700)
			if err != nil {
				return nil, err
			}
			db, err = legacydb.Create(DbType, dbPath, params.ActiveNetParams.Net)
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

func getDataDir(cfg *config.Config) string {
	dataDir := cfg.DataDir
	if len(dataDir) <= 0 {
		dataDir = cfg.HomeDir
	}
	return dataDir
}

// info
func DBGetACCTInfo(dbTx legacydb.Tx) (*AcctInfo, error) {
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

func DBPutACCTInfo(dbTx legacydb.Tx, ai *AcctInfo) error {
	var buff bytes.Buffer
	err := ai.Encode(&buff)
	if err != nil {
		return err
	}
	return dbTx.Metadata().Put(InfoBucketName, buff.Bytes())
}

// balance
func DBGetACCTBalance(dbTx legacydb.Tx, address string) (*AcctBalance, error) {
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

func DBPutACCTBalance(dbTx legacydb.Tx, address string, ab *AcctBalance) error {
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

func DBDelACCTBalance(dbTx legacydb.Tx, address string) error {
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

func DBPutACCTUTXO(dbTx legacydb.Tx, address string, op *types.TxOutPoint, au *AcctUTXO) error {
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

func DBDelACCTUTXO(dbTx legacydb.Tx, address string, op *types.TxOutPoint) error {
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

func DBDelACCTUTXOs(dbTx legacydb.Tx, address string) error {
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

func DBGetACCTUTXOs(dbTx legacydb.Tx, address string) map[string]*AcctUTXO {
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
	result := map[string]*AcctUTXO{}
	err := balUTXOBucket.ForEach(func(ku, vu []byte) error {
		au := NewAcctUTXO()
		err := au.Decode(bytes.NewReader(vu))
		if err != nil {
			return err
		}
		kus := hex.EncodeToString(ku)
		if result[kus] != nil {
			log.Error("Already exists:Outpoint=%s", kus)
		}
		result[kus] = au
		return nil
	})
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return result
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
	ret, err := common.ParseOutpoint(opk)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	return ret, nil
}
