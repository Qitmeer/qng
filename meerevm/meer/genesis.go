package meer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	mparams "github.com/Qitmeer/qng/meerevm/params"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	qcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func QngGenesis(alloc types.GenesisAlloc) *core.Genesis {
	if alloc == nil {
		alloc = DecodeAlloc(qparams.MainNetParam.Params)
	}
	return &core.Genesis{
		Config:     mparams.QngMainnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      alloc,
		Timestamp:  uint64(qparams.MainNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func QngTestnetGenesis(alloc types.GenesisAlloc) *core.Genesis {
	if alloc == nil {
		alloc = DecodeAlloc(qparams.TestNetParam.Params)
	}
	return &core.Genesis{
		Config:     mparams.QngTestnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   8000000,
		Difficulty: big.NewInt(0),
		Alloc:      alloc,
		Timestamp:  uint64(qparams.TestNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func QngMixnetGenesis(alloc types.GenesisAlloc) *core.Genesis {
	if alloc == nil {
		alloc = DecodeAlloc(qparams.MixNetParam.Params)
	}
	return &core.Genesis{
		Config:     mparams.QngMixnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      alloc,
		Timestamp:  uint64(qparams.MixNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func QngPrivnetGenesis(alloc types.GenesisAlloc) *core.Genesis {
	if alloc == nil {
		alloc = DecodeAlloc(qparams.PrivNetParam.Params)
	}
	return &core.Genesis{
		Config:     mparams.QngPrivnetChainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      alloc,
		Timestamp:  uint64(qparams.PrivNetParams.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
}

func DecodePrealloc(data string) types.GenesisAlloc {
	if len(data) <= 0 {
		return types.GenesisAlloc{}
	}
	var p []struct {
		Addr    *big.Int
		Balance *big.Int
		Misc    *struct {
			Nonce uint64
			Code  []byte
			Slots []struct {
				Key qcommon.Hash
				Val qcommon.Hash
			}
		} `rlp:"optional"`
	}
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(types.GenesisAlloc, len(p))
	for _, account := range p {
		acc := types.Account{Balance: account.Balance}
		if account.Misc != nil {
			acc.Nonce = account.Misc.Nonce
			acc.Code = account.Misc.Code

			acc.Storage = make(map[qcommon.Hash]qcommon.Hash)
			for _, slot := range account.Misc.Slots {
				acc.Storage[slot.Key] = slot.Val
			}
		}
		ga[qcommon.BigToAddress(account.Addr)] = acc
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

func DecodeAlloc(network *qparams.Params) types.GenesisAlloc {
	return DoDecodeAlloc(network, "", "")
}

func DoDecodeAlloc(network *qparams.Params, genesisStr string, burnStr string) types.GenesisAlloc {
	if len(genesisStr) <= 0 {
		genesisStr = genesisJson
	}
	if len(burnStr) <= 0 {
		burnStr = burnListJson
	}
	gds := []NetGenesisData{}
	jsonR := strings.NewReader(genesisStr)
	if err := json.NewDecoder(jsonR).Decode(&gds); err != nil {
		panic(err)
	}
	if len(gds) != 4 {
		panic(fmt.Errorf("Error genesis data config"))
	}
	var ngd *NetGenesisData
	for _, gd := range gds {
		if gd.Network == network.Name {
			tgd := gd
			ngd = &tgd
			break
		}
	}
	if ngd == nil {
		panic(fmt.Errorf("No alloc config data from: %s", network.Name))
	}

	burnList := BuildBurnBalance(burnStr)
	genesis := Genesis(network.Net, types.GenesisAlloc{})
	genesis.Alloc = ngd.Data.Genesis.Alloc
	releaseConAddr := common.HexToAddress(RELEASE_CONTRACT_ADDR)
	if releaseAccount, ok := genesis.Alloc[releaseConAddr]; ok {
		releaseAccount.Storage = burnList
		genesis.Alloc[releaseConAddr] = releaseAccount
	}
	if len(ngd.Data.Contracts) > 0 {
		err := UpdateAlloc(genesis, ngd.Data.Contracts)
		if err != nil {
			panic(err)
		}
	}
	return genesis.Alloc
}
