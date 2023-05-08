package blockchain

import (
	"fmt"
)

// update db to new version
func (b *BlockChain) upgradeDB(interrupt <-chan struct{}) error {
	if b.dbInfo.version == currentDatabaseVersion {
		return nil
	} else {
		return fmt.Errorf("Support version(%d), but cur db is version:%d. Please use the old qng version\n", currentDatabaseVersion, b.dbInfo.version)
	}
}
