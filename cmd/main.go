/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package main

import (
	"fmt"
	"github.com/Qitmeer/meerevm/evm"
)

func main() {
	fmt.Println(evm.New().Version())
}
