package etcdtest

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var runSeq atomic.Int64

var runID = fmt.Sprintf("%d", time.Now().UnixNano())

func opTimeout() time.Duration { return 10 * time.Second }

func testTarget(t *testing.T) Target {
	t.Helper()
	tg, err := ResolveTarget()
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	return tg
}

// newClient は独立した接続を持つクライアントを作る。複数プロセスからのアクセスを模擬するために使う。
func newClient(t *testing.T) *clientv3.Client {
	t.Helper()
	cli, err := NewClient(testTarget(t))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func newStore(t *testing.T, ttl int) *Store {
	t.Helper()
	s := NewStore(newClient(t), ttl)
	t.Cleanup(s.Close)
	return s
}

// testPrefix はテストごとに一意なキー prefix を返し、終了時に削除する。/marmot 配下には触れない。
func testPrefix(t *testing.T) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	p := fmt.Sprintf("/etcd-go-client-test/%s/%s/%d/", runID, name, runSeq.Add(1))
	cli := newClient(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), opTimeout())
		defer cancel()
		_, _ = cli.Delete(ctx, p, clientv3.WithPrefix())
	})
	return p
}

func ctxTimeout(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}
