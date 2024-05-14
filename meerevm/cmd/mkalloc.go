//go:build none
// +build none

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils/release"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"log"
	"math/big"
	"os"
	"sort"
	"strconv"
)

const RELEASE_CONTRACT_ADDR = "0x1000000000000000000000000000000000000000"

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
		hm := []string{}
		for hk := range account.Storage {
			hm = append(hm, hk.String())
		}
		sort.Strings(hm)
		for kk := 0; kk < len(hm); kk++ {
			k := common.HexToHash(hm[kk])
			v := account.Storage[k]
			sks = append(sks, k.String())
			svs = append(svs, v.String())
		}
		a = append(a, allocItem{Addr: bigAddr, Balance: account.Balance, Code: account.Code, Nonce: account.Nonce, StorageKey: sks, StorageValue: svs})
	}
	sort.Sort(a)
	return a
}

func makealloc(g *core.Genesis) (string, []byte) {
	a := makelist(g)
	data, err := rlp.EncodeToBytes(a)
	if err != nil {
		panic(err)
	}
	return strconv.QuoteToASCII(string(data)), data
}

func main() {
	filePath := "./../meer/genesis.json"
	gds := []meer.NetGenesisData{}
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
	burnList := BuildBurnBalance()
	fileContent := "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run mkalloc.go \npackage meer\n\n"

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
		genesis := meer.Genesis()
		genesis.Alloc = ngd.Data.Genesis.Alloc
		if _, ok := genesis.Alloc[common.HexToAddress(RELEASE_CONTRACT_ADDR)]; ok {
			releaseAccount := genesis.Alloc[common.HexToAddress(RELEASE_CONTRACT_ADDR)]
			storage := releaseAccount.Storage
			for k, v := range burnList {
				storage[k] = v
			}
			releaseAccount.Storage = storage
			genesis.Alloc[common.HexToAddress(RELEASE_CONTRACT_ADDR)] = releaseAccount
		}
		if len(ngd.Data.Contracts) > 0 {
			err = meer.UpdateAlloc(genesis, ngd.Data.Contracts)
			if err != nil {
				panic(err)
			}
		}
		alloc, src := makealloc(genesis)
		log.Printf("network=%s genesisHash=%s\n", networkTag, hex.EncodeToString(crypto.Keccak256([]byte(src))))
		fileContent += fmt.Sprintf("\nconst %s = %s", networkTag, alloc)
	}

	fileName := "./../meer/genesis_alloc.go"

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
	Time   int64  `json:"time"`
}

// 2022/08/17 20:35:36 MmQitmeerMainNetHonorAddressXY9JH2y burn amount 408011208230864
// 2022/08/17 20:35:36 MmQitmeerMainNetGuardAddressXd7b76q burn amount 514790066054534
// 2022/08/17 20:35:36 All burn amount 922801274285398
// 2022/08/14 17:43:57 end height 910000
// 2022/08/14 17:43:57 end order 1013260
// 2022/08/14 17:43:57 end blockhash efc89d8b4ef5733b6e566d9f06c0596075100f8406d3a9b581c74d42fb99dd79
// 2022/08/14 17:43:57 pow meer amount (1013260 /10) * 12 * 10 = 1013260 * 12 = 12159120
// all amount 1215912000000000+922801274285398 = 2138713274285398

func BuildBurnBalance() map[common.Hash]common.Hash {
	filePath := "./../meer/burn_list.json"
	storage := map[common.Hash]common.Hash{}
	gds := map[string][]BurnDetail{}
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(file).Decode(&gds); err != nil {
		panic(err)
	}
	bas := map[string][]release.MeerMappingBurnDetail{}
	allBurnAmount := uint64(0)
	burnM := map[string]uint64{}
	for k, v := range gds {
		for _, vv := range v {
			addr, err := address.DecodeAddress(vv.From)
			if err != nil {
				panic(vv.From + "meer address err" + err.Error())
			}
			d := release.MeerMappingBurnDetail{
				big.NewInt(vv.Amount),
				big.NewInt(vv.Time),
				big.NewInt(vv.Order),
				big.NewInt(vv.Height),
			}
			//parsed, _ := abi.JSON(strings.NewReader(release.TokenMetaData.ABI))
			//// constructor params
			//hexData, _ := parsed.Pack("", d)
			h16 := addr.Hash160()
			h16hex := hex.EncodeToString(h16[:])
			if _, ok := bas[h16hex]; !ok {
				bas[h16hex] = []release.MeerMappingBurnDetail{}
			}
			bas[h16hex] = append(bas[h16hex], d)
			allBurnAmount += uint64(vv.Amount)
			burnM[k] += uint64(vv.Amount)
		}
	}
	for k, v := range burnM {
		log.Println(k, "burn amount", v)
	}
	log.Println("All burn amount", allBurnAmount)
	for k, v := range bas {
		for i, vv := range v {
			// amount
			s := k + fmt.Sprintf("%064x", big.NewInt(1))
			b, _ := hex.DecodeString(s)
			key0 := crypto.Keccak256(b)
			s = fmt.Sprintf("%064x", big.NewInt(int64(i))) + hex.EncodeToString(key0)
			b, _ = hex.DecodeString(s)
			key0 = crypto.Keccak256(b)
			key0Big := new(big.Int).Add(new(big.Int).SetBytes(key0), big.NewInt(0))
			storage[common.HexToHash(fmt.Sprintf("%064x", key0Big))] = common.HexToHash(fmt.Sprintf("%064x", vv.Amount))
			// time
			s = k + fmt.Sprintf("%064x", big.NewInt(1))
			b, _ = hex.DecodeString(s)
			key1 := crypto.Keccak256(b)
			s = fmt.Sprintf("%064x", big.NewInt(int64(i))) + hex.EncodeToString(key1)
			b, _ = hex.DecodeString(s)
			key1 = crypto.Keccak256(b)
			key1Big := new(big.Int).Add(new(big.Int).SetBytes(key1), big.NewInt(1))
			storage[common.HexToHash(fmt.Sprintf("%064x", key1Big))] = common.HexToHash(fmt.Sprintf("%064x", vv.Time))
			// order
			s = k + fmt.Sprintf("%064x", big.NewInt(1))
			b, _ = hex.DecodeString(s)
			key2 := crypto.Keccak256(b)
			s = fmt.Sprintf("%064x", big.NewInt(int64(i))) + hex.EncodeToString(key2)
			b, _ = hex.DecodeString(s)
			key2 = crypto.Keccak256(b)
			key2Big := new(big.Int).Add(new(big.Int).SetBytes(key2), big.NewInt(2))
			storage[common.HexToHash(fmt.Sprintf("%064x", key2Big))] = common.HexToHash(fmt.Sprintf("%064x", vv.Order))
			// height
			s = k + fmt.Sprintf("%064x", big.NewInt(1))
			b, _ = hex.DecodeString(s)
			key3 := crypto.Keccak256(b)
			s = fmt.Sprintf("%064x", big.NewInt(int64(i))) + hex.EncodeToString(key3)
			b, _ = hex.DecodeString(s)
			key3 = crypto.Keccak256(b)
			key3Big := new(big.Int).Add(new(big.Int).SetBytes(key3), big.NewInt(3))
			storage[common.HexToHash(fmt.Sprintf("%064x", key3Big))] = common.HexToHash(fmt.Sprintf("%064x", vv.Height))
		}
		kk, _ := hex.DecodeString(k + fmt.Sprintf("%064x", big.NewInt(0)))
		kb := crypto.Keccak256(kk)
		storage[common.BytesToHash(kb)] = common.HexToHash(fmt.Sprintf("%064x", len(v)))
	}
	return storage
}
