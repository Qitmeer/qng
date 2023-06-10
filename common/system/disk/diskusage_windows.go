package disk

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func GetFreeDiskSpace(path string) (uint64, error) {

	cwd, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("failed to call UTF16PtrFromString: %v", err)
	}

	var freeBytesAvailableToCaller, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(cwd, &freeBytesAvailableToCaller, &totalNumberOfBytes, &totalNumberOfFreeBytes); err != nil {
		return 0, fmt.Errorf("failed to call GetDiskFreeSpaceEx: %v", err)
	}

	return freeBytesAvailableToCaller, nil
}
