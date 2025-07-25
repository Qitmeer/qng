package address

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/Qitmeer/qng/v2/services/common"
	"github.com/Qitmeer/qng/v2/testutils/testprivatekey"
	"testing"
)

func TestNewAddresses(t *testing.T) {
	params.ActiveNetParams = &params.PrivNetParam
	pb, err := testprivatekey.NewBuilder(0)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyHex := hex.EncodeToString(pb.Get(0))
	privateKey, addr, eaddr, err := common.NewAddresses(privateKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	pkHex := hex.EncodeToString(privateKey.Serialize())
	if pkHex != privateKeyHex {
		t.Fatalf("%s != %s (expect)", pkHex, privateKeyHex)
	}
	t.Logf("privateKey:%s addr:%s pkAddr:%s evmAddr:%s", pkHex, addr.PKHAddress().String(), addr.String(), eaddr.String())
}
