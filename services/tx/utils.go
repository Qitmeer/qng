package tx

import (
	"encoding/binary"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func CalcUTXOHash(txid *hash.Hash, idx uint32) common.Hash {
	data := txid.CloneBytes()

	var sidx [4]byte
	binary.BigEndian.PutUint32(sidx[:], idx)
	data = append(data, sidx[:]...)

	return common.BytesToHash(accounts.TextHash(data))
}

func CalcUTXOSig(hash common.Hash, privKeyHex string) ([]byte, error) {
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(hash.Bytes(), privateKey)
}
