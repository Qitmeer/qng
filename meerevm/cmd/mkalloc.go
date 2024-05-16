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
	fileContent := "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run mkalloc.go \npackage meer\n\n"
	fileContent += fmt.Sprintf("\nconst genesisJson = `%s`", genesisJ)
	fileContent += fmt.Sprintf("\nconst burnListJson = `%s`", burnListJ)

	for _, np := range params.AllNetParams {
		alloc := meer.DoDecodeAlloc(np.Params, genesisJ, burnListJ)
		genesis := meer.Genesis(np.Net, alloc)
		genesisHash := genesis.ToBlock().Hash()
		log.Printf("network = %s, genesisHash= %s\n", np.Name, genesisHash.String())
		fileContent += fmt.Sprintf("\n\nconst %sGenesisHash = \"%s\"", np.Net.String(), genesisHash.String())
	}

	fileName := "./../meer/genesis_alloc.go"

	f, err := os.Create(fileName)

	if err != nil {
		panic(fmt.Sprintf("Save error:%s  %s", fileName, err))
	}
	defer func() {
		err = f.Close()
	}()

	f.WriteString(fileContent)

	fmt.Println("Successfully updated:", fileName)
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
