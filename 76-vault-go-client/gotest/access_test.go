package vaultclient

import (
	"errors"
	"testing"
	"time"
)

// 全ノードが初期化済み・unseal 済みで、アクティブが 1 つ、残りがスタンバイであること。
// 75-vault-cluster の自動 unseal が働いていることの確認にもなる。
func TestClusterHealth(t *testing.T) {
	cfg := testConfig(t)
	ctx := testCtx(t)

	active := 0
	clusterIDs := map[string]bool{}
	for _, addr := range cfg.Addrs {
		c, err := NewClientForAddr(cfg, addr)
		if err != nil {
			t.Fatalf("client for %s: %v", addr, err)
		}
		h, err := c.Sys().HealthWithContext(ctx)
		if err != nil {
			t.Fatalf("health of %s: %v", addr, err)
		}
		t.Logf("%s: version=%s initialized=%v sealed=%v standby=%v", addr, h.Version, h.Initialized, h.Sealed, h.Standby)
		if !h.Initialized || h.Sealed {
			t.Errorf("%s is not available (initialized=%v sealed=%v)", addr, h.Initialized, h.Sealed)
		}
		if !h.Standby {
			active++
		}
		clusterIDs[h.ClusterID] = true
	}
	if active != 1 {
		t.Errorf("active nodes = %d, want 1", active)
	}
	if len(clusterIDs) != 1 {
		t.Errorf("nodes belong to %d different clusters: %v", len(clusterIDs), clusterIDs)
	}
	if len(cfg.Addrs) != cfg.Nodes {
		t.Logf("note: %d addresses given, expected %d nodes", len(cfg.Addrs), cfg.Nodes)
	}

	if cfg.Nodes < 2 {
		return
	}
	// Raft のメンバーに、全ノードが投票権付きで参加していること
	sec, err := newClient(t).Logical().ReadWithContext(ctx, "sys/storage/raft/configuration")
	if err != nil {
		t.Fatalf("read raft configuration: %v", err)
	}
	cfgData, _ := sec.Data["config"].(map[string]any)
	servers, _ := cfgData["servers"].([]any)
	voters, leaders := 0, 0
	for _, s := range servers {
		m, _ := s.(map[string]any)
		if v, _ := m["voter"].(bool); v {
			voters++
		}
		if l, _ := m["leader"].(bool); l {
			leaders++
		}
	}
	if len(servers) != cfg.Nodes || voters != cfg.Nodes || leaders != 1 {
		t.Errorf("raft: servers=%d voters=%d leaders=%d, want %d/%d/1", len(servers), voters, leaders, cfg.Nodes, cfg.Nodes)
	}
}

// 先頭のノードに接続できなくても、次のノードへ切り替えて接続できること。
func TestEndpointFailover(t *testing.T) {
	cfg := testConfig(t)
	ctx := testCtx(t)

	dead := []string{"https://127.0.0.1:1", "https://127.0.0.1:2"}
	failover := cfg
	failover.Addrs = append(append([]string{}, dead...), cfg.Addrs...)
	start := time.Now()
	c, err := Connect(ctx, failover)
	if err != nil {
		t.Fatalf("connect with dead nodes first: %v", err)
	}
	t.Logf("failover took %s, connected to %s", time.Since(start).Round(time.Millisecond), c.Address())
	if _, err := c.Auth().Token().LookupSelfWithContext(ctx); err != nil {
		t.Errorf("token lookup on failed-over client: %v", err)
	}

	allDead := cfg
	allDead.Addrs = dead
	if _, err := Connect(ctx, allDead); !errors.Is(err, ErrNoHealthyNode) {
		t.Errorf("connect to dead nodes: err = %v, want ErrNoHealthyNode", err)
	}
}

// 1 つのノードに書いた値が全ノードから読め、スタンバイ経由の書き込みもアクティブに届くこと。
func TestReadWriteViaEveryNode(t *testing.T) {
	cfg := testConfig(t)
	ctx := testCtx(t)
	store, mount := newKVStore(t, newClient(t))

	for i, writer := range cfg.Addrs {
		path := "node" + string(rune('0'+i))
		ws := NewStore(clientForAddr(t, writer), mount)
		if _, err := ws.Put(ctx, path, map[string]any{"writer": writer}); err != nil {
			t.Fatalf("write via %s: %v", writer, err)
		}
		for _, reader := range cfg.Addrs {
			sec, err := NewStore(clientForAddr(t, reader), mount).Get(ctx, path)
			if err != nil {
				t.Fatalf("read via %s: %v", reader, err)
			}
			if sec.Data["writer"] != writer {
				t.Errorf("read via %s: writer = %v, want %s", reader, sec.Data["writer"], writer)
			}
		}
	}
	keys, err := store.List(ctx, "")
	if err != nil || len(keys) != len(cfg.Addrs) {
		t.Errorf("List = %v, %v; want %d keys", keys, err, len(cfg.Addrs))
	}
}
