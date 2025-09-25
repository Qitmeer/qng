package mcp

import (
	"fmt"
	"testing"

	"github.com/Qitmeer/qng/meerevm/meer"
)

func TestParseJs(t *testing.T) {
	helps, err := ParseToolHelps(meer.QngJs)
	if err != nil {
		panic(err)
	}
	for _, h := range helps {
		fmt.Printf("Method: %-20s | ParamsNum: %d\n", h.Method, h.ParamsNum)
	}
}
