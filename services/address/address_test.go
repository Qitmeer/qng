package address

import (
	"encoding/hex"
	"testing"
)

func TestNewAddresses(t *testing.T) {
	privateKeyHex := "fff2cefe258ca60ae5f5abec99b5d63e2a561c40d784ee50b04eddf8efc84b0d"
	privateKey, addr, eaddr, err := NewAddresses(privateKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	pkHex := hex.EncodeToString(privateKey.Serialize())
	if pkHex != privateKeyHex {
		t.Fatalf("%s != %s (expect)", pkHex, privateKeyHex)
	}
	t.Logf("privateKey:%s addr:%s pkAddr:%s evmAddr:%s", pkHex, addr.PKHAddress().String(), addr.String(), eaddr.String())
}
