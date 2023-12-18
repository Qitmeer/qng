// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testprivatekey

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/crypto/bip32"
	"github.com/Qitmeer/qng/params"
)

const (
	CoinbaseIdx = 0
	Password    = "12345"
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

type Builder struct {
	// the bip32 master extended private key from a seed
	hdMaster *bip32.Key
	// the next hd child number from the master
	hdChildNumer uint32
	// privkeys cached all private keys which derived from the master private key.
	// the keys of the private key map are their hd child number.
	privkeys map[uint32][]byte
}

func NewBuilder(id uint32) (*Builder, error) {
	params := params.ActiveNetParams.Params
	// The final seed is seed || nodeId, the purpose to make sure that each harness
	// node use a deterministic private key based on the its node id.
	var finalSeed [hash.HashSize + 4]byte
	// t.Logf("seed is %v",hexutil.Encode(seed[:]))
	copy(finalSeed[:], defaultSeed[:])
	// t.Logf("finalseed is %v",hexutil.Encode(finalSeed[:]))
	binary.LittleEndian.PutUint32(finalSeed[hash.HashSize:], id)
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

	return &Builder{
		hdMaster:     hdMaster,
		hdChildNumer: 1,
		privkeys:     privkeys,
	}, nil
}

func (b *Builder) Build() ([]byte, error) {
	num := b.hdChildNumer
	childx, err := b.hdMaster.NewChildKey(num)
	if err != nil {
		return nil, err
	}
	b.privkeys[num] = childx.Key
	b.hdChildNumer++
	return childx.Key, nil
}

func (b *Builder) Get(idx int) []byte {
	if idx >= len(b.privkeys) {
		return nil
	}
	return b.privkeys[uint32(idx)]
}

func (b *Builder) GetHex(idx int) string {
	if idx >= len(b.privkeys) {
		return ""
	}
	return hex.EncodeToString(b.privkeys[uint32(idx)])
}
