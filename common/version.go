package common

type Version uint32

// version expand base
const VERSION_BITS_V2_EXPAND = 0xc0000000

// old version base v1
const VERSION_BITS_V1 = 0x20000000

// dividing line about the old version
const VERSION_BASE_VAL = VERSION_BITS_V2_EXPAND | VERSION_BITS_V1

// get base version
func (this *Version) IsOldVersion() bool {
	// old version
	return (*this) < VERSION_BASE_VAL
}

// get base version
func (this *Version) GetVersion() uint32 {
	if this.IsOldVersion() { // old version
		return uint32(*this)
	}
	version := (*this) ^ VERSION_BITS_V2_EXPAND
	return uint32(version)
}

func SetVersion(version uint32, useNewVersion bool) Version {
	if !useNewVersion {
		return Version(version)
	}
	version |= VERSION_BITS_V2_EXPAND
	return Version(version)
}
