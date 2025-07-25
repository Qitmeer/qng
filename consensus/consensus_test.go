package consensus

import (
	"github.com/Qitmeer/qng/v2/common/system"
	"github.com/Qitmeer/qng/v2/common/util"
	"github.com/Qitmeer/qng/v2/database"
	_ "github.com/Qitmeer/qng/v2/database/legacydb/ffldb"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/Qitmeer/qng/v2/services/common"
	"os"
	"path/filepath"
	"testing"
)

func TestAloneConsensus(t *testing.T) {
	cfg := common.DefaultConfig("./test")
	cfg.NoFileLogging = true
	cfg.DataDir = util.CleanAndExpandPath(cfg.DataDir)
	cfg.DataDir = filepath.Join(cfg.DataDir, params.ActiveNetParams.Name)
	//
	db, err := database.New(cfg, system.InterruptListener())
	if err != nil {
		t.Error(err)
	}

	cons := NewPure(cfg, db)
	err = cons.Init()
	if err != nil {
		t.Error(err)
	}
	db.Close()
	// remove temporary data
	err = os.RemoveAll(cfg.HomeDir)
	if err != nil {
		t.Error(err)
	}
}
