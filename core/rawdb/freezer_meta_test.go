package rawdb

import (
	"os"
	"testing"
)

func TestReadWriteFreezerTableMeta(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "*")
	if err != nil {
		t.Fatalf("Failed to create file %v", err)
	}
	err = writeMetadata(f, newMetadata(100))
	if err != nil {
		t.Fatalf("Failed to write metadata %v", err)
	}
	meta, err := readMetadata(f)
	if err != nil {
		t.Fatalf("Failed to read metadata %v", err)
	}
	if meta.Version != freezerVersion {
		t.Fatalf("Unexpected version field")
	}
	if meta.VirtualTail != uint64(100) {
		t.Fatalf("Unexpected virtual tail field")
	}
}

func TestInitializeFreezerTableMeta(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "*")
	if err != nil {
		t.Fatalf("Failed to create file %v", err)
	}
	meta, err := loadMetadata(f, uint64(100))
	if err != nil {
		t.Fatalf("Failed to read metadata %v", err)
	}
	if meta.Version != freezerVersion {
		t.Fatalf("Unexpected version field")
	}
	if meta.VirtualTail != uint64(100) {
		t.Fatalf("Unexpected virtual tail field")
	}
}
