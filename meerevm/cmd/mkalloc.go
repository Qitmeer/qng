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
	privateKeyHex := ""
	if len(os.Args) >= 2 {
		privateKeyHex = os.Args[1]
	}
	if len(os.Args) >= 3 {
		network := os.Args[2]
		if network == params.TestNetParam.Name {
			params.ActiveNetParams = &params.TestNetParam
		} else if network == params.PrivNetParam.Name {
			params.ActiveNetParams = &params.PrivNetParam
		} else if network == params.MixNetParam.Name {
			params.ActiveNetParams = &params.MixNetParam
		} else {
			params.ActiveNetParams = &params.MainNetParam
		}
	}

	gd := new(chain.GenesisData)
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(file).Decode(gd); err != nil {
		panic(err)
	}
	chain.ChainConfig.ChainID = big.NewInt(params.ActiveNetParams.MeerEVMCfg.ChainID)
	genesis := chain.DefaultGenesisBlock(chain.ChainConfig)
	genesis.Alloc = gd.Genesis.Alloc

	if len(gd.Contracts) > 0 {
		if len(privateKeyHex) <= 0 {
			panic("You must enter a private key")
		}
		err = chain.UpdateAlloc(genesis, gd.Contracts, privateKeyHex)
		if err != nil {
			panic(err)
		}
	}

	fileName := "./../chain/genesis_alloc.go"

	f, err := os.Create(fileName)

	if err != nil {
		panic(fmt.Sprintf("Save error:%s  %s", fileName, err))
	}
	defer func() {
		err = f.Close()
	}()

	fileContent := "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run mkalloc.go [privateKey] \npackage chain\n\n"
	fileContent += fmt.Sprintf("const allocData = %s", makealloc(genesis))

	f.WriteString(fileContent)

	fmt.Println("Successfully updated:", fileName)
}
