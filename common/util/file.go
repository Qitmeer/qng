package util

import (
	"io/ioutil"
	"os"
)

// FilesExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func ReadFile(path string) ([]byte, error) {
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsExist(err) {
			return nil, err
		}
	}

	ba, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ba, nil
}
