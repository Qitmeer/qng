package simulator

import (
	"encoding/json"
	"github.com/Qitmeer/qng/config"
	qjson "github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types/pow"
	"testing"
)

func TestMockNode(t *testing.T) {
	node, err := StartMockNode(nil)
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()

	nodeinfo, _ := node.GetPublicBlockChainAPI().GetNodeInfo()

	jsonString, err := json.Marshal(nodeinfo.(*qjson.InfoNodeResult))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jsonString))
}

func TestGenerateBlocks(t *testing.T) {
	node, err := StartMockNode(nil)
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()

	targetBlockNum := uint32(5)
	ret, err := node.GetPrivateMinerAPI().Generate(targetBlockNum, pow.MEERXKECCAKV1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != int(targetBlockNum) {
		t.Fatalf("generate block number error: %d != %d ", len(ret), targetBlockNum)
	}
	info, err := node.GetPublicMinerAPI().GetMinerInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(info)

	blockCount, _ := node.GetPublicBlockAPI().GetBlockCount()
	if blockCount.(uint) != uint(targetBlockNum+1) {
		t.Fatalf("block count error: %d != %d ", blockCount.(uint), targetBlockNum+1)
	}
}

func TestOverrideCfg(t *testing.T) {
	node, err := StartMockNode(func(cfg *config.Config) error {
		cfg.DebugLevel = "trace"
		cfg.DebugPrintOrigins = true
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()
}
