// (c) 2021, the Qitmeer developers. All rights reserved.
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"os"
)

var (
	// Version is the version of MeerEvm
	Version = "meerevm-v0.0.0"
)

func main() {
	fmt.Println(Version)
	os.Exit(0)
}
