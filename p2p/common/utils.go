package common

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/crypto"
	"io/ioutil"
	"os"
	"path"
)

const keyPath = "network.key"

// Determines a private key for p2p networking from the p2p service's
// configuration struct. If no key is found, it generates a new one.
func PrivateKey(dataDir string, privateKeyPath string, readWritePermissions os.FileMode) (crypto.PrivKey, error) {
	if len(dataDir) <= 0 && len(privateKeyPath) <= 0 {
		priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			return nil, err
		}
		return priv, nil
	}
	defaultKeyPath := path.Join(dataDir, keyPath)

	_, err := os.Stat(defaultKeyPath)
	defaultKeysExist := !os.IsNotExist(err)
	if err != nil && defaultKeysExist {
		return nil, err
	}

	if privateKeyPath == "" && !defaultKeysExist {
		priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			return nil, err
		}
		rawbytes, err := priv.Raw()
		if err != nil {
			return nil, err
		}
		dst := make([]byte, hex.EncodedLen(len(rawbytes)))
		hex.Encode(dst, rawbytes)
		if err = ioutil.WriteFile(defaultKeyPath, dst, readWritePermissions); err != nil {
			return nil, err
		}
		return priv, nil
	}
	if defaultKeysExist && privateKeyPath == "" {
		privateKeyPath = defaultKeyPath
	}
	return retrievePrivKeyFromFile(privateKeyPath)
}

// Retrieves a p2p networking private key from a file path.
func retrievePrivKeyFromFile(path string) (crypto.PrivKey, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading private key from file:%v", err)
	}
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err = hex.Decode(dst, src)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string:%w", err)
	}
	unmarshalledKey, err := crypto.UnmarshalSecp256k1PrivateKey(dst)
	if err != nil {
		return nil, err
	}
	return unmarshalledKey, nil
}

func ToECDSAPrivKey(privKey crypto.PrivKey) (*ecdsa.PrivateKey, error) {
	pkb, err := privKey.Raw()
	if err != nil {
		return nil, err
	}
	pk, err := ecrypto.HexToECDSA(hex.EncodeToString(pkb))
	if err != nil {
		return nil, err
	}
	return pk, nil
}
