package blockchain

import (
	"fmt"
)

// update db to new version
func (b *BlockChain) upgradeDB(interrupt <-chan struct{}) error {
	if b.dbInfo.Version() == currentDatabaseVersion {
		return nil
	} else {
		return fmt.Errorf("Your database version(%d) is too old and can only use old qng\n", b.dbInfo.Version())
	}
}
