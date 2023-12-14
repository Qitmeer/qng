package simulator

import (
	"fmt"
	"testing"
	"time"
)

func TestMockNode(t *testing.T) {
	node, err := StartMockNode()
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		fmt.Println("wait for:", i)
	}
	defer func() {
		err = node.Stop()
		if err != nil {
			t.Error(err)
		}
	}()
}
