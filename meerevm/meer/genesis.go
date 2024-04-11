package meer

import (
	"fmt"
	mparams "github.com/Qitmeer/qng/meerevm/params"
	qparams "github.com/Qitmeer/qng/params"
	qcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"strings"
)

func QngGenesis() *core.Genesis {
	return &core.Genesis{
		Config:     mparams.QngMainnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      DecodePrealloc(getAllocData(qparams.MainNetParams.Name)),
		Timestamp:  uint64(qparams.MainNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func QngTestnetGenesis() *core.Genesis {
	return &core.Genesis{
		Config:     mparams.QngTestnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   8000000,
		Difficulty: big.NewInt(0),
		Alloc:      DecodePrealloc(getAllocData(qparams.TestNetParams.Name)),
		Timestamp:  uint64(qparams.TestNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func QngMixnetGenesis() *core.Genesis {
	return &core.Genesis{
		Config:     mparams.QngMixnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      DecodePrealloc(getAllocData(qparams.MixNetParams.Name)),
		Timestamp:  uint64(qparams.MixNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func QngPrivnetGenesis() *core.Genesis {
	return &core.Genesis{
		Config:     mparams.QngPrivnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      DecodePrealloc(getAllocData(qparams.PrivNetParams.Name)),
		Timestamp:  uint64(qparams.PrivNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func DecodePrealloc(data string) core.GenesisAlloc {
	if len(data) <= 0 {
		return core.GenesisAlloc{}
	}
	var p []struct {
		Addr, Balance *big.Int
		Code          []byte
		Nonce         uint64
		StorageKey    []string
		StorageValue  []string
	}
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(core.GenesisAlloc, len(p))
	for _, account := range p {
		if len(account.StorageKey) != len(account.StorageValue) {
			log.Error(fmt.Sprintf("account.StorageKey != account.StorageValue"))
			continue
		}
		storage := map[qcommon.Hash]qcommon.Hash{}
		for i := 0; i < len(account.StorageKey); i++ {
			storage[qcommon.HexToHash(account.StorageKey[i])] = qcommon.HexToHash(account.StorageValue[i])
		}
		ga[qcommon.BigToAddress(account.Addr)] = core.GenesisAccount{
			Balance: account.Balance,
			Code:    account.Code,
			Storage: storage,
			Nonce:   account.Nonce,
		}
	}
	return ga
}

type GenesisData struct {
	Genesis   core.Genesis `json:"genesis"`
	Contracts []Contract   `json:"contracts"`
}

type NetGenesisData struct {
	Network string      `json:"network"`
	Data    GenesisData `json:"data"`
}

type Contract struct {
	ABI   string `json:"abi"`
	BIN   string `json:"bin"`
	Input string `json:"input"`
}

func getAllocData(network string) string {
	if network == qparams.TestNetParam.Name {
		return testAllocData
	} else if network == qparams.PrivNetParam.Name {
		return privAllocData
	} else if network == qparams.MixNetParam.Name {
		return mixAllocData
	}
	return mainAllocData
}
