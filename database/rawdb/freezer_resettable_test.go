package rawdb

import (
	"bytes"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

func TestResetFreezer(t *testing.T) {
	items := []struct {
		id   uint64
		blob []byte
	}{
		{0, bytes.Repeat([]byte{0}, 2048)},
		{1, bytes.Repeat([]byte{1}, 2048)},
		{2, bytes.Repeat([]byte{2}, 2048)},
	}
	f, _ := NewResettableFreezer(t.TempDir(), "", false, 2048, freezerTestTableDef)
	defer f.Close()

	f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for _, item := range items {
			op.AppendRaw("test", item.id, item.blob)
		}
		return nil
	})
	for _, item := range items {
		blob, _ := f.Ancient("test", item.id)
		if !bytes.Equal(blob, item.blob) {
			t.Fatal("Unexpected blob")
		}
	}

	// Reset freezer
	f.Reset()
	count, _ := f.Ancients()
	if count != 0 {
		t.Fatal("Failed to reset freezer")
	}
	for _, item := range items {
		blob, _ := f.Ancient("test", item.id)
		if len(blob) != 0 {
			t.Fatal("Unexpected blob")
		}
	}

	// Fill the freezer
	f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for _, item := range items {
			op.AppendRaw("test", item.id, item.blob)
		}
		return nil
	})
	for _, item := range items {
		blob, _ := f.Ancient("test", item.id)
		if !bytes.Equal(blob, item.blob) {
			t.Fatal("Unexpected blob")
		}
	}
}

func TestFreezerCleanup(t *testing.T) {
	items := []struct {
		id   uint64
		blob []byte
	}{
		{0, bytes.Repeat([]byte{0}, 2048)},
		{1, bytes.Repeat([]byte{1}, 2048)},
		{2, bytes.Repeat([]byte{2}, 2048)},
	}
	datadir := t.TempDir()
	f, _ := NewResettableFreezer(datadir, "", false, 2048, freezerTestTableDef)
	f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for _, item := range items {
			op.AppendRaw("test", item.id, item.blob)
		}
		return nil
	})
	f.Close()
	os.Rename(datadir, tmpName(datadir))

	// Open the freezer again, trigger cleanup operation
	f, _ = NewResettableFreezer(datadir, "", false, 2048, freezerTestTableDef)
	f.Close()

	if _, err := os.Lstat(tmpName(datadir)); !os.IsNotExist(err) {
		t.Fatal("Failed to cleanup leftover directory")
	}
}
