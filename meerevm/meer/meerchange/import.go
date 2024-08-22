package meerchange

import (
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	qtypes "github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"strings"
)

var (
	LogImportSigHash = crypto.Keccak256Hash([]byte("Import()"))
)

type MeerchangeImportData struct {
	OutPoint *qtypes.TxOutPoint
	Output   *qtypes.TxOutput
}

func (e *MeerchangeImportData) GetFuncName() string {
	return "importToUtxo"
}

func (e *MeerchangeImportData) GetLogName() string {
	return "Import"
}

func NewMeerchangeImportData(qtx *qtypes.Transaction, tx *types.Transaction) (*MeerchangeImportData, error) {
	txh := qtx.TxHash()
	amount := tx.Value().Div(tx.Value(), common.Precision)
	if amount.Uint64() <= 0 {
		return nil, fmt.Errorf("import amount empty:%s", tx.Value().String())
	}
	signer := types.NewPKSigner(GetChainID())
	pkb, err := signer.GetPublicKey(tx)
	if err != nil {
		return nil, err
	}
	pubKey, err := ecc.Secp256k1.ParsePubKey(pkb)
	if err != nil {
		return nil, err
	}
	addrUn, err := address.NewSecpPubKeyAddress(pubKey.SerializeUncompressed(), params.ActiveNetParams.Params)
	if err != nil {
		return nil, err
	}
	pkScript, err := txscript.PayToAddrScript(addrUn)
	if err != nil {
		return nil, err
	}

	return &MeerchangeImportData{
		OutPoint: qtypes.NewOutPoint(&txh, 0),
		Output:   qtypes.NewTxOutput(qtypes.Amount{Value: amount.Int64(), Id: qtypes.MEERA}, pkScript),
	}, nil
}

func IsMeerChangeImportTx(tx *types.Transaction) bool {
	if !IsMeerChangeTx(tx) {
		return false
	}
	if len(tx.Data()) < 4 {
		return false
	}
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return false
	}
	method, err := contractAbi.MethodById(tx.Data()[:4])
	if err != nil {
		return false
	}
	if method.Name != (&MeerchangeImportData{}).GetFuncName() {
		return false
	}
	return true
}
