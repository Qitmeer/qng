package address

import (
	"errors"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/params"
	"golang.org/x/crypto/ripemd160"
)

// ScriptHashAddress is an Address for a pay-to-script-hash (P2SH)
// transaction.
type ScriptHashAddress struct {
	net   *params.Params
	hash  [ripemd160.Size]byte
	netID [2]byte
}

// Hash160 returns the underlying array of the script hash.  This can be useful
// when an array is more appropriate than a slice (for example, when used as map
// keys).
func (a *ScriptHashAddress) Hash160() *[ripemd160.Size]byte {
	return &a.hash
}

// EncodeAddress returns the string encoding of a pay-to-script-hash
// address.  Part of the Address interface.
func (a *ScriptHashAddress) Encode() string {
	return encodeAddress(a.hash[:], a.netID)
}

func (a *ScriptHashAddress) String() string {
	return a.Encode()
}

func (a *ScriptHashAddress) EcType() ecc.EcType {
	return ecc.ECDSA_Secp256k1
}

// Script returns the bytes to be included in a txout script to pay
// to a script hash.  Part of the Address interface.
func (a *ScriptHashAddress) Script() []byte {
	return a.hash[:]
}

// IsForNetwork returns whether or not the address is associated with the
// passed network.
func (a *ScriptHashAddress) IsForNetwork(net protocol.Network) bool {
	return a.net.Net == net
}

// newAddressScriptHashFromHash is the internal API to create a script hash
// address with a known leading identifier byte for a network, rather than
// looking it up through its parameters.  This is useful when creating a new
// address structure from a string encoding where the identifer byte is already
// known.
func newScriptHashAddressFromHash(scriptHash []byte, netID [2]byte) (*ScriptHashAddress, error) {
	// Check for a valid script hash length.
	if len(scriptHash) != ripemd160.Size {
		return nil, errors.New("scriptHash must be 20 bytes")
	}

	addr := &ScriptHashAddress{netID: netID}
	copy(addr.hash[:], scriptHash)
	return addr, nil
}

// NewAddressScriptHash returns a new AddressScriptHash.
func NewScriptHashAddress(serializedScript []byte, net *params.Params) (*ScriptHashAddress, error) {
	scriptHash := hash.Hash160(serializedScript)
	sha, err := newScriptHashAddressFromHash(scriptHash, net.ScriptHashAddrID)
	if err != nil {
		return nil, err
	}
	sha.net = net
	return sha, nil
}

// NewAddressScriptHashFromHash returns a new AddressScriptHash.  scriptHash
// must be 20 bytes.
func NewScriptHashAddressFromHash(scriptHash []byte, net *params.Params) (*ScriptHashAddress, error) {
	ash, err := newScriptHashAddressFromHash(scriptHash, net.ScriptHashAddrID)
	if err != nil {
		return nil, err
	}
	ash.net = net

	return ash, nil
}
