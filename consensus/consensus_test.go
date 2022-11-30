package consensus

import (
	"github.com/Qitmeer/qng/common/util"
	_ "github.com/Qitmeer/qng/database/ffldb"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/common"
	"os"
	"path/filepath"
	"testing"
)

func TestAloneConsensus(t *testing.T) {
	cfg:=common.DefaultConfig("./test")
	cfg.NoFileLogging=true
	cfg.DataDir = util.CleanAndExpandPath(cfg.DataDir)
	cfg.DataDir = filepath.Join(cfg.DataDir, params.ActiveNetParams.Name)
	//
	db, err := common.LoadBlockDB(cfg)
	if err != nil {
		t.Error(err)
	}
	cons:=NewPure(cfg, db)
	err = cons.Init()
	if err != nil {
		t.Error(err)
	}
	// remove temporary data
	err = os.RemoveAll(cfg.HomeDir)
	if err != nil {
		t.Error(err)
	}
}