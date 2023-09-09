// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2016 The btcsuite developers
// Copyright (c) 2016-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package indexers implements optional block chain indexes.
package index

import (
	"encoding/binary"
	"errors"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
)

var (
	// byteOrder is the preferred byte order used for serializing numeric
	// fields for storage in the database.
	byteOrder = binary.LittleEndian

	// errInterruptRequested indicates that an operation was cancelled due
	// to a user-requested interrupt.
	errInterruptRequested = errors.New("interrupt requested")
)

// NeedsInputser provides a generic interface for an indexer to specify the it
// requires the ability to look up inputs for a transaction.
type NeedsInputser interface {
	NeedsInputs() bool
}

// Indexer provides a generic interface for an indexer that is managed by an
// index manager such as the Manager type provided by this package.
type Indexer interface {
	// Name returns the human-readable name of the index.
	Name() string

	// Init is invoked when the index manager is first initializing the
	// index.  This differs from the Create method in that it is called on
	// every load, including the case the index was just created.
	Init() error

	// ConnectBlock is invoked when the index manager is notified that a new
	// block has been connected to the main chain.
	ConnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error

	// DisconnectBlock is invoked when the index manager is notified that a
	// block has been disconnected from the main chain.
	DisconnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error
}
