package main

import (
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/rollups"
)

func main()  {
	interrupt := system.InterruptListener()
	rn,err:=rollups.New(nil,nil)
	if err != nil {
		return
	}
	defer rn.Stop()
	err=rn.Start(interrupt)
	if err != nil {
		return
	}
}
