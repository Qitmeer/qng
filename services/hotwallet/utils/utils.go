package utils

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
)

// FileExists reports whether the named file or directory exists.
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func FileCopy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func Encode(s interface{}) ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func Decode(val []byte, obj interface{}) error {
	decoder := gob.NewDecoder(bytes.NewReader(val))
	err := decoder.Decode(obj)
	if err != nil {
		return err
	}
	return nil
}
