package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//OpenBrowser opens web brower
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	//log.Info(runtime.GOOS)

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		//protocl pre require  http https
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// GetUserDataDir get user data dir
func GetUserDataDir() string {
	homePath := ""

	switch runtime.GOOS {
	case "windows":
		homePath := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if homePath == "" {
			homePath = os.Getenv("USERPROFILE")
		}
	//case "darwin":
	default: // "linux", "freebsd", "openbsd", "netbsd"
		homePath = os.Getenv("HOME")
	}

	return filepath.Join(homePath, ".qitmeer-wallet")
}

// MakeDirAll make dir
func MakeDirAll(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return nil
	}

	return os.MkdirAll(path, os.ModePerm)
}
