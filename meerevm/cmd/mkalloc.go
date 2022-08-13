//go:build none
// +build none

package main

import (
	"encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/meerevm/chain"
	"math/big"
	"os"
	"sort"
	"strconv"

	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rlp"
)

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
	fileContent := "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run mkalloc.go \npackage chain\n\n"

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

		if len(ngd.Data.Contracts) > 0 {
			err = chain.UpdateAlloc(genesis, ngd.Data.Contracts)
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
