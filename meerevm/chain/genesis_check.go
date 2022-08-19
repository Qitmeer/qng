package chain

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
)

const MainAllocHash = "204bb2f79e29453543b74b620533c854ddd327603fdcadb5c938b4e29e2cbd0b"
const MixAllocHash = "e380c81b956194ce4e38c218f9b99300ef725f58b7ec13963fea763c8379446f"
const TestAllocHash = "e380c81b956194ce4e38c218f9b99300ef725f58b7ec13963fea763c8379446f"
const PrivAllocHash = "e380c81b956194ce4e38c218f9b99300ef725f58b7ec13963fea763c8379446f"

func Check() bool {
	mainAllocDataHash := crypto.Keccak256([]byte(mainAllocData))
	mixAllocDataHash := crypto.Keccak256([]byte(mixAllocData))
	testAllocDataHash := crypto.Keccak256([]byte(testAllocData))
	privAllocDataHash := crypto.Keccak256([]byte(privAllocData))
	fmt.Printf("mainHash:\n%v\nmixHash:%v\ntestHash:%v\nprivHash:%v\n", hex.EncodeToString(mainAllocDataHash), hex.EncodeToString(mixAllocDataHash),
		hex.EncodeToString(testAllocDataHash), hex.EncodeToString(privAllocDataHash))

	if MainAllocHash != hex.EncodeToString(mainAllocDataHash) || MixAllocHash != hex.EncodeToString(mixAllocDataHash) ||
		TestAllocHash != hex.EncodeToString(testAllocDataHash) || PrivAllocHash != hex.EncodeToString(privAllocDataHash) {
		return false
	}
	return true
}
