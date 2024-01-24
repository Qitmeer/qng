package address

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/encode/bech32"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/params"
	"strings"
)

// encodeSegWitAddress creates a bech32 (or bech32m for SegWit v1) encoded
// address string representation from witness version and witness program.
func encodeSegWitAddress(hrp string, witnessVersion byte, witnessProgram []byte) (string, error) {
	// Group the address bytes into 5 bit groups, as this is what is used to
	// encode each character in the address string.
	converted, err := bech32.ConvertBits(witnessProgram, 8, 5, true)
	if err != nil {
		return "", err
	}

	// Concatenate the witness version and program, and encode the resulting
	// bytes using bech32 encoding.
	combined := make([]byte, len(converted)+1)
	combined[0] = witnessVersion
	copy(combined[1:], converted)

	var bech string
	switch witnessVersion {
	case 0:
		bech, err = bech32.Encode(hrp, combined)

	case 1:
		bech, err = bech32.EncodeM(hrp, combined)

	default:
		return "", fmt.Errorf("unsupported witness version %d",
			witnessVersion)
	}
	if err != nil {
		return "", err
	}

	// Check validity by decoding the created address.
	version, program, err := decodeSegWitAddress(bech)
	if err != nil {
		return "", fmt.Errorf("invalid segwit address: %v", err)
	}

	if version != witnessVersion || !bytes.Equal(program, witnessProgram) {
		return "", fmt.Errorf("invalid segwit address")
	}

	return bech, nil
}

// decodeSegWitAddress parses a bech32 encoded segwit address string and
// returns the witness version and witness program byte representation.
func decodeSegWitAddress(address string) (byte, []byte, error) {
	// Decode the bech32 encoded address.
	_, data, bech32version, err := bech32.DecodeGeneric(address)
	if err != nil {
		return 0, nil, err
	}

	// The first byte of the decoded address is the witness version, it must
	// exist.
	if len(data) < 1 {
		return 0, nil, fmt.Errorf("no witness version")
	}

	// ...and be <= 16.
	version := data[0]
	if version > 16 {
		return 0, nil, fmt.Errorf("invalid witness version: %v", version)
	}

	// The remaining characters of the address returned are grouped into
	// words of 5 bits. In order to restore the original witness program
	// bytes, we'll need to regroup into 8 bit words.
	regrouped, err := bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return 0, nil, err
	}

	// The regrouped data must be between 2 and 40 bytes.
	if len(regrouped) < 2 || len(regrouped) > 40 {
		return 0, nil, fmt.Errorf("invalid data length")
	}

	// For witness version 0, address MUST be exactly 20 or 32 bytes.
	if version == 0 && len(regrouped) != 20 && len(regrouped) != 32 {
		return 0, nil, fmt.Errorf("invalid data length for witness "+
			"version 0: %v", len(regrouped))
	}

	// For witness version 0, the bech32 encoding must be used.
	if version == 0 && bech32version != bech32.Version0 {
		return 0, nil, fmt.Errorf("invalid checksum expected bech32 " +
			"encoding for address with witness version 0")
	}

	// For witness version 1, the bech32m encoding must be used.
	if version == 1 && bech32version != bech32.VersionM {
		return 0, nil, fmt.Errorf("invalid checksum expected bech32m " +
			"encoding for address with witness version 1")
	}

	return version, regrouped, nil
}

// AddressSegWit is the base address type for all SegWit addresses.
type AddressSegWit struct {
	hrp            string
	witnessVersion byte
	witnessProgram []byte
}

// EncodeAddress returns the bech32 (or bech32m for SegWit v1) string encoding
// of an AddressSegWit.
//
// NOTE: This method is part of the Address interface.
func (a *AddressSegWit) Encode() string {
	str, err := encodeSegWitAddress(
		a.hrp, a.witnessVersion, a.witnessProgram[:],
	)
	if err != nil {
		return ""
	}
	return str
}

// ScriptAddress returns the witness program for this address.
//
// NOTE: This method is part of the Address interface.
func (a *AddressSegWit) Script() []byte {
	return a.witnessProgram[:]
}

// IsForNet returns whether the AddressSegWit is associated with the passed
//
// NOTE: This method is part of the Address interface.
func (a *AddressSegWit) IsForNet(net *params.Params) bool {
	return a.hrp == net.Bech32HRPSegwit
}

// IsForNetwork returns whether or not the address is associated with the
// passed network.
func (a *AddressSegWit) IsForNetwork(net protocol.Network) bool {
	switch net {
	case params.MainNetParams.Net:
		return a.IsForNet(&params.MainNetParams)
	case params.PrivNetParams.Net:
		return a.IsForNet(&params.PrivNetParams)
	case params.TestNetParams.Net:
		return a.IsForNet(&params.TestNetParams)
	case params.MixNetParams.Net:
		return a.IsForNet(&params.MixNetParams)
	}
	return false
}

// String returns a human-readable string for the AddressWitnessPubKeyHash.
// This is equivalent to calling EncodeAddress, but is provided so the type
// can be used as a fmt.Stringer.
//
// NOTE: This method is part of the Address interface.
func (a *AddressSegWit) String() string {
	return a.Encode()
}

// Hrp returns the human-readable part of the bech32 (or bech32m for SegWit v1)
// encoded AddressSegWit.
func (a *AddressSegWit) Hrp() string {
	return a.hrp
}

// WitnessVersion returns the witness version of the AddressSegWit.
func (a *AddressSegWit) WitnessVersion() byte {
	return a.witnessVersion
}

// WitnessProgram returns the witness program of the AddressSegWit.
func (a *AddressSegWit) WitnessProgram() []byte {
	return a.witnessProgram[:]
}

func (a *AddressSegWit) Hash160() *[20]byte {
	return nil
}

func (a *AddressSegWit) EcType() ecc.EcType {
	return ecc.ECDSA_Secp256k1
}

// AddressWitnessPubKeyHash is an Address for a pay-to-witness-pubkey-hash
// (P2WPKH) output.
type AddressWitnessPubKeyHash struct {
	AddressSegWit
}

// NewAddressWitnessPubKeyHash returns a new AddressWitnessPubKeyHash.
func NewAddressWitnessPubKeyHash(witnessProg []byte, net *params.Params) (*AddressWitnessPubKeyHash, error) {
	return newAddressWitnessPubKeyHash(net.Bech32HRPSegwit, witnessProg)
}

// newAddressWitnessPubKeyHash is an internal helper function to create an
// AddressWitnessPubKeyHash with a known human-readable part, rather than
// looking it up through its parameters.
func newAddressWitnessPubKeyHash(hrp string,
	witnessProg []byte) (*AddressWitnessPubKeyHash, error) {

	// Check for valid program length for witness version 0, which is 20
	// for P2WPKH.
	if len(witnessProg) != 20 {
		return nil, errors.New("witness program must be 20 " +
			"bytes for p2wpkh")
	}

	addr := &AddressWitnessPubKeyHash{
		AddressSegWit{
			hrp:            strings.ToLower(hrp),
			witnessVersion: 0x00,
			witnessProgram: witnessProg,
		},
	}

	return addr, nil
}

// Hash160 returns the witness program of the AddressWitnessPubKeyHash as a
// byte array.
func (a *AddressWitnessPubKeyHash) Hash160() *[20]byte {
	var pubKeyHashWitnessProgram [20]byte
	copy(pubKeyHashWitnessProgram[:], a.witnessProgram)
	return &pubKeyHashWitnessProgram
}

// AddressWitnessScriptHash is an Address for a pay-to-witness-script-hash
// (P2WSH) output.
type AddressWitnessScriptHash struct {
	AddressSegWit
}

// NewAddressWitnessScriptHash returns a new AddressWitnessPubKeyHash.
func NewAddressWitnessScriptHash(witnessProg []byte,
	net *params.Params) (*AddressWitnessScriptHash, error) {

	return newAddressWitnessScriptHash(net.Bech32HRPSegwit, witnessProg)
}

// newAddressWitnessScriptHash is an internal helper function to create an
// AddressWitnessScriptHash with a known human-readable part, rather than
// looking it up through its parameters.
func newAddressWitnessScriptHash(hrp string,
	witnessProg []byte) (*AddressWitnessScriptHash, error) {

	// Check for valid program length for witness version 0, which is 32
	// for P2WSH.
	if len(witnessProg) != 32 {
		return nil, errors.New("witness program must be 32 " +
			"bytes for p2wsh")
	}

	addr := &AddressWitnessScriptHash{
		AddressSegWit{
			hrp:            strings.ToLower(hrp),
			witnessVersion: 0x00,
			witnessProgram: witnessProg,
		},
	}

	return addr, nil
}

// AddressTaproot is an Address for a pay-to-taproot (P2TR) output. See BIP 341
// for further details.
type AddressTaproot struct {
	AddressSegWit
}

// NewAddressTaproot returns a new AddressTaproot.
func NewAddressTaproot(witnessProg []byte,
	net *params.Params) (*AddressTaproot, error) {

	return newAddressTaproot(net.Bech32HRPSegwit, witnessProg)
}

// newAddressWitnessScriptHash is an internal helper function to create an
// AddressWitnessScriptHash with a known human-readable part, rather than
// looking it up through its parameters.
func newAddressTaproot(hrp string,
	witnessProg []byte) (*AddressTaproot, error) {

	// Check for valid program length for witness version 1, which is 32
	// for P2TR.
	if len(witnessProg) != 32 {
		return nil, errors.New("witness program must be 32 bytes for " +
			"p2tr")
	}

	addr := &AddressTaproot{
		AddressSegWit{
			hrp:            strings.ToLower(hrp),
			witnessVersion: 0x01,
			witnessProgram: witnessProg,
		},
	}

	return addr, nil
}
