package common

import (
	"fmt"
	"time"
)

const (
	// currentDatabaseVersion indicates what the current database
	// version is.
	CurrentDatabaseVersion = 14
)

// -----------------------------------------------------------------------------
// The database information contains information about the version and date
// of the blockchain database.
//
// It consists of a separate key for each individual piece of information:
//
//   Key        Value    Size      Description
//   version    uint32   4 bytes   The version of the database
//   compver    uint32   4 bytes   The script compression version of the database
//   bidxver    uint32   4 bytes   The block index version of the database
//   created    uint64   8 bytes   The date of the creation of the database
// -----------------------------------------------------------------------------

// DatabaseInfo is the structure for a database.
type DatabaseInfo struct {
	version uint32
	compVer uint32
	bidxVer uint32
	created time.Time
}

func (di *DatabaseInfo) Version() uint32 {
	return di.version
}

func (di *DatabaseInfo) CompVer() uint32 {
	return di.compVer
}

func (di *DatabaseInfo) BidxVer() uint32 {
	return di.bidxVer
}

func (di *DatabaseInfo) Created() time.Time {
	return di.created
}

func (di *DatabaseInfo) String() string {
	return fmt.Sprintf("version:%d compVer:%d bidxVer:%d created:%s",
		di.version, di.compVer, di.bidxVer, di.created.String())
}

func NewDatabaseInfo(version uint32, compVer uint32, bidxVer uint32, created time.Time) *DatabaseInfo {
	return &DatabaseInfo{
		version: version,
		compVer: compVer,
		bidxVer: bidxVer,
		created: created,
	}
}
