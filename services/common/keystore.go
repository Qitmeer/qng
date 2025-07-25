package common

import (
	"fmt"
	"github.com/Qitmeer/qng/v2/core/address"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"os"
	"reflect"
	"sync"
	"time"
)

type unlocked struct {
	*keystore.Key
	abort chan struct{}
	acc   *Account
}

// KeyStoreType is the reflect type of a keystore backend.
var KeyStoreType = reflect.TypeOf(&QngKeyStore{})

func init() {
	keystore.KeyStoreType = KeyStoreType
}

type QngKeyStore struct {
	*keystore.KeyStore
	mu       sync.RWMutex
	unlocked map[common.Address]*unlocked
	index    int
}

func NewQngKeyStore(ks *keystore.KeyStore) *QngKeyStore {
	return &QngKeyStore{
		KeyStore: ks,
		mu:       sync.RWMutex{},
		unlocked: map[common.Address]*unlocked{},
		index:    0,
	}
}

func (ks *QngKeyStore) TimedUnlock(a accounts.Account, passphrase string, timeout time.Duration) error {
	tu := func() error {
		return ks.KeyStore.TimedUnlock(a, passphrase, timeout)
	}
	qa, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return tu()
	}
	if qa.Address != a.Address {
		return fmt.Errorf("%s != %s", qa.Address, a.Address)
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()
	u, found := ks.unlocked[a.Address]
	if found {
		if u.abort == nil {
			return tu()
		}
		// Terminate the expire goroutine and replace it below.
		close(u.abort)
	}
	var acc *Account
	if u == nil {
		acc, err = NewAccount(&a, ks.index, key.PrivateKey)
		if err != nil {
			return err
		}
		ks.index++

	} else {
		acc = u.acc
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{}), acc: acc}
		go ks.expire(a.Address, u, timeout)
	} else {
		u = &unlocked{Key: key, acc: acc}
	}
	ks.unlocked[a.Address] = u

	return tu()
}

func (ks *QngKeyStore) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		ks.mu.Lock()
		if ks.unlocked[addr] == u {
			u.PrivateKey = nil
		}
		ks.mu.Unlock()
	}
}

func (ks *QngKeyStore) getDecryptedKey(a accounts.Account, auth string) (accounts.Account, *keystore.Key, error) {
	a, err := ks.Find(a)
	if err != nil {
		return a, nil, err
	}
	key, err := LoadKey(a.Address, a.URL.Path, auth)
	return a, key, err
}

func (ks *QngKeyStore) Lock(addr common.Address) error {
	ks.mu.Lock()
	unl, found := ks.unlocked[addr]
	ks.mu.Unlock()
	if found {
		ks.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	}
	return ks.KeyStore.Lock(addr)
}

func (ks *QngKeyStore) Locks(addre string) error {
	var eAddr common.Address
	if common.IsHexAddress(addre) {
		eAddr = common.HexToAddress(addre)
	} else {
		addr, err := address.DecodeAddress(addre)
		if err != nil {
			return err
		}
		if !addr.IsForNetwork(params.ActiveNetParams.Net) {
			return fmt.Errorf("network error:%s", addr.String())
		}
		for _, a := range ks.unlocked {
			if a.acc.PKHAddress().String() == addr.String() ||
				a.acc.PKAddress().String() == addr.String() {
				eAddr = a.Address
				break
			}
		}
	}
	return ks.KeyStore.Lock(eAddr)
}

func (ks *QngKeyStore) getUnlocked(addre string) *unlocked {
	var u *unlocked
	for _, a := range ks.unlocked {
		if a.acc.PKHAddress().String() == addre ||
			a.acc.PKAddress().String() == addre ||
			a.Address.String() == addre {
			u = a
			break
		}
	}
	if u == nil {
		return nil
	}
	return u
}

func (ks *QngKeyStore) GetKey(addre string) *keystore.Key {
	u := ks.getUnlocked(addre)
	if u == nil || u.PrivateKey == nil {
		return nil
	}
	return u.Key
}

func (ks *QngKeyStore) GetQngAccount(addre string) *Account {
	u := ks.getUnlocked(addre)
	if u == nil {
		return nil
	}
	return u.acc
}

func (ks *QngKeyStore) GetQngAccountByIdx(idx int) *Account {
	var u *unlocked
	for _, a := range ks.unlocked {
		if a.acc.Index == idx {
			u = a
			break
		}
	}
	if u == nil {
		return nil
	}
	return u.acc
}

func LoadKey(addr common.Address, filename, auth string) (*keystore.Key, error) {
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
