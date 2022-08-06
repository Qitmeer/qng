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
)

// npm install solcjs
// solcjs--version
// 0.8.3+commit.8d00100c.Emscripten.clang

const SOLC = "solcjs"

// go run github.com/Qitmeer/go-etherum/cmd/abigen/main.go
const ABIGEN = "abigen"

var fileContent = "// It is called by go generate and used to automatically generate pre-computed \n// Copyright 2017-2022 The qitmeer developers \n// This file is auto generate by : go run compile_solidity.go \npackage testutils\n\n"

func main() {
	filepath := "../solidity_bin.go"
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

func compileToken() {
	if execCompileSolidity("../token/meererc20.sol") {
		execCMD("ls")
		execCMD("ls", "./build")
		// ___{dir}_{filename}_sol_{contractname}.bin
		b, err := ioutil.ReadFile("./build/___token_meererc20_sol_MEER20USDT.bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const ERC20Code ="%s"
`, string(b))
		// generate abi.go
		execABIGO("./build/___token_meererc20_sol_MEER20USDT.abi", "token", "../token/meererc20.go")
	}
}

func compileWETH() {
	if execCompileSolidity("../swap/weth.sol") {
		b, err := ioutil.ReadFile("./build/___swap_weth_sol_MockWETH.bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const WETH ="%s"
`, string(b))
		// generate abi.go
		execABIGO("./build/___swap_weth_sol_MockWETH.abi", "weth", "../swap/weth/weth.go")
	}
}

func compileSwapFactory() {
	if execCompileSolidity("../swap/factory.sol") {
		b, err := ioutil.ReadFile("./build/___swap_factory_sol_MockUniswapV2FactoryUniswapV2Pair.bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const PAIR ="%s"
`, string(b))
		// generate abi.go
		execABIGO("./build/___swap_factory_sol_MockUniswapV2FactoryUniswapV2Pair.abi", "pair", "../swap/pair/pair.go")
		b, err = ioutil.ReadFile("./build/___swap_factory_sol_MockUniswapV2Factory.bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const FACTORY ="%s"
`, string(b))
		// generate abi.go
		execABIGO("./build/___swap_factory_sol_MockUniswapV2Factory.abi", "factory", "../swap/factory/factory.go")
	}
}

func compileSwapRouter() {
	if execCompileSolidity("../swap/router.sol") {
		b, err := ioutil.ReadFile("./build/___swap_router_sol_MockUniswapV2Router02.bin")
		if err != nil {
			log.Fatal(err)
		}
		fileContent += fmt.Sprintf(`
const ROUTER ="%s"
`, string(b))
		// generate abi.go
		execABIGO("./build/___swap_router_sol_MockUniswapV2Router02.abi", "router", "../swap/router/router.go")
	}
}

func execCompileSolidity(filename string) bool {
	return execCMD(SOLC, "--optimize", "--bin", "--abi", filename, "-o", "build")
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
