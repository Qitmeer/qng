// Copyright 2017-2018 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package qx

import (
	"github.com/Qitmeer/qng/services/wallet/hd"
)

type DerivePathFlag struct {
	Path hd.DerivationPath
}

func (d *DerivePathFlag) Set(s string) error {
	path, err := hd.ParseDerivationPath(s)
	if err != nil {
		return err
	}
	d.Path = path
	return nil
}

func (d *DerivePathFlag) String() string {
	return d.Path.String()
}
