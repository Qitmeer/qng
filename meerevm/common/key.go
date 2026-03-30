/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package common

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"io"
	"os"
	"path/filepath"
)

const (
	DefaultKeyfileName = "keyfile.json"
)

type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

func NewKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *Key {
	key := &Key{
		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key
}

func NewKey(rand io.Reader) (*Key, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}
	return NewKeyFromECDSA(privateKeyECDSA), nil
}

func GenerateKeyfile(privateKeyHex string, keyfilepath string, jsonFormat bool, lightKDF bool) error {
	// Check if keyfile path given and make sure it doesn't already exist.
	if keyfilepath == "" {
		keyfilepath = DefaultKeyfileName
	}
	if _, err := os.Stat(keyfilepath); err == nil {
		return fmt.Errorf("Keyfile already exists at %s.", keyfilepath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("Error checking if keyfile exists: %v", err)
	}
	if len(privateKeyHex) <= 0 {
		return fmt.Errorf("Private key is empty")
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return err
	}

	// Create the keyfile object with a random UUID.
	UUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("Failed to generate random uuid: %v", err)
	}
	key := &keystore.Key{
		Id:         UUID,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey: privateKey,
	}

	// Encrypt key with passphrase.
	passphrase := utils.GetPassPhrase("", true)
	scryptN, scryptP := keystore.StandardScryptN, keystore.StandardScryptP
	if lightKDF {
		scryptN, scryptP = keystore.LightScryptN, keystore.LightScryptP
	}
	keyjson, err := keystore.EncryptKey(key, passphrase, scryptN, scryptP)
	if err != nil {
		return fmt.Errorf("Error encrypting key: %v", err)
	}

	// Store the file to disk.
	if err := os.MkdirAll(filepath.Dir(keyfilepath), 0700); err != nil {
		return fmt.Errorf("Could not create directory %s", filepath.Dir(keyfilepath))
	}
	if err := os.WriteFile(keyfilepath, keyjson, 0600); err != nil {
		return fmt.Errorf("Failed to write keyfile to %s: %v", keyfilepath, err)
	}

	// Output some information.
	type outputGenerate struct {
		Address string
	}
	out := outputGenerate{
		Address: key.Address.Hex(),
	}
	if jsonFormat {
		mustPrintJSON(out)
	} else {
		fmt.Println("Address:", out.Address)
	}
	return nil
}

func InspectKeyFile(keyfilepath string, jsonFormat bool, showPrivate bool) (string, error) {
	// Read key from file.
	keyjson, err := os.ReadFile(keyfilepath)
	if err != nil {
		utils.Fatalf("Failed to read the keyfile at '%s': %v", keyfilepath, err)
	}

	// Decrypt key with passphrase.
	passphrase := utils.GetPassPhrase("", false)
	key, err := keystore.DecryptKey(keyjson, passphrase)
	if err != nil {
		utils.Fatalf("Error decrypting key: %v", err)
	}

	// Output all relevant information we can retrieve.
	type outputInspect struct {
		Address    string
		PublicKey  string
		PrivateKey string
	}
	out := outputInspect{
		Address: key.Address.Hex(),
		PublicKey: hex.EncodeToString(
			crypto.FromECDSAPub(&key.PrivateKey.PublicKey)),
	}
	prvkey := hex.EncodeToString(crypto.FromECDSA(key.PrivateKey))
	if showPrivate {
		out.PrivateKey = prvkey
	}

	if jsonFormat {
		mustPrintJSON(out)
	} else {
		fmt.Println("Address:       ", out.Address)
		fmt.Println("Public key:    ", out.PublicKey)
		if showPrivate {
			fmt.Println("Private key:   ", out.PrivateKey)
		}
	}
	return prvkey, nil
}

// mustPrintJSON prints the JSON encoding of the given object and
// exits the program with an error message when the marshaling fails.
func mustPrintJSON(jsonObject interface{}) {
	str, err := json.MarshalIndent(jsonObject, "", "  ")
	if err != nil {
		utils.Fatalf("Failed to marshal JSON object: %v", err)
	}
	fmt.Println(string(str))
}
