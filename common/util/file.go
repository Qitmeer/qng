package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

func GetPathByBrother(name string, brother string) (string, error) {
	bp, err := filepath.Abs(brother)
	if err != nil {
		return "", err
	}
	retPath := filepath.Join(bp, "../")
	var retPathAbs string
	retPathAbs, err = filepath.Abs(retPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(retPathAbs, name), nil
}
