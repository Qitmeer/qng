/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import (
	"encoding/binary"
	"fmt"
	"github.com/Qitmeer/qng/common/util"
	"github.com/Qitmeer/qng/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"os"
	"path"
)

var blockNumberPrefix = []byte("M") // blockNumberPrefix + hash -> num (uint64 big endian)

// blockNumberKey = blockNumberPrefix + hash
func blockNumberKey(hash common.Hash) []byte {
	return append(blockNumberPrefix, hash.Bytes()...)
}

func ReadBlockNumber(db ethdb.KeyValueReader, hash common.Hash) *uint64 {
	data, _ := db.Get(blockNumberKey(hash))
	if len(data) != 8 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

func WriteBlockNumber(db ethdb.KeyValueWriter, hash common.Hash, number uint64) {
	key := blockNumberKey(hash)
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)

	if err := db.Put(key, enc); err != nil {
		log.Error(fmt.Sprintf("Failed to store hash to number mapping:%v", err))
	}
}

func DeleteBlockNumber(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(blockNumberKey(hash)); err != nil {
		log.Error(fmt.Sprintf("Failed to delete hash to number mapping:%v", err))
	}
}

func Cleanup(cfg *config.Config) {
	dbPath := path.Join(cfg.DataDir, ClientIdentifier)
	err := os.RemoveAll(dbPath)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info(fmt.Sprintf("Finished cleanup:%s", dbPath))
}

func Exist(cfg *config.Config) bool {
	dbPath := path.Join(cfg.DataDir, ClientIdentifier)
	return util.FileExists(dbPath)
}
