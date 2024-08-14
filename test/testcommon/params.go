package testcommon

import (
	"math/big"

	"github.com/Qitmeer/qng/common"
	"github.com/Qitmeer/qng/meerevm/params"
)

var (
	CHAIN_ID = params.QngPrivnetChainConfig.ChainID

	MAX_UINT256 = new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 255), common.Big1)
)
