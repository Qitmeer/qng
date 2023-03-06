package utils

import (
	"testing"
)

func TestOpen(t *testing.T) {
	OpenBrowser("http://www.baidu.com")
}

func TestUserDataDir(t *testing.T) {

	if dir := GetUserDataDir(); dir == "" {
		t.Errorf("Failed to get user data directory")
	} else if err := MakeDirAll(dir); err != nil {
		t.Errorf("Failed to make dir %s", dir)
	}
}
