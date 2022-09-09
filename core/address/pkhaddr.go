package address

import (
	"errors"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/params"
	"golang.org/x/crypto/ripemd160"
)

// PubKeyHashAddress is an Address for a pay-to-pubkey-hash (P2PKH)
// transaction.
type PubKeyHashAddress struct {
	net   *params.Params
	netID [2]byte
	hash  [ripemd160.Size]byte
}

// EcType returns the digital signature algorithm for the associated public key
// hash.
func (a *PubKeyHashAddress) EcType() ecc.EcType {
	switch a.netID {
	case a.net.PubKeyHashAddrID:
		return ecc.ECDSA_Secp256k1
	case a.net.PKHEdwardsAddrID:
		return ecc.EdDSA_Ed25519
	case a.net.PKHSchnorrAddrID:
		return ecc.ECDSA_SecpSchnorr
	}
	return -1
}
func (a *PubKeyHashAddress) Encode() string {
	//TODO error handling
	return encodeAddress(a.hash[:], a.netID)
}

// String returns a human-readable string for the pay-to-pubkey-hash address.
// This is equivalent to calling EncodeAddress, but is provided so the type can
// be used as a fmt.Stringer.
func (a *PubKeyHashAddress) String() string {
	return a.Encode()
}

func (a *PubKeyHashAddress) Hash160() *[ripemd160.Size]byte {
	return &a.hash
}

// Script returns the bytes to be included in a txout script to pay
// to a pubkey hash.  Part of the Address interface.
func (a *PubKeyHashAddress) Script() []byte {
	return a.hash[:]
}

// IsForNetwork returns whether or not the address is associated with the
// passed network.
func (a *PubKeyHashAddress) IsForNetwork(net protocol.Network) bool {
	return a.net.Net == net
}

// NewAddressPubKeyHash returns a new AddressPubKeyHash.  pkHash must
// be 20 bytes.
func NewPubKeyHashAddress(pkHash []byte, net *params.Params, algo ecc.EcType) (*PubKeyHashAddress, error) {
	var addrID [2]byte
	switch algo {
	case ecc.ECDSA_Secp256k1:
		addrID = net.PubKeyHashAddrID
	case ecc.EdDSA_Ed25519:
		addrID = net.PKHEdwardsAddrID
	case ecc.ECDSA_SecpSchnorr:
		addrID = net.PKHSchnorrAddrID
	default:
		return nil, errors.New("unknown ECDSA algorithm")
	}
	apkh, err := newPubKeyHashAddress(pkHash, addrID)
	if err != nil {
		return nil, err
	}
	apkh.net = net
	return apkh, nil
}

// newPubKeyHashAddress is the internal API to create a pubkey hash address
// with a known leading identifier byte for a network, rather than looking
// it up through its parameters.  This is useful when creating a new address
// structure from a string encoding where the identifer byte is already
// known.
func newPubKeyHashAddress(pkHash []byte, netID [2]byte) (*PubKeyHashAddress,
	error) {
	// Check for a valid pubkey hash length.
	if len(pkHash) != ripemd160.Size {
		return nil, errors.New("pkHash must be 20 bytes")
	}
	addr := &PubKeyHashAddress{netID: netID}
	copy(addr.hash[:], pkHash)
	return addr, nil
}
