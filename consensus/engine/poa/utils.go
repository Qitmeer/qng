package poa

import (
	"bytes"
	"github.com/Qitmeer/qng/v2/common/hash"
	ptypes "github.com/Qitmeer/qng/v2/consensus/engine/poa/types"
	"github.com/Qitmeer/qng/v2/core/coinbase"
	s "github.com/Qitmeer/qng/v2/core/serialization"
	qtypes "github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
	"io"
)

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *qtypes.BlockHeader, sigcache *sigLRU) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.BlockHash()
	if address, known := sigcache.Get(hash); known {
		return address, nil
	}
	poa, ok := header.Engine.(*ptypes.PoA)
	if !ok {
		return common.Address{}, errMissingSignature
	}
	// Retrieve the signature from the header extra-data
	if len(poa.Seal) < ptypes.SealSize {
		return common.Address{}, errMissingSignature
	}
	signature := poa.Seal

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(SealHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *qtypes.BlockHeader) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.(crypto.KeccakState).Read(hash[:])
	return hash
}

// DagPoARLP returns the rlp bytes which needs to be signed for the proof-of-authority
// sealing. The RLP to sign consists of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func DagPoARLP(header *qtypes.BlockHeader) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header)
	return b.Bytes()
}

func encodeSigHeader(w io.Writer, header *qtypes.BlockHeader) error {
	bh := header
	sec := uint32(bh.Timestamp.Unix())
	return s.WriteElements(w, bh.Version, &bh.ParentRoot, &bh.TxRoot,
		&bh.StateRoot, bh.Difficulty, sec, bh.Engine.(*ptypes.PoA).SealBytes())
}

type Block struct {
	block  *qtypes.SerializedBlock
	height uint64
}

func NewBlock(block *qtypes.SerializedBlock) (*Block, error) {
	height := uint64(0)
	if !block.Hash().IsEqual(params.ActiveNetParams.GenesisHash) {
		h, err := coinbase.ExtractCoinbaseHeight(block.Transactions()[0].Tx)
		if err != nil {
			return nil, err
		}
		height = h
	}

	return &Block{
		block:  block,
		height: height,
	}, nil
}

func (b *Block) parent() *hash.Hash {
	return b.block.Block().Parents[0]
}

func (b *Block) Engine() *ptypes.PoA {
	return b.block.Block().Header.Engine.(*ptypes.PoA)
}

func (b *Block) Coinbase() common.Address {
	return b.Engine().Beneficiary
}

func (b *Block) Header() *qtypes.BlockHeader {
	return &b.block.Block().Header
}
