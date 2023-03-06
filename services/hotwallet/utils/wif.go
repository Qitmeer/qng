package utils

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/qx"

	"github.com/Qitmeer/qng/common/encode/base58"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/crypto/ecc/secp256k1"
	"github.com/Qitmeer/qng/params"
)

// ErrMalformedPrivateKey describes an error where a WIF-encoded private
// key cannot be decoded due to being improperly formatted.  This may occur
// if the byte length is incorrect or an unexpected magic number was
// encountered.
var ErrMalformedPrivateKey = errors.New("malformed private key")

// ErrChecksumMismatch describes an error where decoding failed due
// to a bad checksum.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// compressMagic is the magic byte used to identify a WIF encoding for
// an address created from a compressed serialized public key.
const compressMagic byte = 0x01

// WIF contains the individual components described by the Wallet Import Format
// (WIF).  A WIF string is typically used to represent a private key and its
// associated address in a way that  may be easily copied and imported into or
// exported from wallet software.  WIF strings may be decoded into this
// structure by calling DecodeWIF or created with a user-provided private key
// by calling NewWIF.
type WIF struct {
	// PrivKey is the private key being imported or exported.
	PrivKey *secp256k1.PrivateKey

	// CompressPubKey specifies whether the address controlled by the
	// imported or exported private key was created by hashing a
	// compressed (33-byte) serialized public key, rather than an
	// uncompressed (65-byte) one.
	CompressPubKey bool

	// netID is the bitcoin network identifier byte used when
	// WIF encoding the private key.
	netID [2]byte
}

// NewWIF creates a new WIF structure to export an address and its private key
// as a string encoded in the Wallet Import Format.  The compress argument
// specifies whether the address intended to be imported or exported was created
// by serializing the public key compressed rather than uncompressed.
func NewWIF(privKey *secp256k1.PrivateKey, net *params.Params, compress bool) (*WIF, error) {
	if net == nil {
		return nil, errors.New("no network")
	}
	return &WIF{privKey, compress, net.PrivateKeyID}, nil
}

// IsForNet returns whether or not the decoded WIF structure is associated
// with the passed bitcoin network.
func (w *WIF) IsForNet(net *params.Params) bool {
	return w.netID == net.PrivateKeyID
}

// DecodeWIFV09 creates a new qitmeer-wallet V0.9 WIF structure by decoding the string encoding of
// the import format.
//
// The WIF string must be a base58-encoded string of the following byte
// sequence:
//
//  * 1 byte to identify the network, must be 0x80 for mainnet or 0xef for
//    either testnet3 or the regression test network
//  * 32 bytes of a binary-encoded, big-endian, zero-padded private key
//  * Optional 1 byte (equal to 0x01) if the address being imported or exported
//    was created by taking the RIPEMD160 after SHA256 hash of a serialized
//    compressed (33-byte) public key
//  * 4 bytes of checksum, must equal the first four bytes of the double SHA256
//    of every byte before the checksum in this sequence
//
// If the base58-decoded byte sequence does not match this, DecodeWIF will
// return a non-nil error.  ErrMalformedPrivateKey is returned when the WIF
// is of an impossible length or the expected compressed pubkey magic number
// does not equal the expected value of 0x01.  ErrChecksumMismatch is returned
// if the expected WIF checksum does not match the calculated checksum.
func DecodeWIFV09(wif string, net *params.Params) (*WIF, error) {
	decoded := base58.Decode([]byte(wif))
	if len(decoded) == 0 {
		return nil, ErrMalformedPrivateKey
	}
	decodedLen := len(decoded)
	var compress bool
	netID := [2]byte{decoded[0], decoded[1]}
	if netID != net.PrivateKeyID {
		return nil, fmt.Errorf("net is err ")
	}
	// Length of base58 decoded WIF must be 32 bytes + an optional 1 byte
	// (0x01) if compressed, plus 1 byte for netID + 4 bytes of checksum.
	switch decodedLen {
	case 2 + secp256k1.PrivKeyBytesLen + 1 + 4:
		if decoded[34] != compressMagic {
			return nil, ErrMalformedPrivateKey
		}
		compress = true
	case 2 + secp256k1.PrivKeyBytesLen + 4:
		compress = false
	default:
		return nil, ErrMalformedPrivateKey
	}
	// Checksum is first four bytes of double SHA256 of the identifier byte
	// and privKey.  Verify this matches the final 4 bytes of the decoded
	// private key.
	var tosum []byte
	if compress {
		tosum = decoded[:2+secp256k1.PrivKeyBytesLen+1]
	} else {
		tosum = decoded[:2+secp256k1.PrivKeyBytesLen]
	}
	cksum := hash.DoubleHashB(tosum)[:4]
	if !bytes.Equal(cksum, decoded[decodedLen-4:]) {
		return nil, ErrChecksumMismatch
	}
	privKeyBytes := decoded[2 : 2+secp256k1.PrivKeyBytesLen]
	privKey, _ := secp256k1.PrivKeyFromBytes(privKeyBytes)
	return &WIF{privKey, compress, netID}, nil
}

func DecodeWIF(wif string, net *params.Params) (*WIF, error) {
	bytes, compressed, err := qx.DecodeWIF(wif)
	if err != nil {
		return nil, err
	}
	priv, _ := secp256k1.PrivKeyFromBytes(bytes)
	return NewWIF(priv, net, compressed)
}

// StringV09 creates the Wallet Import Format string encoding of a qitmeer-wallet V0.9 WIF structure.
// See DecodeWIF for a detailed breakdown of the format and requirements of
// a valid WIF string.
func (w *WIF) StringV09() string {
	// Precalculate size.  Maximum number of bytes before base58 encoding
	// is one byte for the network, 32 bytes of private key, possibly one
	// extra byte if the pubkey is to be compressed, and finally four
	// bytes of checksum.
	encodeLen := 2 + secp256k1.PrivKeyBytesLen + 4
	if w.CompressPubKey {
		encodeLen++
	}

	a := make([]byte, 0, encodeLen)
	a = append(a, w.netID[0])
	a = append(a, w.netID[1])
	// Pad and append bytes manually, instead of using Serialize, to
	// avoid another call to make.
	a = paddedAppend(secp256k1.PrivKeyBytesLen, a, w.PrivKey.D.Bytes())
	if w.CompressPubKey {
		a = append(a, compressMagic)
	}
	cksum := hash.DoubleHashB(a)[:4]
	a = append(a, cksum...)

	ret, err := base58.Encode(a)
	if err != nil {
		return ""
	}
	return string(ret)
}

func (w *WIF) String() string {
	wif, _ := qx.EncodeWIF(w.CompressPubKey, hex.EncodeToString(w.PrivKey.Serialize()))
	return wif
}

// SerializePubKey serializes the associated public key of the imported or
// exported private key in either a compressed or uncompressed format.  The
// serialization format chosen depends on the value of w.CompressPubKey.
func (w *WIF) SerializePubKey() []byte {
	pk := (*secp256k1.PublicKey)(&w.PrivKey.PublicKey)
	if w.CompressPubKey {
		return pk.SerializeCompressed()
	}
	return pk.SerializeUncompressed()
}

// paddedAppend appends the src byte slice to dst, returning the new slice.
// If the length of the source is smaller than the passed size, leading zero
// bytes are appended to the dst slice before appending src.
func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
