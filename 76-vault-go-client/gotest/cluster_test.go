package vaultclient

import (
	"context"
	"os"
	"testing"
	"time"
)

// アクティブノードがステップダウンしても、別のノードが引き継ぎ、読み書きを継続できること。
// リーダー交代で数秒間リクエストが失敗するため、VAULT_TEST_STEPDOWN=1 を指定したときだけ実行する。
func TestLeaderStepDown(t *testing.T) {
	if os.Getenv("VAULT_TEST_STEPDOWN") != "1" {
		t.Skip("set VAULT_TEST_STEPDOWN=1 to run (the active node steps down)")
	}
	cfg := testConfig(t)
	if len(cfg.Addrs) < 2 {
		t.Skip("needs a cluster")
	}
	ctx := testCtx(t)
	admin := newClient(t)
	store, mount := newKVStore(t, admin)
	if _, err := store.Put(ctx, "before", map[string]any{"v": "1"}); err != nil {
		t.Fatal(err)
	}

	oldActive := activeAddr(ctx, t, cfg)
	t.Logf("active before step-down: %s", oldActive)
	if err := clientForAddr(t, oldActive).Sys().StepDownWithContext(ctx); err != nil {
		t.Fatalf("step-down: %v", err)
	}

	var newActive string
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if a := activeAddrOrEmpty(ctx, t, cfg); a != "" && a != oldActive {
			newActive = a
			break
		}
		time.Sleep(time.Second)
	}
	if newActive == "" {
		t.Fatal("no new active node within 30s")
	}
	t.Logf("active after step-down: %s", newActive)

	// 再接続したクライアントで、既存データを読め、新しく書き込める
	c, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	s := NewStore(c, mount)
	if got, err := s.Get(ctx, "before"); err != nil || got.Data["v"] != "1" {
		t.Errorf("read data written before step-down: %v, %v", got, err)
	}
	if _, err := s.Put(ctx, "after", map[string]any{"v": "2"}); err != nil {
		t.Errorf("write after step-down: %v", err)
	}
	for _, addr := range cfg.Addrs {
		if got, err := NewStore(clientForAddr(t, addr), mount).Get(ctx, "after"); err != nil || got.Data["v"] != "2" {
			t.Errorf("read via %s after step-down: %v, %v", addr, got, err)
		}
	}
}

func activeAddr(ctx context.Context, t *testing.T, cfg Config) string {
	t.Helper()
	a := activeAddrOrEmpty(ctx, t, cfg)
	if a == "" {
		t.Fatal("no active node")
	}
	return a
}

func activeAddrOrEmpty(ctx context.Context, t *testing.T, cfg Config) string {
	t.Helper()
	for _, addr := range cfg.Addrs {
		c, err := NewClientForAddr(cfg, addr)
		if err != nil {
			continue
		}
		c.SetMaxRetries(0)
		if h, err := c.Sys().HealthWithContext(ctx); err == nil && h.Initialized && !h.Sealed && !h.Standby {
			return addr
		}
	}
	return ""
}
