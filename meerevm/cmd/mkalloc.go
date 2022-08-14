//go:build none
// +build none

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/meerevm/chain"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math/big"
	"os"
	"sort"
	"strconv"

	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

const RELEASE_ADDR = "0x1000000000000000000000000000000000000000"

type allocItem struct {
	Addr         *big.Int
	Balance      *big.Int
	Code         []byte
	Nonce        uint64
	StorageKey   []string
	StorageValue []string
}

type allocList []allocItem

func (a allocList) Len() int           { return len(a) }
func (a allocList) Less(i, j int) bool { return a[i].Addr.Cmp(a[j].Addr) < 0 }
func (a allocList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func makelist(g *core.Genesis) allocList {
	a := make(allocList, 0, len(g.Alloc))
	for addr, account := range g.Alloc {
		bigAddr := new(big.Int).SetBytes(addr.Bytes())

		sks := []string{}
		svs := []string{}
		for k, v := range account.Storage {
			sks = append(sks, k.String())
			svs = append(svs, v.String())
		}
		a = append(a, allocItem{Addr: bigAddr, Balance: account.Balance, Code: account.Code, Nonce: account.Nonce, StorageKey: sks, StorageValue: svs})
	}
	sort.Sort(a)
	return a
}

func makealloc(g *core.Genesis) string {
	a := makelist(g)
	data, err := rlp.EncodeToBytes(a)
	if err != nil {
		panic(err)
	}
	return strconv.QuoteToASCII(string(data))
}

func main() {
	filePath := "./../chain/genesis.json"
	privateKeyHex := ""
	if len(os.Args) >= 2 {
		privateKeyHex = os.Args[1]
	}

	gds := []chain.NetGenesisData{}
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(file).Decode(&gds); err != nil {
		panic(err)
	}

	if len(gds) != 4 {
		panic(fmt.Errorf("Error genesis data config"))
	}
	fileContent := "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run mkalloc.go [privateKey] \npackage chain\n\n"
	burnList := BuildBurnBalance()
	for _, ngd := range gds {
		networkTag := ""
		if ngd.Network == params.TestNetParam.Name {
			params.ActiveNetParams = &params.TestNetParam
			networkTag = "testAllocData"
		} else if ngd.Network == params.PrivNetParam.Name {
			params.ActiveNetParams = &params.PrivNetParam
			networkTag = "privAllocData"
		} else if ngd.Network == params.MixNetParam.Name {
			params.ActiveNetParams = &params.MixNetParam
			networkTag = "mixAllocData"
		} else {
			params.ActiveNetParams = &params.MainNetParam
			networkTag = "mainAllocData"
		}

		chain.ChainConfig.ChainID = big.NewInt(params.ActiveNetParams.MeerEVMCfg.ChainID)
		genesis := chain.DefaultGenesisBlock(chain.ChainConfig)
		genesis.Alloc = ngd.Data.Genesis.Alloc
		if _, ok := genesis.Alloc[common.HexToAddress(RELEASE_ADDR)]; ok {
			releaseAccount := genesis.Alloc[common.HexToAddress(RELEASE_ADDR)]
			storage := releaseAccount.Storage
			for k, v := range burnList {
				storage[k] = v
			}
			releaseAccount.Storage = storage
			genesis.Alloc[common.HexToAddress(RELEASE_ADDR)] = releaseAccount
		}
		if len(ngd.Data.Contracts) > 0 {
			if len(privateKeyHex) <= 0 {
				panic("You must enter a private key")
			}
			err = chain.UpdateAlloc(genesis, ngd.Data.Contracts, privateKeyHex)
			if err != nil {
				panic(err)
			}
		}
		fileContent += fmt.Sprintf("\nconst %s = %s", networkTag, makealloc(genesis))
	}

	fileName := "./../chain/genesis_alloc.go"

	f, err := os.Create(fileName)

	if err != nil {
		panic(fmt.Sprintf("Save error:%s  %s", fileName, err))
	}
	defer func() {
		err = f.Close()
	}()

	f.WriteString(fileContent)

	fmt.Println("Successfully updated:", fileName)
}

type BurnDetail struct {
	Order  int64  `json:"order"`
	Height int64  `json:"height"`
	From   string `json:"from"`
	Amount int64  `json:"amount"`
	Time   string `json:"time"`
}

// 2022/08/14 17:43:57 MmQitmeerMainNetGuardAddressXd7b76q burn amount 641194999865334
// 2022/08/14 17:43:57 MmQitmeerMainNetHonorAddressXY9JH2y burn amount 522926594198330
// 2022/08/14 17:43:57 All burn amount 1164121594063664

func BuildBurnBalance() map[common.Hash]common.Hash {
	filePath := "./../chain/burn_list.json"
	storage := map[common.Hash]common.Hash{}
	gds := map[string][]BurnDetail{}
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(file).Decode(&gds); err != nil {
		panic(err)
	}
	bas := map[string]int64{}
	allBurnAmount := uint64(0)
	burnM := map[string]uint64{}
	for k, v := range gds {
		for _, vv := range v {
			addr, err := address.DecodeAddress(vv.From)
			if err != nil {
				panic(vv.From + "meer address err" + err.Error())
			}

			h16 := addr.Hash160()
			h16hex := hex.EncodeToString(h16[:])
			if _, ok := bas[h16hex]; !ok {
				bas[h16hex] = vv.Amount
			} else {
				bas[h16hex] += vv.Amount
			}
			allBurnAmount += uint64(vv.Amount)
			burnM[k] += uint64(vv.Amount)
		}
	}
	for k, v := range burnM {
		log.Println(k, "burn amount", v)
	}
	log.Println("All burn amount", allBurnAmount)
	pos := "0000000000000000000000000000000000000000000000000000000000000000"
	for k, v := range bas {
		b2, _ := hex.DecodeString(k + pos)
		key := crypto.Keccak256(b2)
		storage[common.BytesToHash(key)] = common.HexToHash(fmt.Sprintf("%064x", v))
	}
	return storage
}
