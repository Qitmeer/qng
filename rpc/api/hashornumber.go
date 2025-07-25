package api

import (
	"fmt"
	qcommon "github.com/Qitmeer/qng/v2/common"
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/ethereum/go-ethereum/common"
	"strconv"
)

type HashOrNumber struct {
	Hash   *hash.Hash
	EVM    common.Hash
	Number uint64
}

func (hn *HashOrNumber) IsHash() bool {
	return hn.Hash != nil || hn.EVM != common.Hash{}
}

func (hn *HashOrNumber) String() string {
	if hn.IsHash() {
		if hn.Hash != nil {
			return "hash:" + hn.Hash.String()
		} else {
			return "hash:" + hn.EVM.String()
		}

	}
	return fmt.Sprintf("number:%d", hn.Number)
}

func NewHashOrNumber(data string) (*HashOrNumber, error) {
	if len(data) <= 0 {
		return nil, fmt.Errorf("HashOrNumber:no input data")
	}
	hn := &HashOrNumber{
		Hash:   nil,
		EVM:    common.Hash{},
		Number: 0,
	}
	num, err := strconv.ParseUint(data, 10, 64)
	if err == nil {
		hn.Number = num
		return hn, nil
	}
	if qcommon.Has0xPrefix(data) {
		hn.EVM = common.HexToHash(data)
	} else {
		h, err := hash.NewHashFromStr(data)
		if err != nil {
			return nil, err
		}
		hn.Hash = h
	}
	return hn, nil
}

func NewHashOrNumberByNumber(number uint64) *HashOrNumber {
	return &HashOrNumber{
		Hash:   nil,
		EVM:    common.Hash{},
		Number: number,
	}
}
