package meer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	RELEASE_CONTRACT_ADDR = "0x1000000000000000000000000000000000000000"
	// 2022/08/17 20:35:36 MmQitmeerMainNetGuardAddressXd7b76q burn amount 514790066054534
	GuardAddress = "MmQitmeerMainNetGuardAddressXd7b76q"
	// 2022/08/17 20:35:36 MmQitmeerMainNetHonorAddressXY9JH2y burn amount 408011208230864
	HonorAddress = "MmQitmeerMainNetHonorAddressXY9JH2y"

	// record the records size
	SlotPositionOfRecordsSize = 0

	// record the field value
	SlotPositionOfRecordFieldValue = 1
)

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
	burnList := map[string][]BurnDetail{}
	jsonR := strings.NewReader(burnStr)
	if err := json.NewDecoder(jsonR).Decode(&burnList); err != nil {
		panic(err)
	}

	allBurnAmount := uint64(0)
	burnM := map[string]uint64{}
	recordsGroupByBurner := map[address.PubKeyHashAddress][]BurnDetail{}
	var handleBurnRecords = func(k string, burnRecords []BurnDetail) {
		for _, burnDetail := range burnRecords {
			burnerAddress, err := address.DecodeAddress(burnDetail.From)
			if err != nil {
				panic(burnDetail.From + "meer address err" + err.Error())
			}
			burnerPKHAddr := burnerAddress.(*address.PubKeyHashAddress)
			recordsGroupByBurner[*burnerPKHAddr] = append(recordsGroupByBurner[*burnerPKHAddr], burnDetail)
			allBurnAmount += uint64(burnDetail.Amount)
			burnM[k] += uint64(burnDetail.Amount)
		}
	}
	handleBurnRecords(GuardAddress, burnList[GuardAddress])
	handleBurnRecords(HonorAddress, burnList[HonorAddress])

	for burnerPKHAddr, burnRecords := range recordsGroupByBurner {
		storage[buildMappingRecordsSizeStorageSlotKey(&burnerPKHAddr)] = common.HexToHash(fmt.Sprintf("%064x", len(burnRecords)))
		for sequenceNumber, burnDetail := range burnRecords {
			storage[buildMappingFiledsPositionStorageSlotKey(&burnerPKHAddr, sequenceNumber, 0)] = common.HexToHash(fmt.Sprintf("%064x", big.NewInt(burnDetail.Amount)))
			storage[buildMappingFiledsPositionStorageSlotKey(&burnerPKHAddr, sequenceNumber, 1)] = common.HexToHash(fmt.Sprintf("%064x", big.NewInt(burnDetail.Time)))
			storage[buildMappingFiledsPositionStorageSlotKey(&burnerPKHAddr, sequenceNumber, 2)] = common.HexToHash(fmt.Sprintf("%064x", big.NewInt(burnDetail.Order)))
			storage[buildMappingFiledsPositionStorageSlotKey(&burnerPKHAddr, sequenceNumber, 3)] = common.HexToHash(fmt.Sprintf("%064x", big.NewInt(burnDetail.Height)))
		}
	}

	for k, v := range burnM {
		log.Trace(k, "burn amount", v)
	}
	log.Debug("All burn", "amount", allBurnAmount)
	return storage
}

/*
*

	solidity code like:
	struct BurnDetail {
	    uint amount; // valuePosition=0
	    uint time;   // valuePosition=1
	    uint order;  // valuePosition=2
	    uint height; // valuePosition=3
	}
	mapping(string => BurnDetail[]) burnUsers;
	@param mapKey is burnUsers key of user's address
	@param keyPosition is the BurnDetail[] index , slot storage position 0-1-2-3-4-5...
	@param valuePosition is the BurnDetail fields storage position 0-1-2-3, just the field order

*
*/
func buildMappingFiledsPositionStorageSlotKey(mapKey types.Address, keyPosition, valuePosition int) common.Hash {
	h16 := mapKey.Hash160()
	s := fmt.Sprintf("%x", h16[:]) + fmt.Sprintf("%064x", big.NewInt(SlotPositionOfRecordFieldValue))
	b, _ := hex.DecodeString(s)
	keyHash := crypto.Keccak256(b)
	s = fmt.Sprintf("%064x", big.NewInt(int64(keyPosition))) + hex.EncodeToString(keyHash)
	b, _ = hex.DecodeString(s)
	keyHash = crypto.Keccak256(b)
	key0Big := new(big.Int).Add(new(big.Int).SetBytes(keyHash), big.NewInt(int64(valuePosition)))
	return common.HexToHash(fmt.Sprintf("%064x", key0Big))
}

/*
*

	solidity code like:
	struct BurnDetail {
	    uint amount;
	    uint time;
	    uint order;
	    uint height;
	}
	mapping(string => BurnDetail[]) burnUsers;
	@param mapKey is burnUsers key of user's address

*
*/
func buildMappingRecordsSizeStorageSlotKey(mapKey types.Address) common.Hash {
	h16 := mapKey.Hash160()
	b, _ := hex.DecodeString(fmt.Sprintf("%x", h16[:]) + fmt.Sprintf("%064x", big.NewInt(SlotPositionOfRecordsSize)))
	keyHash := crypto.Keccak256(b)
	return common.BytesToHash(keyHash)
}
