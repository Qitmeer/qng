//go:build !windows && !openbsd
// +build !windows,!openbsd

package disk

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func GetFreeDiskSpace(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to call Statfs: %v", err)
	}

	// Available blocks * size per block = available space in bytes
	var bavail = stat.Bavail
	// nolint:staticcheck
	if stat.Bavail < 0 {
		// FreeBSD can have a negative number of blocks available
		// because of the grace limit.
		bavail = 0
	}
	//nolint:unconvert
	return uint64(bavail) * uint64(stat.Bsize), nil
}
