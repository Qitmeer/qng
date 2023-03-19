package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"os"
	"sync"
	"time"
)

type unlocked struct {
	*keystore.Key
	abort chan struct{}
}

type QngKeyStore struct {
	*keystore.KeyStore
	mu       sync.RWMutex
	unlocked map[types.Address]*unlocked
}

func NewQngKeyStore(ks *keystore.KeyStore) *QngKeyStore {
	return &QngKeyStore{
		ks,
		sync.RWMutex{},
		map[types.Address]*unlocked{},
	}
}

func GetQngAddrsFromPrivateKey(privateKeyStr string, param *params.Params) ([]types.Address, error) {
	data, err := hex.DecodeString(privateKeyStr)
	if err != nil {
		return nil, err
	}
	_, pubKey := ecc.Secp256k1.PrivKeyFromBytes(data)
	addrs := make([]types.Address, 0)
	//pk addr
	addr, err := address.NewSecpPubKeyAddress(pubKey.SerializeCompressed(), param)
	if err != nil {
		return nil, err
	}

	addrs = append(addrs, addr)
	addrs = append(addrs, addr.PKHAddress())
	return addrs, nil
}

func (ks *QngKeyStore) expire(addr types.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		ks.mu.Lock()
		// only drop if it's still the same key instance that dropLater
		// was launched with. we can check that using pointer equality
		// because the map stores a new pointer every time the key is
		// unlocked.
		if ks.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(ks.unlocked, addr)
		}
		ks.mu.Unlock()
	}
}

func (ks *QngKeyStore) GetKey(addr common.Address, filename, auth string) (*keystore.Key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

func (ks *QngKeyStore) getDecryptedKey(a accounts.Account, auth string) (accounts.Account, *keystore.Key, error) {
	a, err := ks.Find(a)
	if err != nil {
		return a, nil, err
	}
	key, err := ks.GetKey(a.Address, a.URL.Path, auth)
	return a, key, err
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}
