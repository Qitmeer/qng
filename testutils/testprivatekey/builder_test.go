package testprivatekey

import (
	"github.com/Qitmeer/qng/common/util/hexutil"
	"github.com/Qitmeer/qng/params"
	"testing"
)

var (
	expect = struct {
		ver       string
		key       string
		chaincode string
		priv0     string
		priv1     string
	}{
		"0x040bee6e",
		"0x38015593945529cc0bd761108ad2fbd98a3f5f8e030c5acd3747ce3e54d95c16",
		"0x4eb4e56ada09795313734db329c362923c5b6fac75b924780e68b9c9b18a24b3",
		"0xe0b26a52b1a9676a365d6452fb04a1c05b58e959683862d73105e58d4416baba",
		"0xfff2cefe258ca60ae5f5abec99b5d63e2a561c40d784ee50b04eddf8efc84b0d",
	}
)

func TestPrivateKeyBuild(t *testing.T) {
	params.ActiveNetParams = &params.PrivNetParam
	pb, err := NewBuilder(0)
	if err != nil {
		t.Fatal(err)
	}

	if hexutil.Encode(pb.hdMaster.Key) != expect.key {
		t.Fatalf("hd master key not matched, expect %v but got %v", pb.hdMaster.Key, expect.key)
	}
	if hexutil.Encode(pb.hdMaster.Version) != expect.ver {
		t.Fatalf("hd master version not matched, expect %v but got %v", pb.hdMaster.Version, expect.ver)
	}
	if hexutil.Encode(pb.hdMaster.ChainCode) != expect.chaincode {
		t.Fatalf("hd master chain code not matched, expect %v but got %v", pb.hdMaster.ChainCode, expect.chaincode)
	}

	_, err = pb.Build()
	if err != nil {
		t.Fatalf("failed get new address : %v", err)
	}
	if hexutil.Encode(pb.Get(0)) != expect.priv0 {
		t.Fatalf("hd key0 priv key not matched, expect %x but got %v", pb.Get(0), expect.priv0)
	}
	if hexutil.Encode(pb.Get(1)) != expect.priv1 {
		t.Fatalf("hd key0 priv key not matched, expect %x but got %v", pb.Get(1), expect.priv1)
	}
}
