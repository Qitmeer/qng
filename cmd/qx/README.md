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
    tx-lock-vin           lock vin script.
    tx-sign               sign a transactions using a private key.
    msg-sign              create a message signature
    msg-verify            validate a message signature
    signature-decode      decode a ECDSA signature
```

## Money from MEER to EVM

```
$ # pkscript
$ ./qx tx-lock-vin -i XkCfdHoHHe2raZwNoY4sKcXFf6Jy9Q8XotAHenYsucPrEoj1FeUTR:0
$ 2c24a926e6550d9897f8bfe21d9bef9c4a58cd21aa44668a969ad2c1885d3796
$ ./qx tx-encode -v 1 -i 2c24a926e6550d9897f8bfe21d9bef9c4a58cd21aa44668a969ad2c1885d3796:0:2147483647 -l 0 -o XkCfdHoHHe2raZwNoY4sKcXFf6Jy9Q8XotAHenYsucPrEoj1FeUTR:9.9999:1 |./qx tx-sign -k (privateKey) -p 21039d05472a845abf3cf5548567ee968d3ef3cd0f064fdb1bd2b6b791ab28f681ffac -n mixnet 
```

## Money from EVM to MEER

```
$ # pkhScript
$ ./qx tx-lock-vin -i XkCfdHoHHe2raZwNoY4sKcXFf6Jy9Q8XotAHenYsucPrEoj1FeUTR:2
$ 76a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac
$ #pkascript
$ ./qx tx-lock-vin -i XkCfdHoHHe2raZwNoY4sKcXFf6Jy9Q8XotAHenYsucPrEoj1FeUTR:3
$ 35586b436664486f4848653272615a774e6f5934734b63584666364a79395138586f744148656e597375635072456f6a314665555452
$ ./qx tx-encode -v 1 -i 0000000000000000000000000000000000000000000000000000000000000000:4294967294:258:35586b436664486f4848653272615a774e6f5934734b63584666364a79395138586f744148656e597375635072456f6a314665555452 -l 0 -o XmRTajVTajFiaEkd7PygFw46vNsoNW6fWE5:100:0 |./qx tx-sign -k (privateKey) -p 76a914ada117669b04771e481cc68ae8d4f33f913d1eda88ac -n mixnet 
```
