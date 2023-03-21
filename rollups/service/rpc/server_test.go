package rpc

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"testing"

	"github.com/Qitmeer/qit/rpc"
	"github.com/stretchr/testify/require"
)

type testAPI struct{}

func (t *testAPI) Frobnicate(n int) int {
	return n * 2
}

func TestBaseServer(t *testing.T) {
	appVersion := "test"
	server := NewServer(
		"127.0.0.1",
		10000+rand.Intn(22768),
		appVersion,
		WithAPIs([]rpc.API{
			{
				Namespace: "test",
				Service:   new(testAPI),
			},
		}),
	)
	require.NoError(t, server.Start())
	defer func() {
		_ = server.Stop()
	}()

	rpcClient, err := rpc.Dial(fmt.Sprintf("http://%s", server.endpoint))
	require.NoError(t, err)

	t.Run("supports GET /healthz", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("http://%s/healthz", server.endpoint))
		require.NoError(t, err)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.EqualValues(t, fmt.Sprintf("{\"version\":\"%s\"}\n", appVersion), string(body))
	})

	t.Run("supports health_status", func(t *testing.T) {
		var res string
		require.NoError(t, rpcClient.Call(&res, "health_status"))
		require.Equal(t, appVersion, res)
	})

	t.Run("supports additional RPC APIs", func(t *testing.T) {
		var res int
		require.NoError(t, rpcClient.Call(&res, "test_frobnicate", 2))
		require.Equal(t, 4, res)
	})
}
