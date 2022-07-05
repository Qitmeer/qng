# qx

qx is a command-line tool that provides a variety of commands for key management and transaction construction, such as random "seed" generation, public/private key encoding etc. qx cab be built and distributed as a single file binary, which works like the swiss army knife of qitmeer


## Installation

### How to build

```shell
~ go build
~ ./qx
```

## Usage

```
$ ./qx
Usage: qx [--version] [--help] <command> [<args>]

encode and decode :
    base58-encode         encode a base16 string to a base58 string
    base58-decode         decode a base58 string to a base16 string
    base58check-encode    encode a base58check string
    base58check-decode    decode a base58check string
    base64-encode         encode a base16 string to a base64 string
    base64-decode         decode a base64 string to a base16 string
    rlp-encode            encode a string to a rlp encoded base16 string
    rlp-decode            decode a rlp base16 string to a human-readble representation

hash :
    blake2b256            calculate Blake2b 256 hash of a base16 data.
    blake2b512            calculate Blake2b 512 hash of a base16 data.
    sha256                calculate SHA256 hash of a base16 data.
    sha3-256              calculate SHA3 256 hash of a base16 data.
    keccak-256            calculate legacy keccak 256 hash of a bash16 data.
    blake256              calculate blake256 hash of a base16 data.
    ripemd160             calculate ripemd160 hash of a base16 data.
    bitcion160            calculate ripemd160(sha256(data))
    hash160               calculate ripemd160(blake2b256(data))

difficulty :
    compact-to-uint64     convert cuckoo compact difficulty to uint64.
    uint64-to-compact     convert cuckoo uint64 difficulty to compact.
    diff-to-gps           convert cuckoo compact difficulty to GPS.
    gps-to-diff           convert cuckoo GPS to uint64 difficulty.

entropy (seed) & mnemoic & hd & ec
    entropy               generate a cryptographically secure pseudorandom entropy (seed)
    hd-new                create a new HD(BIP32) private key from an entropy (seed)
    hd-to-ec              convert the HD (BIP32) format private/public key to a EC private/public key
    hd-to-public          derive the HD (BIP32) public key from a HD private key
    hd-decode             decode a HD (BIP32) private/public key serialization format
    hd-derive             Derive a child HD (BIP32) key from another HD public or private key.
    mnemonic-new          create a mnemonic world-list (BIP39) from an entropy
    mnemonic-to-entropy   return back to the entropy (the random seed) from a mnemonic world list (BIP39)
    mnemonic-to-seed      convert a mnemonic world-list (BIP39) to its 512 bits seed
    ec-new                create a new EC private key from an entropy (seed).
    ec-to-public          derive the EC public key from an EC private key (the compressed format by default )
    ec-to-wif             convert an EC private key to a WIF, associates with the compressed public key by default.
    wif-to-ec             convert a WIF private key to an EC private key.
    wif-to-public         derive the EC public key from a WIF private key.

addr & tx & sign
    ec-to-addr            convert an EC public key to a paymant address. default is qitmeer address
    tx-encode             encode a unsigned transaction.
    tx-decode             decode a transaction in base16 to json format.
    tx-sign               sign a transactions using a private key.
    msg-sign              create a message signature
    msg-verify            validate a message signature
    signature-decode      decode a ECDSA signature
```
## Examples
### TxTypeRegular (Regular MEER Tx)
```bash
$ ./qx tx-encode -v 1 -i 5fdad6bb6781416b0361a10eb6183dec45fb31edcf2da10d22893ee7bb6502ca:0:4294967295:TxTypeRegular -l 0 -o XmRTajVTajFiaEkd7PygFw46vNsoNW6fWE5:9.9999:0:TxTypeRegular

output:// 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010000f0a29a3b000000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac000000000000000004b3c3620100-7b22696e707574223a7b2230223a307d2c226f7574707574223a7b2230223a307d7d
$ ./qx tx-decode 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010000f0a29a3b000000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac000000000000000004b3c3620100-7b22696e707574223a7b2230223a307d2c226f7574707574223a7b2230223a307d7d

```
```json
{
	"txid": "ccc47713f2231b3f3923d9cb8cc309ad9a460982a83403a34ec2153e42a19e70",
	"txhash": "2abf3815995363b5010a29cceff1bf54c344f11dc52bfb6d012e4a20a01b27d3",
	"version": 1,
	"locktime": 0,
	"expire": 0,
	"vin": [{
		"type": "TxTypeRegular",
		"scriptSig": {
			"asm": "",
			"hex": ""
		}
	}],
	"vout": [{
		"coin": "MEER",
		"amount": 999990000,
		"scriptPubKey": {
			"asm": "OP_DUP OP_HASH160 ada117669b04771e481cc68ae8d4f33f913d1eda OP_EQUALVERIFY OP_CHECKSIG",
			"hex": "76a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac",
			"reqSigs": 1,
			"type": "pubkeyhash",
			"addresses": ["MmbRPt1wRcHz1mAiH8Ri1ANtGRyY5ygJ3hJ"]
		}
	}]
}
```
```bash
$ ./qx sign -k (privateKey) -n mixnet 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010000f0a29a3b000000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac000000000000000004b3c3620100-7b22696e707574223a7b2230223a307d2c226f7574707574223a7b2230223a307d7d

output:// 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010000f0a29a3b000000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac000000000000000004b3c362016a4730440220216d2d8e2ab92b3e94e9decfbf0a235df213b628fdf040f746b01988fac23deb02202cc4d7bcee0147ba08a0ce5636a23c3a56cfd03de41f8d16e54894e982ead61b0121039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ff

./qx tx-decode 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010000f0a29a3b000000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac000000000000000004b3c362016a4730440220216d2d8e2ab92b3e94e9decfbf0a235df213b628fdf040f746b01988fac23deb02202cc4d7bcee0147ba08a0ce5636a23c3a56cfd03de41f8d16e54894e982ead61b0121039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ff
```
```json
{
	"txid": "ccc47713f2231b3f3923d9cb8cc309ad9a460982a83403a34ec2153e42a19e70",
	"txhash": "7f164a6d0b15cba70792de063740b8408f5ce57ed0a3c903971ebb0203075b03",
	"version": 1,
	"locktime": 0,
	"expire": 0,
	"vin": [{
		"txid": "5fdad6bb6781416b0361a10eb6183dec45fb31edcf2da10d22893ee7bb6502ca",
		"vout": 0,
		"sequence": 4294967295,
		"scriptSig": {
			"asm": "30440220216d2d8e2ab92b3e94e9decfbf0a235df213b628fdf040f746b01988fac23deb02202cc4d7bcee0147ba08a0ce5636a23c3a56cfd03de41f8d16e54894e982ead61b01 039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ff",
			"hex": "4730440220216d2d8e2ab92b3e94e9decfbf0a235df213b628fdf040f746b01988fac23deb02202cc4d7bcee0147ba08a0ce5636a23c3a56cfd03de41f8d16e54894e982ead61b0121039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ff"
		}
	}],
	"vout": [{
		"coin": "MEER",
		"amount": 999990000,
		"scriptPubKey": {
			"asm": "OP_DUP OP_HASH160 ada117669b04771e481cc68ae8d4f33f913d1eda OP_EQUALVERIFY OP_CHECKSIG",
			"hex": "76a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac",
			"reqSigs": 1,
			"type": "pubkeyhash",
			"addresses": ["MmbRPt1wRcHz1mAiH8Ri1ANtGRyY5ygJ3hJ"]
		}
	}]
}
```

### TxTypeCrossChainExport（TX from MEER to EVM)
```bash
$ ./qx tx-encode -v 1 -i 5fdad6bb6781416b0361a10eb6183dec45fb31edcf2da10d22893ee7bb6502ca:0:4294967295:TxTypeCrossChainExport -l 0 -o XkCfdHoHHe2raZwNoY4sKcXFf6Jy9Q8XotAHenYsucPrEoj1FeUTR:9.9999:1:TxTypeCrossChainExport
```
```code
output:// 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010100f0a29a3b000000002321039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac00000000000000000ca4c3620100-7b22696e707574223a7b2230223a3235377d2c226f7574707574223a7b2230223a3235377d7d
```
```bash
$ ./qx tx-decode 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010100f0a29a3b000000002321039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac00000000000000000ca4c3620100-7b22696e707574223a7b2230223a3235377d2c226f7574707574223a7b2230223a3235377d7d
```
```json
{
	"txid": "696c2817e9d9815795eaa1801e71b1944f87dc82bda0aedcccd53961eeea3964",
	"txhash": "40c8e3ef5f5e99fe501919d888d60782bf06d4046c87aaf28307fa35ecd55c72",
	"version": 1,
	"locktime": 0,
	"expire": 0,
	"vin": [{
		"type": "TxTypeCrossChainExport",
		"scriptSig": {
			"asm": "",
			"hex": ""
		}
	}],
	"vout": [{
		"coin": "ETH",
		"coinid": 1,
		"amount": 999990000,
		"scriptPubKey": {
			"asm": "039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ff OP_CHECKSIG",
			"hex": "21039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac",
			"reqSigs": 1,
			"type": "pubkey",
			"addresses": ["MkB8FU1Q5DGaUnwSs6yDj66JwGeEP9KLTaxkbmSQhN9nTmp3ef5jk"]
		}
	}]
}
```
```bash
$ ./qx sign -k (privateKey) -n mixnet 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010100f0a29a3b000000002321039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac00000000000000000ca4c3620100-7b22696e707574223a7b2230223a3235377d2c226f7574707574223a7b2230223a3235377d7d
```
```code
output:// 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010100f0a29a3b000000002321039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac00000000000000000ca4c3620149483045022100f52afbaa1b3df5cd60023bcd5ab5ab931d8ddd4b6cae98860cfe731b2bcf1631022020dc0b80bf3a676f9c92887a2c25dbe84a5692ec297a50f6eb36f7b5ca3c3f3301
```
```bash
$ ./qx tx-decode 0100000001ca0265bbe73e89220da12dcfed31fb45ec3d18b60ea161036b418167bbd6da5f00000000ffffffff010100f0a29a3b000000002321039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac00000000000000000ca4c3620149483045022100f52afbaa1b3df5cd60023bcd5ab5ab931d8ddd4b6cae98860cfe731b2bcf1631022020dc0b80bf3a676f9c92887a2c25dbe84a5692ec297a50f6eb36f7b5ca3c3f3301
```
```json
{
	"txid": "696c2817e9d9815795eaa1801e71b1944f87dc82bda0aedcccd53961eeea3964",
	"txhash": "fe8e8fedc4c1366126fd6448581f3c0b0aa9eed645be7997508421fce08d951b",
	"version": 1,
	"locktime": 0,
	"expire": 0,
	"vin": [{
		"txid": "5fdad6bb6781416b0361a10eb6183dec45fb31edcf2da10d22893ee7bb6502ca",
		"vout": 0,
		"sequence": 4294967295,
		"scriptSig": {
			"asm": "3045022100f52afbaa1b3df5cd60023bcd5ab5ab931d8ddd4b6cae98860cfe731b2bcf1631022020dc0b80bf3a676f9c92887a2c25dbe84a5692ec297a50f6eb36f7b5ca3c3f3301",
			"hex": "483045022100f52afbaa1b3df5cd60023bcd5ab5ab931d8ddd4b6cae98860cfe731b2bcf1631022020dc0b80bf3a676f9c92887a2c25dbe84a5692ec297a50f6eb36f7b5ca3c3f3301"
		}
	}],
	"vout": [{
		"coin": "ETH",
		"coinid": 1,
		"amount": 999990000,
		"scriptPubKey": {
			"asm": "039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ff OP_CHECKSIG",
			"hex": "21039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac",
			"reqSigs": 1,
			"type": "pubkey",
			"addresses": ["MkB8FU1Q5DGaUnwSs6yDj66JwGeEP9KLTaxkbmSQhN9nTmp3ef5jk"]
		}
	}]
}
```

### TxTypeCrossChainImport  (TX from EVM to MEER)
```bash
$ ./qx tx-encode -v 1 -i 0000000000000000000000000000000000000000000000000000000000000000:4294967294:258:TxTypeCrossChainImport -l 0 -o XkCfdHoHHe2raZwNoY4sKcXFf6Jy9Q8XotAHenYsucPrEoj1FeUTR:100:0:TxTypeCrossChainImport
```
```code
output:// 01000000010000000000000000000000000000000000000000000000000000000000000000feffffff0201000001000000e40b54020000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac0000000000000000cba5c3620100-7b22696e707574223a7b2230223a3235387d2c226f7574707574223a7b2230223a3235387d7d
```
```bash
./qx tx-encode 01000000010000000000000000000000000000000000000000000000000000000000000000feffffff0201000001000000e40b54020000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac0000000000000000cba5c3620100-7b22696e707574223a7b2230223a3235387d2c226f7574707574223a7b2230223a3235387d7d
```
```json
{
	"txid": "a827b5b2017ccb0a805fcf3c3a0ea89989ae7c86a5019b7673e12f1a1f730e70",
	"txhash": "178c38b90566a6e89db52a73224a4c27f7854b85e63ff7ce5d3b62d538095545",
	"version": 1,
	"locktime": 0,
	"expire": 0,
	"vin": [{
		"type": "TxTypeCrossChainImport",
		"scriptSig": null
	}],
	"vout": [{
		"coin": "MEER",
		"amount": 10000000000,
		"scriptPubKey": {
			"asm": "OP_DUP OP_HASH160 ada117669b04771e481cc68ae8d4f33f913d1eda OP_EQUALVERIFY OP_CHECKSIG",
			"hex": "76a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac",
			"reqSigs": 1,
			"type": "pubkeyhash",
			"addresses": ["MmbRPt1wRcHz1mAiH8Ri1ANtGRyY5ygJ3hJ"]
		}
	}]
}
```
```bash
$ ./qx sign -k (privateKey) -n mixnet 01000000010000000000000000000000000000000000000000000000000000000000000000feffffff0201000001000000e40b54020000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac0000000000000000cba5c3620100-7b22696e707574223a7b2230223a3235387d2c226f7574707574223a7b2230223a3235387d7d
```
```cpde
output:// 01000000010000000000000000000000000000000000000000000000000000000000000000feffffff0201000001000000e40b54020000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac0000000000000000cba5c3620180354d6b42384655315135444761556e7753733679446a36364a7747654550394b4c5461786b626d5351684e396e546d70336566356a6b49483045022100ec14a41b314afc8c4348bab9122176f8d6b4ccbcaf4be1d2dc9be539cf89d98202203eb78a38c0403a8a8bcb63a30f4f39e1bd05031f55bd4c31a902567a5d6f155601
```
```bash
$ ./qx tx-decode 01000000010000000000000000000000000000000000000000000000000000000000000000feffffff0201000001000000e40b54020000001976a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac0000000000000000cba5c3620180354d6b42384655315135444761556e7753733679446a36364a7747654550394b4c5461786b626d5351684e396e546d70336566356a6b49483045022100ec14a41b314afc8c4348bab9122176f8d6b4ccbcaf4be1d2dc9be539cf89d98202203eb78a38c0403a8a8bcb63a30f4f39e1bd05031f55bd4c31a902567a5d6f155601
```
```json
{
	"txid": "a827b5b2017ccb0a805fcf3c3a0ea89989ae7c86a5019b7673e12f1a1f730e70",
	"txhash": "999a6a5283bfc0030ac82c7aee67dd5624b1f4da7241df8b2210963f4bfb8ce7",
	"version": 1,
	"locktime": 0,
	"expire": 0,
	"vin": [{
		"type": "TxTypeCrossChainImport",
		"scriptSig": null
	}],
	"vout": [{
		"coin": "MEER",
		"amount": 10000000000,
		"scriptPubKey": {
			"asm": "OP_DUP OP_HASH160 ada117669b04771e481cc68ae8d4f33f913d1eda OP_EQUALVERIFY OP_CHECKSIG",
			"hex": "76a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac",
			"reqSigs": 1,
			"type": "pubkeyhash",
			"addresses": ["MmbRPt1wRcHz1mAiH8Ri1ANtGRyY5ygJ3hJ"]
		}
	}]
}
```
