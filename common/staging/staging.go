package staging

import (
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/database"
	l "github.com/Qitmeer/qng/log"
	"sync/atomic"
)

// CommitAllChanges creates a transaction in `databaseContext`, and commits all changes in `stagingArea` through it.
func CommitAllChanges(databaseContext database.DB, stagingArea *model.StagingArea) error {
	if onEnd := l.LogAndMeasureExecutionTime(log, "CommitAllChanges"); onEnd != nil {
		defer onEnd()
	}
	return databaseContext.Update(func(dbTx database.Tx) error {
		return stagingArea.Commit(dbTx)
	})
}

var lastShardingID uint64

// GenerateShardingID generates a unique staging sharding ID.
func GenerateShardingID() model.StagingShardID {
	return model.StagingShardID(atomic.AddUint64(&lastShardingID, 1))
}
