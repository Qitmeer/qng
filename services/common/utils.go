package common

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"strconv"
)

func GetQngAddrsFromPrivateKey(privateKeyStr string) ([]types.Address, error) {
	_, pkAddr, _, err := NewAddresses(privateKeyStr)
	if err != nil {
		return nil, err
	}
	addrs := make([]types.Address, 0)
	addrs = append(addrs, pkAddr)
	addrs = append(addrs, pkAddr.PKHAddress())
	return addrs, nil
}

type Account struct {
	EvmAcct   *accounts.Account
	UtxoAccts []types.Address
	Index     int
}

func NewAccount(act *accounts.Account, idx int, privKey *ecdsa.PrivateKey) (*Account, error) {
	privkeyHex := hex.EncodeToString(crypto.FromECDSA(privKey))
	addrs, err := GetQngAddrsFromPrivateKey(privkeyHex)
	if err != nil {
		return nil, err
	}
	return &Account{EvmAcct: act, UtxoAccts: addrs, Index: idx}, nil
}

func (a Account) String() string {
	ret := fmt.Sprintf("%d: %s", a.Index, a.EvmAcct.Address.String())
	for _, v := range a.UtxoAccts {
		ret = fmt.Sprintf("%s %s", ret, v.String())
	}
	return ret
}

func (a Account) PKAddress() types.Address {
	return a.UtxoAccts[0]
}

func (a Account) PKHAddress() types.Address {
	return a.UtxoAccts[1]
}

// MakeAddress converts an account specified directly as a hex encoded string or
// a key index in the key store to an internal account representation.
func MakeAddress(ks *keystore.KeyStore, account string) (accounts.Account, error) {
	// If the specified account is a valid address, return it
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}
	// Otherwise try to interpret the account as a keystore index
	index, err := strconv.Atoi(account)
	if err != nil || index < 0 {
		return accounts.Account{}, fmt.Errorf("invalid account address or index %q", account)
	}
	accs := ks.Accounts()
	if len(accs) <= index {
		return accounts.Account{}, fmt.Errorf("index %d higher than number of accounts %d", index, len(accs))
	}
	return accs[index], nil
}
