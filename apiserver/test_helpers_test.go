package apiserver

import (
	"stock/store"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestServer(t *testing.T) (*ApiServer, *redis.Client, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	server := &ApiServer{
		store: store.New(rdb),
	}

	return server, rdb, mr
}
