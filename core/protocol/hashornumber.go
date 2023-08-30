package protocol

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"strconv"
)

type HashOrNumber struct {
	Hash   *hash.Hash
	Number uint32
}

func (hn *HashOrNumber) IsHash() bool {
	return hn.Hash != nil
}

func (hn *HashOrNumber) String() string {
	if hn.IsHash() {
		return "hash:" + hn.Hash.String()
	}
	return fmt.Sprintf("number:%d", hn.Number)
}

func NewHashOrNumber(data string) (*HashOrNumber, error) {
	if len(data) <= 0 {
		return nil, fmt.Errorf("HashOrNumber:no input data")
	}
	num, err := strconv.ParseUint(data, 10, 32)
	if err == nil {
		return &HashOrNumber{Number: uint32(num)}, nil
	}
	h, err := hash.NewHashFromStr(data)
	if err != nil {
		return nil, err
	}
	return &HashOrNumber{Hash: h}, nil
}
