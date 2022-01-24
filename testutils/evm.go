// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"encoding/hex"
	"errors"
	"math/big"
)

func (w *testWallet) CreateExportRawTx(txid string, amount, fee int64) (string, error) {
	amount -= fee
	if txid == "" {
		return txid, errors.New("CreateExportRawTx Error,Amount Not Enough")
	}
	s, err := w.client.CreateExportRawTx(txid, w.pkAddrs[0].String(),
		0, amount)
	if err != nil {
		return "", err
	}
	s, err = w.client.TestSign(hex.EncodeToString(w.privkeys[0]), s, "")
	if err != nil {
		return "", err
	}
	tx, err := w.client.SendRawTx(s, true)
	if err != nil {
		return "", err
	}
	return tx.String(), nil
}

func (w *testWallet) GetBalance(addr string) (*big.Int, error) {
	hb, err := w.evmClient.GetEthBalance(addr)
	if err != nil {
		return nil, err
	}
	return ConvertEthToMeer(hb), nil
}
