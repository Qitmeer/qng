package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type VersionCheck struct {
	OriginVersion uint32
	BitNumber     uint8
	NewMode       bool
	NewVersion    Version
}

var tests = []VersionCheck{
	{
		OriginVersion: VERSION_BITS_V1,
		BitNumber:     0,
		NewMode:       false,
	},
	{
		OriginVersion: VERSION_BITS_V1,
		BitNumber:     0,
		NewMode:       true,
	},
	{
		OriginVersion: VERSION_BITS_V1,
		BitNumber:     1,
		NewMode:       true,
	},
	{
		OriginVersion: VERSION_BITS_V1,
		BitNumber:     28,
		NewMode:       true,
	},
	{
		OriginVersion: VERSION_BITS_V1,
		BitNumber:     29,
		NewMode:       false,
	},
}

func TestVersion(t *testing.T) {
	for k, v := range tests {
		tests[k].OriginVersion |= 1 << v.BitNumber
		tests[k].NewVersion = SetVersion(tests[k].OriginVersion, v.NewMode)
	}
	for _, v := range tests {
		decodeVersion := v.NewVersion.GetVersion()
		assert.Equal(t, v.NewVersion >= VERSION_BASE_VAL, v.NewMode)
		assert.Equal(t, v.OriginVersion, decodeVersion)
	}
}
