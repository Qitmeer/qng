//go:build none
// +build none

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rlp"
)

type allocItem struct{ Addr, Balance *big.Int }

type allocList []allocItem

func (a allocList) Len() int           { return len(a) }
func (a allocList) Less(i, j int) bool { return a[i].Addr.Cmp(a[j].Addr) < 0 }
func (a allocList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func makelist(g *core.Genesis) allocList {
	a := make(allocList, 0, len(g.Alloc))
	for addr, account := range g.Alloc {
		if len(account.Storage) > 0 || len(account.Code) > 0 || account.Nonce != 0 {
			panic(fmt.Sprintf("can't encode account %x", addr))
		}
		bigAddr := new(big.Int).SetBytes(addr.Bytes())
		a = append(a, allocItem{bigAddr, account.Balance})
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
	var filePath string
	if len(os.Args) != 2 {
		filePath = "./genesis.json"
	} else {
		filePath = os.Args[1]
	}

	g := new(core.Genesis)
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(file).Decode(g); err != nil {
		panic(err)
	}

	fileName := "./genesis_alloc.go"

	f, err := os.Create(fileName)

	if err != nil {
		panic(fmt.Sprintf("Save error:%s  %s", fileName, err))
	}
	defer func() {
		err = f.Close()
	}()

	fileContent := "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run mkalloc.go \npackage chain\n\n"
	fileContent += fmt.Sprintf("const allocData = %s", makealloc(g))

	f.WriteString(fileContent)

	fmt.Println("Successfully updated:", fileName)
}
