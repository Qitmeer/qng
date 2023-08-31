package main

import (
	"github.com/Qitmeer/qng/common/system"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

func TestShutdown(t *testing.T) {
	defer goleak.VerifyNone(t)

	runtime.GOMAXPROCS(runtime.NumCPU())
	debug.SetGCPercent(20)

	var err error
	var wg sync.WaitGroup
	waitTime:=time.Second*10
	wg.Add(1)
	go func() {
		os.Args = []string{"-A=./", "--privnet"}
		err = qng()
		if  err != nil {
			t.Error(err.Error())
		}
		wg.Done()
	}()

	t.Logf("It will auto shutdown after %s",waitTime)
	time.Sleep(waitTime)
	system.ShutdownRequestChannel <- struct{}{}
	wg.Wait()

	require.NoError(t, err)
}
