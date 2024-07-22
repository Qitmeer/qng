// Copyright (c) 2017-2024 The qitmeer developers
package rawdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb/ancienttest"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestMemoryFreezer(t *testing.T) {
	ancienttest.TestAncientSuite(t, func(kinds []string) ethdb.AncientStore {
		tables := make(map[string]bool)
		for _, kind := range kinds {
			tables[kind] = true
		}
		return NewMemoryFreezer(false, tables)
	})
	ancienttest.TestResettableAncientSuite(t, func(kinds []string) ethdb.ResettableAncientStore {
		tables := make(map[string]bool)
		for _, kind := range kinds {
			tables[kind] = true
		}
		return NewMemoryFreezer(false, tables)
	})
}
