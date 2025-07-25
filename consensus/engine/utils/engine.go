package utils

import (
	"fmt"
	"github.com/Qitmeer/qng/v2/consensus/engine"
	"github.com/Qitmeer/qng/v2/consensus/engine/poa/types"
	"github.com/Qitmeer/qng/v2/consensus/engine/pow"
	"io"
)

func NewConsensusEngine(r io.Reader) (engine.Engine, error) {
	b := make([]byte, 1)
	n, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	if n != len(b) {
		return nil, fmt.Errorf("Read length error:%d != %d", n, len(b))
	}
	if engine.EngineType(b[0]).IsPoA() {
		return types.New(r)
	} else if engine.EngineType(b[0]).IsPoW() {
		return pow.New(r, b[0])
	}
	return nil, fmt.Errorf("Not support:%s", engine.EngineType(b[0]).String())
}
