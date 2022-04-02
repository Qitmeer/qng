package types

import (
	"fmt"
	"testing"
)

func TestDe(t *testing.T) {
	fmt.Println(decimalToAny(536870912, 2))
	fmt.Printf("0x%x\n", anyToDecimal("00100000000000000000000000000000", 2))
	fmt.Printf("0x%x\n", anyToDecimal("00111111111100000000000000000000", 2))
}
