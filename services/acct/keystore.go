package acct

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
	"github.com/ethereum/go-ethereum/cmd/utils"
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

func (api *PublicAccountManagerAPI) Unlock(account, passphrase string, timeout time.Duration) error {
	a, err := utils.MakeAddress(api.a.qks.KeyStore, account)
	if err != nil {
		return err
	}
	_, key, err := api.a.qks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}

	api.a.qks.mu.Lock()
	defer api.a.qks.mu.Unlock()
	addrs, err := GetQngAddrsFromPrivateKey(key.PrivateKey.D.String(), api.a.chain.ChainParams())
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		u, found := api.a.qks.unlocked[addr]
		if found {
			if u.abort == nil {
				// The address was unlocked indefinitely, so unlocking
				// it with a timeout would be confusing.
				zeroKey(key.PrivateKey)
				return nil
			}
			// Terminate the expire goroutine and replace it below.
			close(u.abort)
		}
		if timeout > 0 {
			u = &unlocked{Key: key, abort: make(chan struct{})}
			go api.a.qks.expire(addr, u, timeout)
		} else {
			u = &unlocked{Key: key}
		}
		api.a.qks.unlocked[addr] = u
	}
	return nil
}

// Lock removes the private key with the given address from memory.
func (api *PublicAccountManagerAPI) Lock(addres string) error {
	addr, err := address.DecodeAddress(addres)
	if err != nil {
		return err
	}
	api.a.qks.mu.Lock()
	if unl, found := api.a.qks.unlocked[addr]; found {
		api.a.qks.mu.Unlock()
		api.a.qks.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	} else {
		api.a.qks.mu.Unlock()
	}

	return nil
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}
