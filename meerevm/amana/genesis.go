package amana

import (
	mparams "github.com/Qitmeer/qng/meerevm/params"
	qparams "github.com/Qitmeer/qng/params"
	qcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

func AmanaGenesis() *core.Genesis {
	return &core.Genesis{
		Config:     mparams.AmanaChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x000000000000000000000000000000000000000000000000000000000000000071bc4403af41634cda7c32600a8024d54e7f64990000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   0x47b760,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(),
		Timestamp:  uint64(qparams.MainNetParam.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func decodePrealloc() types.GenesisAlloc {
	ga := types.GenesisAlloc{}
	ga[qcommon.HexToAddress("0x71bc4403af41634cda7c32600a8024d54e7f6499")] = core.GenesisAccount{Balance: big.NewInt(params.Ether).Mul(big.NewInt(params.Ether), big.NewInt(10000000000))}
	return ga
}
