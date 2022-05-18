// Copyright 2017-2018 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package qx

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/params"
)

type QitmeerBase58checkVersionFlag struct {
	Ver  []byte
	flag string
	PK   bool
}

func (n *QitmeerBase58checkVersionFlag) Set(s string) error {
	n.Ver = []byte{}
	switch s {
	case "mainnet":
		if n.PK {
			n.Ver = append(n.Ver, params.MainNetParams.PubKeyAddrID[0:]...)
		} else {
			n.Ver = append(n.Ver, params.MainNetParams.PubKeyHashAddrID[0:]...)
		}

	case "privnet":
		if n.PK {
			n.Ver = append(n.Ver, params.PrivNetParams.PubKeyAddrID[0:]...)
		} else {
			n.Ver = append(n.Ver, params.PrivNetParams.PubKeyHashAddrID[0:]...)
		}
	case "testnet":
		if n.PK {
			n.Ver = append(n.Ver, params.TestNetParams.PubKeyAddrID[0:]...)
		} else {
			n.Ver = append(n.Ver, params.TestNetParams.PubKeyHashAddrID[0:]...)
		}
	case "mixnet":
		if n.PK {
			n.Ver = append(n.Ver, params.MixNetParams.PubKeyAddrID[0:]...)
		} else {
			n.Ver = append(n.Ver, params.MixNetParams.PubKeyHashAddrID[0:]...)
		}
	default:
		v, err := hex.DecodeString(s)
		if err != nil {
			return err
		}
		n.Ver = append(n.Ver, v...)
	}
	n.flag = s
	return nil
}

func (n *QitmeerBase58checkVersionFlag) String() string {
	return n.flag
}

func (n *QitmeerBase58checkVersionFlag) Update() {
	n.Set(n.flag)
}
