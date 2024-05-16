//go:build none
// +build none

package main

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/params"
	"log"
	"os"
)

func main() {
	genesisJ, err := readFromFile("./../meer/genesis.json")
	if err != nil {
		panic(err)
	}
	burnListJ, err := readFromFile("./../meer/burn_list.json")
	if err != nil {
		panic(err)
	}
	fileHeader := "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run mkalloc.go \npackage meer\n\n"
	fileContent := fileHeader
	fileContent += fmt.Sprintf("\nconst genesisJson = `%s`", genesisJ)
	fileContent += fmt.Sprintf("\nconst burnListJson = `%s`", burnListJ)
	err = saveFile("./../meer/genesis_alloc.go", fileContent)
	if err != nil {
		panic(err)
	}
	fileContent = fileHeader
	for _, np := range params.AllNetParams {
		alloc := meer.DoDecodeAlloc(np.Params, genesisJ, burnListJ)
		genesis := meer.Genesis(np.Net, alloc)
		genesisHash := genesis.ToBlock().Hash()
		log.Printf("network = %s, genesisHash= %s\n", np.Name, genesisHash.String())
		fileContent += fmt.Sprintf("\nconst %sGenesisHash = \"%s\"", np.Net.String(), genesisHash.String())
	}
	err = saveFile("./../meer/genesis_hash.go", fileContent)
	if err != nil {
		panic(err)
	}
}

func readFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	var buff bytes.Buffer
	_, err = buff.ReadFrom(file)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}

func saveFile(fileName string, fileContent string) error {
	f, err := os.Create(fileName)

	if err != nil {
		return fmt.Errorf("Save error:%s  %s", fileName, err)
	}
	defer func() {
		err = f.Close()
	}()

	f.WriteString(fileContent)

	fmt.Println("Successfully updated:", fileName)

	return nil
}
