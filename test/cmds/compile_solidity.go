//go:build none
// +build none

// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// wget https://github.com/ethereum/solidity/releases/download/v0.8.26/solc-static-linux
// solc--version
// solc, the solidity compiler commandline interface
// Version: 0.8.26+commit.8a97fa7a.Linux.g++

const SOLC = "solc"

// go install github.com/ethereum/go-ethereum/cmd/abigen@latest
const ABIGEN = "abigen"

var fileContent = "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run compile_solidity.go \npackage testcommon\n\n"

func main() {
	filepath := "../testcommon/solidity_bin.go"
	f, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}

	// compile solidity
	compileToken()
	compileWETH()
	compileSwapFactory()
	compileSwapRouter()
	// generate file
	f.WriteString(fileContent)
	fmt.Println("Successfully updated:", filepath)
}

func getPrefix(filePath string) string {
	if SOLC == "solc" {
		return ""
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalln(err)
		return ""
	}
	dir := filepath.Dir(absPath)
	absFile := dir + "/" + filepath.Base(absPath)
	absFile = strings.ReplaceAll(absFile, "/", "_")
	absFile = strings.ReplaceAll(absFile, ".", "_")
	return absFile + "_"
}
func compileToken() {
	fileName := "../token/meererc20.sol"
	if execCompileSolidity(fileName) {
		execCMD("ls")
		execCMD("ls", "./build")
		prefix := fmt.Sprintf("./build/%sMEER20USDT", getPrefix(fileName))
		// ___{dir}_{filename}_sol_{contractname}.bin

		b, err := ioutil.ReadFile(prefix + ".bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const ERC20Code ="%s"
`, string(b))
		// generate abi.go
		execABIGO(prefix+".abi", "token", "../token/meererc20.go")
	}
}
func compileWETH() {
	fileName := "../swap/weth.sol"
	if execCompileSolidity(fileName) {
		prefix := fmt.Sprintf("./build/%sMockWETH", getPrefix(fileName))
		b, err := ioutil.ReadFile(prefix + ".bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const WETH ="%s"
`, string(b))
		// generate abi.go
		execABIGO(prefix+".abi", "weth", "../swap/weth/weth.go")
	}
}

func compileSwapFactory() {
	fileName := "../swap/factory.sol"
	if execCompileSolidity(fileName) {
		prefix := fmt.Sprintf("./build/%sMockUniswapV2FactoryUniswapV2Pair", getPrefix(fileName))
		b, err := ioutil.ReadFile(prefix + ".bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const PAIR ="%s"
`, string(b))
		// generate abi.go
		execABIGO(prefix+".abi", "pair", "../swap/pair/pair.go")
		prefix = fmt.Sprintf("./build/%sMockUniswapV2Factory", getPrefix(fileName))
		b, err = ioutil.ReadFile(prefix + ".bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const FACTORY ="%s"
`, string(b))
		// generate abi.go
		execABIGO(prefix+".abi", "factory", "../swap/factory/factory.go")
	}
}

func compileSwapRouter() {
	fileName := "../swap/router.sol"
	if execCompileSolidity(fileName) {
		prefix := fmt.Sprintf("./build/%sMockUniswapV2Router02", getPrefix(fileName))
		b, err := ioutil.ReadFile(prefix + ".bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const ROUTER ="%s"
`, string(b))
		// generate abi.go
		execABIGO(prefix+".abi", "router", "../swap/router/router.go")
	}
}

func execCompileSolidity(filename string) bool {
	execCMD("rm", "-rf", "build")
	return execCMD(SOLC, "--optimize", "--optimize-runs", "200", "--bin", "--abi", filename, "-o", "build")
}

func execABIGO(filename, packagename, outfilepath string) bool {
	return execCMD(ABIGEN, "--abi", filename, "--pkg", packagename, "--type", "Token", "-out", outfilepath)
}

func execCMD(name string, arg ...string) bool {
	cmd := exec.Command(name, arg...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	b, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(name, string(b))
	return true
}
