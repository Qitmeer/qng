package amana

import (
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	eparams "github.com/ethereum/go-ethereum/params"
	"math/big"
)

// deprecated
// TODO:Purely for compatibility with the testnet network, it can be completely removed if recreated in the future
var AmanaTestnetChainConfig = &eparams.ChainConfig{
	ChainID:             big.NewInt(81341),
	HomesteadBlock:      big.NewInt(0),
	DAOForkBlock:        big.NewInt(0),
	DAOForkSupport:      false,
	EIP150Block:         big.NewInt(0),
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	IstanbulBlock:       big.NewInt(0),
	MuirGlacierBlock:    big.NewInt(0),
	BerlinBlock:         big.NewInt(0),
	LondonBlock:         big.NewInt(0),
	ArrowGlacierBlock:   big.NewInt(0),
	GrayGlacierBlock:    big.NewInt(0),
	Clique: &eparams.CliqueConfig{
		Period: 3,
		Epoch:  100,
	},
}

// deprecated
func AmanaTestnetGenesis() *core.Genesis {
	return &core.Genesis{
		Config:     AmanaTestnetChainConfig,
		Nonce:      1,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x000000000000000000000000000000000000000000000000000000000000000071bc4403af41634cda7c32600a8024d54e7f64990000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   0x47b760,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(),
		Timestamp:  uint64(qparams.TestNetParam.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}
