package scriptbasetypes

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

type TxSignBase interface {
	Sign(privateKey string, mtx *types.Transaction, inputIndex int, param *params.Params) error
}

func NewTxSignObject(scripttype txscript.ScriptClass) TxSignBase {
	var s TxSignBase
	switch scripttype {
	case txscript.CLTVPubKeyHashTy:
		s = &CLTVPubKeyHashScript{}
	case txscript.PubKeyTy:
		s = &PubKeyScript{}
	case 255:
		s = &CrossImportScript{}
	default:
		// pubkeyhash
		s = &ScriptTypeRegular{}
	}
	return s
}

func GetScriptType(scriptTyp string) txscript.ScriptClass {
	switch scriptTyp {
	case "standard":
		return txscript.PubKeyHashTy
	case "pubkey":
		return txscript.PubKeyTy
	case "cltvpubkeyhash":
		return txscript.CLTVPubKeyHashTy
	case "crossimport":
		return 255 // special script
	default:
		return txscript.NonStandardTy
	}
}
