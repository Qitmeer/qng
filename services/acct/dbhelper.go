package acct

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
)

// info
func DBPutACCTInfo(dbTx database.Tx, ai *AcctInfo) error {
	var buff bytes.Buffer
	err := ai.Encode(&buff)
	if err != nil {
		return err
	}
	return dbTx.Metadata().Put(InfoBucketName, buff.Bytes())
}

// balance
func DBPutACCTBalance(dbTx database.Tx, address string, ab *AcctBalance) error {
	var buff bytes.Buffer
	err := ab.Encode(&buff)
	if err != nil {
		return err
	}
	return dbTx.Metadata().Put([]byte(address), buff.Bytes())
}

// utxo
func GetACCTUTXOKey(address string) []byte {
	return []byte(fmt.Sprintf("%s%s", address, AddressUTXOsSuffix))
}

func DBPutACCTUTXO(dbTx database.Tx, op *types.TxOutPoint, au *AcctUTXO) error {
	var buff bytes.Buffer
	err := au.Encode(&buff)
	if err != nil {
		return err
	}
	key := blockchain.OutpointKey(*op)
	return dbTx.Metadata().Put(*key, buff.Bytes())
}
