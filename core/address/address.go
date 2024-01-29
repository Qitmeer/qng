// Copyright 2017-2018 The qitmeer developers

package address

import (
	"fmt"
	"github.com/Qitmeer/qng/common/encode/base58"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/params"
	"golang.org/x/crypto/ripemd160"
	"strings"
)

// encodeAddress returns a human-readable payment address given a ripemd160 hash
// and netID which encodes the network and address type.  It is used in both
// pay-to-pubkey-hash (P2PKH) and pay-to-script-hash (P2SH) address encoding.
func encodeAddress(hash160 []byte, netID [2]byte) string {
	// Format is 2 bytes for a network and address class (i.e. P2PKH vs
	// P2SH), 20 bytes for a RIPEMD160 hash, and 4 bytes of checksum.
	res, _ := base58.QitmeerCheckEncode(hash160[:ripemd160.Size], netID[:])
	return string(res)
}

// encodePKAddress returns a human-readable payment address to a public key
// given a serialized public key, a netID, and a signature suite.
func encodePKAddress(serializedPK []byte, netID [2]byte, algo ecc.EcType) string {
	pubKeyBytes := []byte{0x00}

	switch algo {
	case ecc.ECDSA_Secp256k1:
		pubKeyBytes[0] = byte(ecc.ECDSA_Secp256k1)
	case ecc.EdDSA_Ed25519:
		pubKeyBytes[0] = byte(ecc.EdDSA_Ed25519)
	case ecc.ECDSA_SecpSchnorr:
		pubKeyBytes[0] = byte(ecc.ECDSA_SecpSchnorr)
	}

	// Pubkeys are encoded as [0] = type/ybit, [1:33] = serialized pubkey
	compressed := serializedPK
	if algo == ecc.ECDSA_Secp256k1 || algo == ecc.ECDSA_SecpSchnorr {
		pub, err := ecc.Secp256k1.ParsePubKey(serializedPK)
		if err != nil {
			return ""
		}
		pubSerComp := pub.SerializeCompressed()

		// Set the y-bit if needed.
		if pubSerComp[0] == 0x03 {
			pubKeyBytes[0] |= (1 << 7)
		}

		compressed = pubSerComp[1:]
	}

	pubKeyBytes = append(pubKeyBytes, compressed...)
	res, _ := base58.QitmeerCheckEncode(pubKeyBytes, netID[:])
	return string(res)
}

// PubKeyFormat describes what format to use for a pay-to-pubkey address.
type PubKeyFormat int

const (
	// PKFUncompressed indicates the pay-to-pubkey address format is an
	// uncompressed public key.
	PKFUncompressed PubKeyFormat = iota

	// PKFCompressed indicates the pay-to-pubkey address format is a
	// compressed public key.
	PKFCompressed
)

func toPKHAddress(net *params.Params, netID [2]byte, b []byte) *PubKeyHashAddress {
	addr := &PubKeyHashAddress{net: net, netID: netID}
	copy(addr.hash[:], hash.Hash160(b))
	return addr
}

type ContractAddress struct {
	pk       []byte
	addrType types.AddressType
}

// DecodeAddress decodes the string encoding of an address and returns
// the Address if addr is a valid encoding for a known address type
func DecodeAddress(addr string) (types.Address, error) {
	oneIndex := strings.LastIndexByte(addr, '1')
	if oneIndex >= 1 {
		prefix := addr[:oneIndex+1]
		if params.IsBech32SegwitPrefix(prefix) {
			witnessVer, witnessProg, err := decodeSegWitAddress(addr)
			if err != nil {
				return nil, err
			}

			// We currently only support P2WPKH and P2WSH, which is
			// witness version 0 and P2TR which is witness version
			// 1.
			if witnessVer != 0 && witnessVer != 1 {
				return nil, fmt.Errorf("unsupported witness version: %#x", byte(witnessVer))
			}

			// The HRP is everything before the found '1'.
			hrp := prefix[:len(prefix)-1]

			switch len(witnessProg) {
			case 20:
				return newAddressWitnessPubKeyHash(hrp, witnessProg)
			case 32:
				if witnessVer == 1 {
					return newAddressTaproot(hrp, witnessProg)
				}

				return newAddressWitnessScriptHash(hrp, witnessProg)
			default:
				return nil, fmt.Errorf("unsupported witness program length: %d", len(witnessProg))
			}
		}
	}
	// Switch on decoded length to determine the type.
	decoded, netID, err := base58.QitmeerCheckDecode(addr)
	if err != nil {
		if err == base58.ErrChecksum {
			return nil, ErrChecksumMismatch
		}
		return nil, fmt.Errorf("decoded address is of unknown format: %v",
			err.Error())
	}

	// TODO, refactor the params design for address
	net, err := detectNetworkForAddress(addr)
	if err != nil {
		return nil, ErrUnknownAddressType
	}

	// TODO, refactor the params design for address
	switch netID {
	case net.PubKeyAddrID:
		return NewPubKeyAddress(decoded, net)

	case net.PubKeyHashAddrID:
		return NewPubKeyHashAddress(decoded, net, ecc.ECDSA_Secp256k1)

	case net.PKHEdwardsAddrID:
		return NewPubKeyHashAddress(decoded, net, ecc.EdDSA_Ed25519)

	case net.PKHSchnorrAddrID:
		return NewPubKeyHashAddress(decoded, net, ecc.ECDSA_SecpSchnorr)

	case net.ScriptHashAddrID:
		return NewScriptHashAddressFromHash(decoded, net)

	default:
		return nil, ErrUnknownAddressType
	}
}

// TODO, refactor the params design for address
// detectNetworkForAddress pops the first character from a string encoded
// address and detects what network type it is for.
func detectNetworkForAddress(addr string) (*params.Params, error) {
	if len(addr) < 1 {
		return nil, fmt.Errorf("empty string given for network detection")
	}

	networkChar := addr[0:1]
	switch networkChar {
	case params.MainNetParams.NetworkAddressPrefix:
		return &params.MainNetParams, nil
	case params.TestNetParams.NetworkAddressPrefix:
		return &params.TestNetParams, nil
	case params.PrivNetParams.NetworkAddressPrefix:
		return &params.PrivNetParams, nil
	case params.MixNetParam.NetworkAddressPrefix:
		return &params.MixNetParams, nil
	}

	return nil, fmt.Errorf("unknown network type in string encoded address")
}
