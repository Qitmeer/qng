package blockchain

import (
	"fmt"
)

// update db to new version
func (b *BlockChain) upgradeDB(interrupt <-chan struct{}) error {
	if b.dbInfo.Version() == currentDatabaseVersion {
		return nil
	} else if b.dbInfo.Version() == 13 {
		return fmt.Errorf("The data is temporarily incompatible, we will find a solution as soon as possible. Currently, only newly synchronized data is supported")
	} else {
		return fmt.Errorf("Your database version(%d) is too old and can only use old qng (release-v1.0.x)\n", b.dbInfo.Version())
	}
}
