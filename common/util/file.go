package util

import (
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

	ba, err := os.ReadFile(path)
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

// IsNonEmptyDir checks if a directory exists and is non-empty.
func IsNonEmptyDir(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer f.Close()
	names, _ := f.Readdirnames(1)
	return len(names) > 0
}
