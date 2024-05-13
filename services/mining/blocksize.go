// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2016-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import "errors"

var ErrorMinBlockSize = errors.New("Block size is to small")

func (p *Policy) SetBlockMaxSize(size uint32) error {
	if size < p.BlockMinSize {
		return ErrorMinBlockSize
	}
	p.BlockMaxSize = size
	return nil
}
