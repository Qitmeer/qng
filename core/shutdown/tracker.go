package shutdown

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"io/ioutil"
	"os"
	"path"
)

const lockFileName = "shutdown.lock"

type Tracker struct {
	filePath string
}

func (t *Tracker) Check() error {
	if !Exists(t.filePath) {
		return nil
	}
	bhbs, err := ReadFile(t.filePath)
	if err != nil {
		return err
	}
	if len(bhbs) <= 0 {
		return nil
	}
	bh := hash.MustHexToHash(string(bhbs))
	err = fmt.Errorf("Illegal withdrawal at block:%s, you can cleanup your block data base by '--cleanup'.", bh.String())
	log.Error(err.Error())
	return err
}

func (t *Tracker) Wait(bh *hash.Hash) error {
	outFile, err := os.OpenFile(t.filePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		outFile.Close()
	}()
	//
	_, err = outFile.WriteString(bh.String())
	if err != nil {
		return err
	}
	return nil
}

func (t *Tracker) Done() error {
	if !Exists(t.filePath) {
		return nil
	}
	return os.Remove(t.filePath)
}

func NewTracker(datadir string) *Tracker {
	return &Tracker{filePath: path.Join(datadir, lockFileName)}
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func ReadFile(path string) ([]byte, error) {
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsExist(err) {
			return nil, err
		}
	}

	ba, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ba, nil
}
