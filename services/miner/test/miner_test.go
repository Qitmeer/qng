package test

import (
	"bytes"
	"encoding/hex"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils"
	"strconv"
	"testing"
	"time"
)

func TestMining(t *testing.T) {
	node, err := testutils.StartMockNode(nil)
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()
	powType := pow.MEERXKECCAKV1
	ret, err := node.GetPublicMinerAPI().GetRemoteGBT(byte(powType), nil)
	if err != nil {
		t.Fatal(err)
	}
	hexBlockHeader, ok := ret.(string)
	if !ok {
		t.Fatalf("Not RemoteGBTResult: %v", ret)
	}
	t.Logf("Receive RemoteGBTResult: %s", hexBlockHeader)
	if len(hexBlockHeader)%2 != 0 {
		hexBlockHeader = "0" + hexBlockHeader
	}
	serializedBlockHeader, err := hex.DecodeString(hexBlockHeader)
	if err != nil {
		t.Fatal(err)
	}
	var header types.BlockHeader
	err = header.Deserialize(bytes.NewReader(serializedBlockHeader))
	if err != nil {
		t.Fatal(err)
	}
	// Initial state.
	hashesCompleted := uint64(0)
	maxNonce := ^uint64(0)
	i := uint64(0)
	start := time.Now()
	h, err := node.GetPublicBlockAPI().GetMainChainHeight()
	if err != nil {
		t.Fatal(err)
	}
	height, err := strconv.ParseUint(h.(string), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	height++
	for ; i <= maxNonce; i++ {
		instance := pow.GetInstance(powType, 0, []byte{})
		instance.SetNonce(i)
		instance.SetMainHeight(pow.MainHeight(height))
		instance.SetParams(params.ActiveNetParams.Params.PowConfig)
		hashesCompleted += 2
		header.Pow = instance

		if header.Pow.FindSolver(header.BlockData(), header.BlockHash(), header.Difficulty) {
			t.Logf("Find %s (hash) at %d (nonce) %d (height) %d (hashes)", header.BlockHash().String(), i, height, hashesCompleted)
			var headerBuf bytes.Buffer
			err = header.Serialize(&headerBuf)
			if err != nil {
				t.Fatal(err)
			}
			hexBH := hex.EncodeToString(headerBuf.Bytes())
			t.Logf("Submit block header: %s", hexBH)
			_, err := node.GetPublicMinerAPI().SubmitBlockHeader(hexBH, nil)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	tip, err := node.GetPublicBlockAPI().Tips()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%v", tip.(json.TipsInfo))

	seconds := time.Since(start).Seconds()
	if seconds > 0 {
		t.Logf("Hashrate: %.6f hash/s\n", float64(hashesCompleted)/seconds)
	}
}
