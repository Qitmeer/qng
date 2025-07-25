package wallet

import (
	ejson "encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/v2/core/json"
	qcommon "github.com/Qitmeer/qng/v2/services/common"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

// PublicWalletManagerAPI provides an API to access Qng wallet function
// information.
type PublicWalletManagerAPI struct {
	a *WalletManager
}

func NewPublicWalletAPI(m *WalletManager) *PublicWalletManagerAPI {
	pmAPI := &PublicWalletManagerAPI{m}
	return pmAPI
}

// Lock removes the private key with the given address from memory.
func (api *PublicWalletManagerAPI) Lock(addres string) error {
	return api.a.qks.Locks(addres)
}

// SendToAddress handles a sendtoaddress RPC request by creating a new
// transaction spending unspent transaction outputs for a wallet to another
// payment address.  Leftover inputs not sent to the payment address or a fee
// for the miner are sent back to a new address in the wallet.  Upon success,
// the TxID for the created transaction is returned.
func (api *PublicWalletManagerAPI) SendToAddress(fromAddress string, to string, lockTime int64) (string, error) {
	b := []byte(to)
	var amounts json.AddressAmountV3
	err := ejson.Unmarshal(b, &amounts)
	if err != nil {
		return "", err
	}
	for _, a := range amounts {
		// Check that signed integer parameters are positive.
		if a.Amount <= 0 {
			return "", fmt.Errorf("amount must be positive")
		}
	}

	return api.a.sendTx(fromAddress, amounts, 0, lockTime)
}

func (api *PublicWalletManagerAPI) ListAccount() (interface{}, error) {
	ws := api.a.qks.Wallets()
	res := []map[string]interface{}{}
	for _, w := range ws {
		as := w.Accounts()
		if len(as) <= 0 {
			continue
		}
		status, err := w.Status()
		if err != nil {
			continue
		}
		for _, ac := range as {
			qa := api.a.qks.GetQngAccount(ac.Address.String())
			if qa != nil {
				jsM := map[string]interface{}{
					"index":    qa.Index,
					"eAddress": ac.Address,
					"path":     ac.URL,
					"status":   status,
				}
				for i, a := range qa.UtxoAccts {
					jsM[fmt.Sprintf("address_%d", i+1)] = a.String()
				}
				res = append(res, jsM)
			} else {
				res = append(res, map[string]interface{}{
					"eAddress": ac.Address,
					"path":     ac.URL,
					"status":   status,
				})
			}
		}

	}
	return res, nil
}

// PrivateWalletManagerAPI provides an API to access Qng wallet function
// information.
type PrivateWalletManagerAPI struct {
	a *WalletManager
}

func NewPrivateWalletAPI(m *WalletManager) *PrivateWalletManagerAPI {
	pmAPI := &PrivateWalletManagerAPI{m}
	return pmAPI
}

// ImportRawKey stores the given hex encoded ECDSA key into the key directory,
// encrypting it with the passphrase.
func (api *PrivateWalletManagerAPI) ImportRawKey(privkey string, password string) (common.Address, error) {
	ac, err := api.a.ImportRawKey(privkey, password)
	if err != nil {
		return common.Address{}, err
	}
	return ac.Address, nil
}

func (api *PrivateWalletManagerAPI) Unlock(account, passphrase string, timeout time.Duration) error {
	a, err := qcommon.MakeAddress(api.a.qks.KeyStore, account)
	if err != nil {
		return err
	}
	return api.a.qks.TimedUnlock(a, passphrase, timeout)
}
