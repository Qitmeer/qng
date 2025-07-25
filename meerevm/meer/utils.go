package meer

import (
	"fmt"
	mmeer "github.com/Qitmeer/qng/v2/consensus/model/meer"
	"github.com/Qitmeer/qng/v2/core/address"
	"github.com/Qitmeer/qng/v2/core/blockchain/utxo"
	qtypes "github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/crypto/ecc"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	qparams "github.com/Qitmeer/qng/v2/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/version"
	"math/big"
)

func makeHeader(cfg *ethconfig.Config, parent *types.Block, state *state.StateDB, timestamp int64, gaslimit uint64, difficulty *big.Int) *types.Header {
	ptt := int64(parent.Time())
	if timestamp <= ptt {
		timestamp = ptt + 1
	}

	header := &types.Header{
		Root:       state.IntermediateRoot(cfg.Genesis.Config.IsEIP158(parent.Number())),
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: difficulty,
		GasLimit:   gaslimit,
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       uint64(timestamp),
	}
	if cfg.Genesis.Config.IsLondon(header.Number) {
		header.BaseFee = eip1559.CalcBaseFee(cfg.Genesis.Config, parent.Header())
		if !cfg.Genesis.Config.IsLondon(parent.Number()) {
			parentGasLimit := parent.GasLimit() * cfg.Genesis.Config.ElasticityMultiplier()
			header.GasLimit = core.CalcGasLimit(parentGasLimit, parentGasLimit)
		}
	}
	// Apply EIP-4844, EIP-4788.
	if cfg.Genesis.Config.IsCancun(header.Number, header.Time) {
		var excessBlobGas uint64
		if cfg.Genesis.Config.IsCancun(parent.Number(), parent.Time()) {
			excessBlobGas = eip4844.CalcExcessBlobGas(cfg.Genesis.Config, parent.Header(), uint64(timestamp))
		}
		header.BlobGasUsed = new(uint64)
		header.ExcessBlobGas = &excessBlobGas
		header.ParentBeaconRoot = new(common.Hash)
	}
	return header
}

func BuildEVMBlock(block *qtypes.SerializedBlock) (*mmeer.Block, error) {
	result := &mmeer.Block{Id: block.Hash(), Txs: []mmeer.Tx{}, Time: block.Block().Header.Timestamp}

	for idx, tx := range block.Transactions() {
		if idx == 0 {
			continue
		}
		if tx.IsDuplicate {
			continue
		}

		if qtypes.IsCrossChainExportTx(tx.Tx) {
			if tx.Object != nil {
				result.Txs = append(result.Txs, tx.Object.(*mmeer.ExportTx))
				continue
			}
			ctx, err := mmeer.NewExportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if qtypes.IsCrossChainImportTx(tx.Tx) {
			ctx, err := mmeer.NewImportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			err = ctx.SetCoinbaseTx(block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if qtypes.IsCrossChainVMTx(tx.Tx) {
			if tx.Object != nil {
				vt := tx.Object.(*mmeer.VMTx)
				if vt.Coinbase.IsEqual(block.Transactions()[0].Hash()) {
					result.Txs = append(result.Txs, vt)
					continue
				}
			}
			ctx, err := mmeer.NewVMTx(tx.Tx, block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			tx.Object = ctx
			result.Txs = append(result.Txs, ctx)
		}
	}
	return result, nil
}

func CheckUTXOPubkey(pubKey ecc.PublicKey, entry *utxo.UtxoEntry) ([]byte, error) {
	if len(entry.PkScript()) <= 0 {
		return nil, fmt.Errorf("PkScript is empty")
	}

	addrUn, err := address.NewSecpPubKeyAddress(pubKey.SerializeUncompressed(), qparams.ActiveNetParams.Params)
	if err != nil {
		return nil, err
	}

	addr, err := address.NewSecpPubKeyAddress(pubKey.SerializeCompressed(), qparams.ActiveNetParams.Params)
	if err != nil {
		return nil, err
	}

	scriptClass, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(entry.PkScript(), qparams.ActiveNetParams.Params)
	if err != nil {
		return nil, err
	}
	if len(pksAddrs) != 1 {
		return nil, fmt.Errorf("PKScript num no support:%d", len(pksAddrs))
	}

	switch scriptClass {
	case txscript.PubKeyHashTy, txscript.PubkeyHashAltTy, txscript.PubKeyTy, txscript.PubkeyAltTy:
		if pksAddrs[0].Encode() == addr.PKHAddress().String() ||
			pksAddrs[0].Encode() == addrUn.PKHAddress().String() {
			return pubKey.SerializeUncompressed(), nil
		}
	default:
		return nil, fmt.Errorf("UTXO error about no support %s", scriptClass.String())
	}
	return nil, fmt.Errorf("UTXO error")
}

// Semantic holds the textual version string for major.minor.patch.
var Semantic = fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)

// WithMeta holds the textual version string including the metadata.
var WithMeta = func() string {
	v := Semantic
	if version.Meta != "" {
		v += "-" + version.Meta
	}
	return v
}()
