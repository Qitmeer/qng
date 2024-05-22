package address

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"reflect"
	"strings"
	"testing"
)

func TestAddresses(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		encoded string
		valid   bool
		result  types.Address
		f       func() (types.Address, error)
		net     *params.Params
	}{
		// P2TR address tests.
		{
			name:    "segwit v1 mainnet p2tr",
			addr:    "m1paardr2nczq0rx5rqpfwnvpzm497zvux64y0f7wjgcs7xuuuh2nnqze6df7",
			encoded: "m1paardr2nczq0rx5rqpfwnvpzm497zvux64y0f7wjgcs7xuuuh2nnqze6df7",
			valid:   true,
			result: TstAddressTaproot(
				1, [32]byte{
					0xef, 0x46, 0xd1, 0xaa, 0x78, 0x10, 0x1e, 0x33,
					0x50, 0x60, 0x0a, 0x5d, 0x36, 0x04, 0x5b, 0xa9,
					0x7c, 0x26, 0x70, 0xda, 0xa9, 0x1e, 0x9f, 0x3a,
					0x48, 0xc4, 0x3c, 0x6e, 0x73, 0x97, 0x54, 0xe6,
				}, params.MainNetParams.Bech32HRPSegwit,
			),
			f: func() (types.Address, error) {
				scriptHash := []byte{
					0xef, 0x46, 0xd1, 0xaa, 0x78, 0x10, 0x1e, 0x33,
					0x50, 0x60, 0x0a, 0x5d, 0x36, 0x04, 0x5b, 0xa9,
					0x7c, 0x26, 0x70, 0xda, 0xa9, 0x1e, 0x9f, 0x3a,
					0x48, 0xc4, 0x3c, 0x6e, 0x73, 0x97, 0x54, 0xe6,
				}
				return NewAddressTaproot(
					scriptHash, &params.MainNetParams,
				)
			},
			net: &params.MainNetParams,
		},

		// Invalid bech32m tests. Source:
		{
			name:  "segwit v1 invalid human-readable part",
			addr:  "m1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vq5zuyut",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit v1 mainnet bech32 instead of bech32m",
			addr:  "m1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqh2y7hd",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit v1 mainnet bech32 instead of bech32m upper case",
			addr:  "m1S0XLXVLHEMJA6C4DQV22UAPCTQUPFHLXM9H8Z3K2E72Q4K9HCZ7VQ54WELL",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit v1 mainnet bech32m invalid character in checksum",
			addr:  "m1p38j9r5y49hruaue7wxjce0updqjuyyx0kh56v8s25huc6995vvpql3jow4",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit mainnet witness v17",
			addr:  "m130XLXVLHEMJA6C4DQV22UAPCTQUPFHLXM9H8Z3K2E72Q4K9HCZ7VQ7ZWS8R",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit v1 mainnet bech32m invalid program length (1 byte)",
			addr:  "m1pw5dgrnzv",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit v1 mainnet bech32m invalid program length (41 bytes)",
			addr:  "m1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7v8n0nx0muaewav253zgeav",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit v1 mainnet bech32m zero padding of more than 4 bits",
			addr:  "m1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7v07qwwzcrf",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit v1 mainnet bech32m empty data section",
			addr:  "m1gmk9yu",
			valid: false,
			net:   &params.MainNetParams,
		},

		// Unsupported witness versions (version 0 and 1 only supported at this point)
		{
			name:  "segwit mainnet witness v16",
			addr:  "m1SW50QA3JX3S",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit mainnet witness v2",
			addr:  "m1zw508d6qejxtdg4y5r3zarvaryvg6kdaj",
			valid: false,
			net:   &params.MainNetParams,
		},
		// Invalid segwit addresses
		{
			name:  "segwit invalid checksum",
			addr:  "m1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t5",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit invalid witness version",
			addr:  "m13W508D6QEJXTDG4Y5R3ZARVARY0C5XW7KN40WF2",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit invalid program length",
			addr:  "m1rw5uspcuh",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit invalid program length",
			addr:  "m10w508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7kw5rljs90",
			valid: false,
			net:   &params.MainNetParams,
		},
		{
			name:  "segwit invalid program length for witness version 0 (per BIP141)",
			addr:  "m1QR508D6QEJXTDG4Y5R3ZARVARYV98GJ9P",
			valid: false,
			net:   &params.MainNetParams,
		},
	}

	for _, test := range tests {
		// Decode addr and compare error against valid.
		decoded, err := DecodeAddress(test.addr)
		if (err == nil) != test.valid {
			t.Errorf("%v: decoding test failed: %v", test.name, err)
			return
		}

		if err == nil {
			// Ensure the stringer returns the same address as the
			// original.
			if decodedStringer, ok := decoded.(fmt.Stringer); ok {
				addr := test.addr

				// For Segwit addresses the string representation
				// will always be lower case, so in that case we
				// convert the original to lower case first.
				if strings.Contains(test.name, "segwit") {
					addr = strings.ToLower(addr)
				}

				if addr != decodedStringer.String() {
					t.Errorf("%v: String on decoded value does not match expected value: %v != %v",
						test.name, test.addr, decodedStringer.String())
					return
				}
			}

			// Encode again and compare against the original.
			encoded := decoded.Encode()
			if test.encoded != encoded {
				t.Errorf("%v: decoding and encoding produced different addresses: %v != %v",
					test.name, test.encoded, encoded)
				return
			}

			// Perform type-specific calculations.
			var saddr []byte
			if _, ok := decoded.(*AddressTaproot); ok {
				saddr = TstAddressTaprootSAddr(encoded)
			}
			// Check script address, as well as the Hash160 method for P2PKH and
			// P2SH addresses.
			if !bytes.Equal(saddr, decoded.Script()) {
				t.Errorf("%v: script addresses do not match:\n%x != \n%x",
					test.name, saddr, decoded.Script())
				return
			}
			// Ensure the address is for the expected network.
			if !decoded.IsForNetwork(test.net.Net) {
				t.Errorf("%v: calculated network does not match expected",
					test.name)
				return
			}
		} else {
			// If there is an error, make sure we can print it
			// correctly.
			errStr := err.Error()
			if errStr == "" {
				t.Errorf("%v: error was non-nil but message is"+
					"empty: %v", test.name, err)
			}
		}

		if !test.valid {
			// If address is invalid, but a creation function exists,
			// verify that it returns a nil addr and non-nil error.
			if test.f != nil {
				_, err := test.f()
				if err == nil {
					t.Errorf("%v: address is invalid but creating new address succeeded",
						test.name)
					return
				}
			}
			continue
		}

		// Valid test, compare address created with f against expected result.
		addr, err := test.f()
		if err != nil {
			t.Errorf("%v: address is valid but creating new address failed with error %v",
				test.name, err)
			return
		}

		if !reflect.DeepEqual(addr, test.result) {
			t.Errorf("%v: created address does not match expected result",
				test.name)
			return
		}
	}
}
