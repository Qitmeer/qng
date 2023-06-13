package peers

import (
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"time"
)

type BadResponse struct {
	ID   uint64
	Time time.Time
	Err  *common.Error
}

func (br *BadResponse) String() string {
	return fmt.Sprintf("id:%d time:%s err:%s", br.ID, br.Time.Format(time.RFC3339), br.Err.String())
}
