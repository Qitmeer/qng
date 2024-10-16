package meer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	qcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

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
	genesis := Genesis(network, types.GenesisAlloc{})
	genesis.Alloc = ngd.Data.Genesis.Alloc
	releaseConAddr := common.HexToAddress(RELEASE_CONTRACT_ADDR)
	if releaseAccount, ok := genesis.Alloc[releaseConAddr]; ok {
		for k, v := range burnList {
			kk := k
			vv := v
			releaseAccount.Storage[kk] = vv
		}
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
