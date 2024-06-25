// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestNewNodeCmdArgs(t *testing.T) {
	testDir, _ := ioutil.TempDir("", "test")
	defer os.RemoveAll(testDir)
	c := newNodeConfig(testDir, []string{"--k1=v1", "--k2=v2"})
	node, err := newNode(t, c)
	if err != nil {
		t.Errorf("failed to create new node : %v", err)
	}
	if node.cmd.Dir != "" {
		t.Errorf("failed to create node, expect %v but got %v", "", node.cmd.Dir)
	}
	args := []string{
		"qng",
		"--listen=127.0.0.1:38130",
		"--rpclisten=127.0.0.1:38131",
		`--evmenv="--http --http.port=18545 --ws --ws.port=18546"`,
		"--rpcuser=testuser",
		"--rpcpass=testpass",
		"--appdata=.*/test.*$",
		"--datadir=.*/test.*/data$",
		"--logdir=.*/test.*/log$",
		"--k1=v1",
		"--k2=v2",
	}
	for i := 0; i < len(args); i++ {
		expect, got := args[i], node.cmd.Args[i]
		if i >= 5 && i <= 8 { // test app/data/log dir
			if !regexp.MustCompile(expect).MatchString(got) {
				t.Errorf("failed to create node, expect %v but got %v", expect, got)
			}
		} else { // other test
			if !reflect.DeepEqual(expect, got) {
				t.Errorf("failed to create node, expect %v but got %v", expect, got)
			}
		}
	}
}

func TestNodeStartStop(t *testing.T) {
	found, err := exec.LookPath("qng")
	if err != nil {
		t.Skip(fmt.Sprintf("skip the test since: %v", err))
	} else {
		t.Logf("found qng execuable at %v", found)
	}
	testDir, _ := ioutil.TempDir("", "test")
	defer os.RemoveAll(testDir)
	c := newNodeConfig(testDir, []string{"--privnet"})
	n, err := newNode(t, c)
	if err != nil {
		t.Errorf("new node failed :%v", err)
	}
	err = n.start()
	if err != nil {
		t.Errorf("new node start failed :%v", err)
	}
	time.Sleep(200 * time.Millisecond)
	err = n.stop()
	if err != nil {
		t.Errorf("new node stop failed :%v", err)
	}

}

func TestGenListenArgs(t *testing.T) {
	c := newNodeConfig("test", nil)
	a1, a2, a3, a4, a5 := genListenArgs()
	c.listen, c.rpclisten, c.evmlisten, c.evmWSlisten, c.evmP2PListen = a1, a2, a3, a4, a5
	args := []string{
		"--listen=" + a1,
		"--rpclisten=" + a2,
		fmt.Sprintf(`--evmenv="--http --http.port=%s --ws --ws.port=%s --port=%s"`, a3, a4, a5),
		"--rpcuser=testuser",
		"--rpcpass=testpass",
		"--appdata=test",
		"--datadir=test/data",
		"--logdir=test/log",
	}
	if !reflect.DeepEqual(args, c.args()) {
		t.Errorf("failed to create node, expect %v but got %v", args, c.args())
	}
}
