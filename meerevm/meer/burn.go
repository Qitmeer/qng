package meer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/testutils/release"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"strings"
)

const RELEASE_CONTRACT_ADDR = "0x1000000000000000000000000000000000000000"

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

func BuildBurnBalance(burnStr string) map[common.Hash]common.Hash {
	storage := map[common.Hash]common.Hash{}
	gds := map[string][]BurnDetail{}
	jsonR := strings.NewReader(burnStr)
	if err := json.NewDecoder(jsonR).Decode(&gds); err != nil {
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
				Amount: big.NewInt(vv.Amount),
				Time:   big.NewInt(vv.Time),
				Order:  big.NewInt(vv.Order),
				Height: big.NewInt(vv.Height),
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
		log.Trace(k, "burn amount", v)
	}
	log.Debug("All burn amount", allBurnAmount)
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
