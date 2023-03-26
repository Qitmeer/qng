package amana

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"os"
	"path"
)

func Cleanup(cfg *config.Config) {
	dbPath := path.Join(cfg.DataDir, ClientIdentifier)
	err := os.RemoveAll(dbPath)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info(fmt.Sprintf("Finished cleanup:%s", dbPath))
}
