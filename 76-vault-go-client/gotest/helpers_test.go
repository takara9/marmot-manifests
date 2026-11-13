package vaultclient

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

var runSeq atomic.Int64

var runID = strconv.FormatInt(time.Now().Unix(), 36)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// cleanupCtx は t.Cleanup 内で使う。テストが失敗・タイムアウトしても後片付けを実行できる。
func cleanupCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

func testConfig(t *testing.T) Config {
	t.Helper()
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Token == "" {
		t.Fatalf("no token: set %s (root token printed when the cluster was initialized) or run `vault login`", EnvToken)
	}
	return cfg
}

// newClient は独立した接続を持つクライアントを作る。複数プロセスからのアクセスを模擬するために使う。
func newClient(t *testing.T) *api.Client {
	t.Helper()
	c, err := Connect(testCtx(t), testConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return c
}

// clientForAddr は指定したノードだけに接続するクライアントを作る。
func clientForAddr(t *testing.T, addr string) *api.Client {
	t.Helper()
	cfg := testConfig(t)
	c, err := NewClientForAddr(cfg, addr)
	if err != nil {
		t.Fatalf("client for %s: %v", addr, err)
	}
	c.SetToken(cfg.Token)
	return c
}

// newMount はテスト専用のパスにシークレットエンジンを有効化し、終了時に無効化する。
// 既存のマウント (secret/ など) には触れない。
func newMount(t *testing.T, c *api.Client, typ string, opts map[string]string) string {
	t.Helper()
	path := fmt.Sprintf("vct-%s-%d", runID, runSeq.Add(1))
	err := c.Sys().MountWithContext(testCtx(t), path, &api.MountInput{Type: typ, Options: opts})
	if err != nil {
		t.Fatalf("mount %s at %s: %v", typ, path, err)
	}
	t.Cleanup(func() {
		ctx, cancel := cleanupCtx()
		defer cancel()
		_ = c.Sys().UnmountWithContext(ctx, path)
	})
	return path
}

func newKVStore(t *testing.T, c *api.Client) (*Store, string) {
	t.Helper()
	m := newMount(t, c, "kv", map[string]string{"version": "2"})
	return NewStore(c, m), m
}

// newAuthMount はテスト専用のパスに認証方式を有効化し、終了時に無効化する。
func newAuthMount(t *testing.T, c *api.Client, typ string) string {
	t.Helper()
	path := fmt.Sprintf("vct-%s-%d", runID, runSeq.Add(1))
	if err := c.Sys().EnableAuthWithOptionsWithContext(testCtx(t), path, &api.EnableAuthOptions{Type: typ}); err != nil {
		t.Fatalf("enable auth %s at %s: %v", typ, path, err)
	}
	t.Cleanup(func() {
		ctx, cancel := cleanupCtx()
		defer cancel()
		_ = c.Sys().DisableAuthWithContext(ctx, path)
	})
	return path
}

// newPolicy はテスト専用のポリシーを作成し、終了時に削除する。
func newPolicy(t *testing.T, c *api.Client, rules string) string {
	t.Helper()
	name := fmt.Sprintf("vct-%s-%d", runID, runSeq.Add(1))
	if err := c.Sys().PutPolicyWithContext(testCtx(t), name, rules); err != nil {
		t.Fatalf("put policy: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := cleanupCtx()
		defer cancel()
		_ = c.Sys().DeletePolicyWithContext(ctx, name)
	})
	return name
}

// tokenClient は、指定したポリシーだけを持つ短命のトークンで、別のクライアントを作る。
func tokenClient(t *testing.T, admin *api.Client, policies ...string) *api.Client {
	t.Helper()
	sec, err := admin.Auth().Token().CreateOrphanWithContext(testCtx(t), &api.TokenCreateRequest{
		Policies: policies,
		TTL:      "10m",
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	c := clientForAddr(t, admin.Address())
	c.SetToken(sec.Auth.ClientToken)
	t.Cleanup(func() {
		ctx, cancel := cleanupCtx()
		defer cancel()
		_ = admin.Auth().Token().RevokeAccessorWithContext(ctx, sec.Auth.Accessor)
	})
	return c
}

// statusOf は Vault の API エラーから HTTP ステータスコードを取り出す。該当しなければ 0。
func statusOf(err error) int {
	var re *api.ResponseError
	if errors.As(err, &re) {
		return re.StatusCode
	}
	return 0
}

func str(t *testing.T, sec *api.Secret, key string) string {
	t.Helper()
	if sec == nil || sec.Data == nil {
		t.Fatalf("empty response while reading %q", key)
	}
	v, ok := sec.Data[key].(string)
	if !ok {
		t.Fatalf("field %q is not a string: %#v", key, sec.Data[key])
	}
	return v
}

// newStores は、それぞれ独立した接続を持つ Store を n 個作る。goroutine の中では t.Fatal を呼べないため、事前に作る。
func newStores(t *testing.T, mount string, n int) []*Store {
	t.Helper()
	stores := make([]*Store, n)
	for i := range stores {
		stores[i] = NewStore(newClient(t), mount)
	}
	return stores
}
