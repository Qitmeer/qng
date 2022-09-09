package address

import (
	"errors"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/params"
	"golang.org/x/crypto/ripemd160"
)

// ErrInvalidPubKeyFormat indicates that a serialized pubkey is unusable as it
// is neither in the uncompressed or compressed format.
var ErrInvalidPubKeyFormat = errors.New("invalid pubkey format")

// ---------------------------------------------------------------------------
// SecpPubKeyAddress is an Address for a secp256k1 pay-to-pubkey transaction.
type SecpPubKeyAddress struct {
	net          *params.Params
	pubKeyFormat PubKeyFormat
	pubKey       ecc.PublicKey
	pubKeyHashID [2]byte
}

// EncodeAddress returns the string encoding of the public key as a
// pay-to-pubkey-hash.  Note that the public key format (uncompressed,
// compressed, etc) will change the resulting address.  This is expected since
// pay-to-pubkey-hash is a hash of the serialized public key which obviously
// differs with the format.
// Part of the Address interface.
func (a *SecpPubKeyAddress) Encode() string {
	return encodeAddress(hash.Hash160(a.serialize()), a.pubKeyHashID)
}

// String returns the hex-encoded human-readable string for the pay-to-pubkey
// address.  This is not the same as calling EncodeAddress.
func (a *SecpPubKeyAddress) String() string {
	return encodePKAddress(a.serialize(), a.net.PubKeyAddrID,
		ecc.ECDSA_Secp256k1)
}

// serialize returns the serialization of the public key according to the
// format associated with the address.
func (a *SecpPubKeyAddress) serialize() []byte {
	switch a.pubKeyFormat {
	default:
		fallthrough
	case PKFUncompressed:
		return a.pubKey.SerializeUncompressed()

	case PKFCompressed:
		return a.pubKey.SerializeCompressed()
	}
}

func (a *SecpPubKeyAddress) EcType() ecc.EcType {
	return ecc.ECDSA_Secp256k1
}

// Hash160 returns the underlying array of the pubkey hash.  This can be useful
// when an array is more appropriate than a slice (for example, when used as map
// keys).
func (a *SecpPubKeyAddress) Hash160() *[ripemd160.Size]byte {
	h160 := hash.Hash160(a.pubKey.SerializeCompressed())
	array := new([ripemd160.Size]byte)
	copy(array[:], h160)

	return array
}

// PubKey returns the underlying public key for the address.
func (a *SecpPubKeyAddress) PubKey() ecc.PublicKey {
	return a.pubKey
}

// PKHAddress returns the pay-to-pubkey address converted to a
// pay-to-pubkey-hash address.  Note that the public key format (uncompressed,
// compressed, etc) will change the resulting address.  This is expected since
// pay-to-pubkey-hash is a hash of the serialized public key which obviously
// differs with the format.
func (a *SecpPubKeyAddress) PKHAddress() *PubKeyHashAddress {
	return toPKHAddress(a.net, a.pubKeyHashID, a.serialize())
}

// Script returns the bytes to be included in a txout script to pay
// to a public key.  Setting the public key format will affect the output of
// this function accordingly.  Part of the Address interface.
func (a *SecpPubKeyAddress) Script() []byte {
	return a.serialize()
}

// NewAddressSecpPubKey returns a new AddressSecpPubKey which represents a
// pay-to-pubkey address, using a secp256k1 pubkey.  The serializedPubKey
// parameter must be a valid pubkey and must be uncompressed or compressed.
func NewSecpPubKeyAddress(serializedPubKey []byte,
	net *params.Params) (*SecpPubKeyAddress, error) {
	pubKey, err := ecc.Secp256k1.ParsePubKey(serializedPubKey)

	if err != nil {
		return nil, err
	}

	// Set the format of the pubkey.  This probably should be returned
	// from the crypto layer, but do it here to avoid API churn.  We already know the
	// pubkey is valid since it parsed above, so it's safe to simply examine
	// the leading byte to get the format.
	var pkFormat PubKeyFormat
	switch serializedPubKey[0] {
	case 0x02, 0x03:
		pkFormat = PKFCompressed
	case 0x04:
		pkFormat = PKFUncompressed
	default:
		return nil, ErrInvalidPubKeyFormat
	}

	return &SecpPubKeyAddress{
		net:          net,
		pubKeyFormat: pkFormat,
		pubKey:       pubKey,
		pubKeyHashID: net.PubKeyHashAddrID,
	}, nil
}

// ---------------------------------------------------------------------------
// EdwardsPubKeyAddress is an Address for an Ed25519 pay-to-pubkey transaction.
type EdwardsPubKeyAddress struct {
	net          *params.Params
	pubKey       ecc.PublicKey
	pubKeyHashID [2]byte
}

func (a *EdwardsPubKeyAddress) EcType() ecc.EcType {
	return ecc.EdDSA_Ed25519
}
func (a *EdwardsPubKeyAddress) Encode() string {
	return encodeAddress(hash.Hash160(a.serialize()), a.pubKeyHashID)
}

// String returns the hex-encoded human-readable string for the pay-to-pubkey
// address.  This is not the same as calling EncodeAddress.
func (a *EdwardsPubKeyAddress) String() string {
	return encodePKAddress(a.serialize(), a.net.PubKeyAddrID,
		ecc.EdDSA_Ed25519)
}

// serialize returns the serialization of the public key.
func (a *EdwardsPubKeyAddress) serialize() []byte {
	return a.pubKey.Serialize()
}

// Hash160 returns the underlying array of the pubkey hash.  This can be useful
// when an array is more appropriate than a slice (for example, when used as map
// keys).
func (a *EdwardsPubKeyAddress) Hash160() *[ripemd160.Size]byte {
	h160 := hash.Hash160(a.pubKey.Serialize())
	array := new([ripemd160.Size]byte)
	copy(array[:], h160)
	return array
}

func (a *EdwardsPubKeyAddress) PKHAddress() *PubKeyHashAddress {
	return toPKHAddress(a.net, a.pubKeyHashID, a.serialize())
}

// Script returns the bytes to be included in a txout script to pay
// to a public key.  Setting the public key format will affect the output of
// this function accordingly.  Part of the Address interface.
func (a *EdwardsPubKeyAddress) Script() []byte {
	return a.serialize()
}

// NewAddressEdwardsPubKey returns a new AddressEdwardsPubKey which represents a
// pay-to-pubkey address, using an Ed25519 pubkey.  The serializedPubKey
// parameter must be a valid 32 byte serialized public key.
func NewEdwardsPubKeyAddress(serializedPubKey []byte,
	net *params.Params) (*EdwardsPubKeyAddress, error) {
	pubKey, err := ecc.Ed25519.ParsePubKey(serializedPubKey)
	if err != nil {
		return nil, err
	}

	return &EdwardsPubKeyAddress{
		net:          net,
		pubKey:       pubKey,
		pubKeyHashID: net.PKHEdwardsAddrID,
	}, nil
}

// ---------------------------------------------------------------------------
// SecSchnorrPubKeyAddress is an Address for a secp256k1-schnorr pay-to-pubkey transaction.
type SecSchnorrPubKeyAddress struct {
	net          *params.Params
	pubKey       ecc.PublicKey
	pubKeyHashID [2]byte
}

func (a *SecSchnorrPubKeyAddress) EcType() ecc.EcType {
	return ecc.ECDSA_SecpSchnorr
}

func (a *SecSchnorrPubKeyAddress) Encode() string {
	return encodeAddress(hash.Hash160(a.serialize()), a.pubKeyHashID)
}

// String returns the hex-encoded human-readable string for the pay-to-pubkey
// address.  This is not the same as calling EncodeAddress.
func (a *SecSchnorrPubKeyAddress) String() string {
	return encodePKAddress(a.serialize(), a.net.PubKeyAddrID,
		ecc.ECDSA_SecpSchnorr)
}

func (a *SecSchnorrPubKeyAddress) serialize() []byte {
	return a.pubKey.Serialize()
}

// Hash160 returns the underlying array of the pubkey hash.  This can be useful
// when an array is more appropriate than a slice (for example, when used as map
// keys).
func (a *SecSchnorrPubKeyAddress) Hash160() *[ripemd160.Size]byte {
	h160 := hash.Hash160(a.pubKey.Serialize())
	array := new([ripemd160.Size]byte)
	copy(array[:], h160)
	return array
}

func (a *SecSchnorrPubKeyAddress) PKHAddress() *PubKeyHashAddress {
	return toPKHAddress(a.net, a.pubKeyHashID, a.serialize())
}

// Script returns the bytes to be included in a txout script to pay
// to a public key.  Setting the public key format will affect the output of
// this function accordingly.  Part of the Address interface.
func (a *SecSchnorrPubKeyAddress) Script() []byte {
	return a.serialize()
}

// NewAddressSecSchnorrPubKey returns a new AddressSecpPubKey which represents a
// pay-to-pubkey address, using a secp256k1 pubkey.  The serializedPubKey
// parameter must be a valid pubkey and must be compressed.
func NewSecSchnorrPubKeyAddress(serializedPubKey []byte,
	net *params.Params) (*SecSchnorrPubKeyAddress, error) {
	pubKey, err := ecc.SecSchnorr.ParsePubKey(serializedPubKey)
	if err != nil {
		return nil, err
	}
	return &SecSchnorrPubKeyAddress{
		net:          net,
		pubKey:       pubKey,
		pubKeyHashID: net.PKHSchnorrAddrID,
	}, nil
}

// ---------------------------------------------------------------------------
// NewAddressPubKey returns a new Address. decoded must
// be 33 bytes.
func NewPubKeyAddress(decoded []byte, net *params.Params) (types.Address, error) {
	if len(decoded) == 33 {
		// First byte is the signature suite and ybit.
		suite := decoded[0]
		suite &= ^uint8(1 << 7)
		ybit := !(decoded[0]&(1<<7) == 0)
		toAppend := uint8(0x02)
		if ybit {
			toAppend = 0x03
		}
		switch ecc.EcType(suite) {
		case ecc.ECDSA_Secp256k1:
			return NewSecpPubKeyAddress(
				append([]byte{toAppend}, decoded[1:]...),
				net)
		case ecc.EdDSA_Ed25519:
			return NewEdwardsPubKeyAddress(decoded, net)
		case ecc.ECDSA_SecpSchnorr:
			return NewSecSchnorrPubKeyAddress(
				append([]byte{toAppend}, decoded[1:]...),
				net)
		}
		return nil, ErrUnknownAddressType
	}
	return nil, ErrUnknownAddressType
}

// NewSecpPubKeyCompressedAddress creates a new address using a compressed public key
func NewSecpPubKeyCompressedAddress(pubkey ecc.PublicKey, params *params.Params) (*SecpPubKeyAddress, error) {
	return NewSecpPubKeyAddress(pubkey.SerializeCompressed(), params)
}
