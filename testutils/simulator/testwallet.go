// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package simulator

import (
	"encoding/binary"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/bip32"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	// the default seed used in the testWallet
	defaultSeed = [hash.HashSize]byte{
		0x7e, 0x44, 0x5a, 0xa5, 0xff, 0xd8, 0x34, 0xcb,
		0x2d, 0x3b, 0x2d, 0xb5, 0x0f, 0x89, 0x97, 0xdd,
		0x21, 0xaf, 0x29, 0xbe, 0xc3, 0xd2, 0x96, 0xaa,
		0xa0, 0x66, 0xd9, 0x02, 0xb9, 0x3f, 0x48, 0x4b,
	}
)

// testWallet is a simple in-memory wallet works for a test harness instance's
// node. the purpose of testWallet is to provide basic wallet functionality for
// the integrated-test, such as send tx & verify balance etc.
// testWallet works as a HD (BIP-32) wallet
type testWallet struct {
	// the node id which wallet is targeted for
	nodeId uint32
	// the bip32 master extended private key from a seed
	hdMaster *bip32.Key
	// the next hd child number from the master
	hdChildNumer uint32
	// addrs are all addresses which belong to the master private key.
	// the keys of address map are their hd child numbers.
	addrs    map[uint32]types.Address
	pkAddrs  map[uint32]*address.SecpPubKeyAddress
	ethAddrs map[uint32]common.Address
	// privkeys cached all private keys which derived from the master private key.
	// the keys of the private key map are their hd child number.
	privkeys map[uint32][]byte
}

func newTestWallet(nodeId uint32) (*testWallet, error) {
	params := params.ActiveNetParams.Params
	// The final seed is seed || nodeId, the purpose to make sure that each harness
	// node use a deterministic private key based on the its node id.
	var finalSeed [hash.HashSize + 4]byte
	// t.Logf("seed is %v",hexutil.Encode(seed[:]))
	copy(finalSeed[:], defaultSeed[:])
	// t.Logf("finalseed is %v",hexutil.Encode(finalSeed[:]))
	binary.LittleEndian.PutUint32(finalSeed[hash.HashSize:], nodeId)
	version := bip32.Bip32Version{
		PrivKeyVersion: params.HDPrivateKeyID[:],
		PubKeyVersion:  params.HDPublicKeyID[:],
	}
	// t.Logf("finalseed is %v",hexutil.Encode(finalSeed[:]))
	hdMaster, err := bip32.NewMasterKey2(finalSeed[:], version)
	if err != nil {
		return nil, err
	}
	child0, err := hdMaster.NewChildKey(0)
	if err != nil {
		return nil, err
	}
	key0 := child0.Key
	privkeys := make(map[uint32][]byte)
	privkeys[0] = key0
	addr0, err := testutils.PrivateKeyToAddr(key0, params)
	if err != nil {
		return nil, err
	}
	pkAddr0, err := testutils.PrivateKeyToPkAddress(key0, params)
	if err != nil {
		return nil, err
	}
	ethAddr0, err := testutils.PrivateKeyToETHAddress(key0)
	if err != nil {
		return nil, err
	}
	addrs := make(map[uint32]types.Address)
	pkAddrs := make(map[uint32]*address.SecpPubKeyAddress)
	ethAddrs := make(map[uint32]common.Address)
	addrs[0] = addr0
	pkAddrs[0] = pkAddr0
	ethAddrs[0] = ethAddr0
	return &testWallet{
		nodeId:       nodeId,
		hdMaster:     hdMaster,
		hdChildNumer: 1,
		privkeys:     privkeys,
		addrs:        addrs,
		ethAddrs:     ethAddrs,
		pkAddrs:      pkAddrs,
	}, nil
}

// newAddress create a new address from the wallet's key chain.
func (w *testWallet) newAddress() (types.Address, error) {
	num := w.hdChildNumer
	childx, err := w.hdMaster.NewChildKey(num)
	if err != nil {
		return nil, err
	}
	w.privkeys[num] = childx.Key
	addrx, err := testutils.PrivateKeyToAddr(childx.Key, params.ActiveNetParams.Params)
	if err != nil {
		return nil, err
	}
	pkAddrx, err := testutils.PrivateKeyToPkAddress(childx.Key, params.ActiveNetParams.Params)
	if err != nil {
		return nil, err
	}
	ethAddrx, err := testutils.PrivateKeyToETHAddress(childx.Key)
	if err != nil {
		return nil, err
	}
	w.addrs[num] = addrx
	w.pkAddrs[num] = pkAddrx
	w.ethAddrs[num] = ethAddrx
	w.hdChildNumer++
	return addrx, nil
}

func (w *testWallet) miningAddr() types.Address {
	return w.pkAddrs[0]
}
