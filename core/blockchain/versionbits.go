/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package blockchain

const (
	// VBTopBits defines the bits to set in the version to signal that the
	// version bits scheme is being used.
	VBTopBits = 0x20000000
)

func (b *BlockChain) CalcNextBlockVersion() (uint32, error) {
	return VBTopBits, nil
}
