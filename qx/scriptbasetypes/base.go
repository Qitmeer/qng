package scriptbasetypes

import (
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	"github.com/Qitmeer/qng/v2/params"
)

// special script
// from evm to meer
const SPECIAL_CROSS_TYPE = "crossimport"
const SPECIAL_CROSS_VAL txscript.ScriptClass = 255

type TxSignBase interface {
	Sign(privateKey string, mtx *types.Transaction, inputIndex int, param *params.Params) error
}

func NewTxSignObject(scripttype txscript.ScriptClass, lockTime int64) TxSignBase {
	var s TxSignBase
	switch scripttype {
	case txscript.CLTVPubKeyHashTy:
		s = &CLTVPubKeyHashScript{
			LockTime: lockTime,
		}
	case txscript.PubKeyTy:
		s = &PubKeyScript{}
	case SPECIAL_CROSS_VAL:
		s = &CrossImportScript{}
	default:
		// pubkeyhash
		s = &ScriptTypeRegular{}
	}
	return s
}

func GetScriptType(scriptTyp string) txscript.ScriptClass {
	switch scriptTyp {
	case txscript.PubKeyHashTy.String():
		return txscript.PubKeyHashTy
	case txscript.PubKeyTy.String():
		return txscript.PubKeyTy
	case txscript.CLTVPubKeyHashTy.String():
		return txscript.CLTVPubKeyHashTy
	case SPECIAL_CROSS_TYPE:
		return SPECIAL_CROSS_VAL // special script
	default:
		return txscript.NonStandardTy
	}
}
