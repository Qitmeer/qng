//go:build !windows
// +build !windows

package rawdb

import (
	"errors"
	"os"
	"syscall"
)

func syncDir(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := f.Sync(); err != nil {
		if errors.Is(err, os.ErrInvalid) {
			return nil
		}
		if patherr, ok := err.(*os.PathError); ok && patherr.Err == syscall.EINVAL {
			return nil
		}
		return err
	}
	return nil
}