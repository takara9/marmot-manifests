package vaultclient

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// 文字列、複数行 (SSH 秘密鍵のような PEM)、日本語、ネストした値、大きな値を保存して取り出せること。
func TestKVPutGet(t *testing.T) {
	ctx := testCtx(t)
	store, _ := newKVStore(t, newClient(t))

	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmU=\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"
	big := strings.Repeat("x", 100*1024)
	in := map[string]any{
		"username":    "admin",
		"password":    "p@ss w0rd/\"quoted\"",
		"private_key": pem,
		"comment":     "マーモットの管理者パスワード",
		"nested":      map[string]any{"host": "db.labo.local", "port": "5432"},
		"enabled":     true,
		"big":         big,
	}
	put, err := store.Put(ctx, "marmot/db", in)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if put.VersionMetadata.Version != 1 {
		t.Errorf("first version = %d, want 1", put.VersionMetadata.Version)
	}

	got, err := store.Get(ctx, "marmot/db")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for _, k := range []string{"username", "password", "private_key", "comment", "big"} {
		if got.Data[k] != in[k] {
			t.Errorf("%s did not round-trip (got %d bytes)", k, len(got.Data[k].(string)))
		}
	}
	nested, _ := got.Data["nested"].(map[string]any)
	if nested["host"] != "db.labo.local" || nested["port"] != "5432" {
		t.Errorf("nested = %v", nested)
	}
	if got.Data["enabled"] != true {
		t.Errorf("enabled = %v", got.Data["enabled"])
	}

	if _, err := store.Get(ctx, "marmot/missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get missing: err = %v, want ErrNotFound", err)
	}
}

// 書き込みごとにバージョンが増え、古いバージョンを読めること。保持数を超えた古いバージョンは破棄されること。
func TestKVVersioning(t *testing.T) {
	ctx := testCtx(t)
	store, _ := newKVStore(t, newClient(t))

	for _, pw := range []string{"one", "two", "three"} {
		if _, err := store.Put(ctx, "app/pw", map[string]any{"password": pw}); err != nil {
			t.Fatal(err)
		}
	}
	old, err := store.GetVersion(ctx, "app/pw", 1)
	if err != nil || old.Data["password"] != "one" {
		t.Fatalf("version 1 = %v, %v", old, err)
	}
	cur, _ := store.Get(ctx, "app/pw")
	if cur.Data["password"] != "three" || cur.VersionMetadata.Version != 3 {
		t.Errorf("current = %v (version %d)", cur.Data, cur.VersionMetadata.Version)
	}

	if err := store.SetMaxVersions(ctx, "app/pw", 2); err != nil {
		t.Fatalf("set max versions: %v", err)
	}
	if _, err := store.Put(ctx, "app/pw", map[string]any{"password": "four"}); err != nil {
		t.Fatal(err)
	}
	vs, err := store.Versions(ctx, "app/pw")
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 || vs[0].Version != 3 || vs[1].Version != 4 {
		t.Errorf("retained versions = %+v, want [3 4]", vs)
	}
	if _, err := store.GetVersion(ctx, "app/pw", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("trimmed version 1: err = %v, want ErrNotFound", err)
	}
}

// 古いバージョン番号での更新 (CAS) が拒否されること。
func TestKVStaleCASRejected(t *testing.T) {
	ctx := testCtx(t)
	store, _ := newKVStore(t, newClient(t))

	if _, err := store.Put(ctx, "k", map[string]any{"v": "1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutCAS(ctx, "k", map[string]any{"v": "2"}, 1); err != nil {
		t.Fatalf("PutCAS with current version: %v", err)
	}
	_, err := store.PutCAS(ctx, "k", map[string]any{"v": "stale"}, 1)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("PutCAS with stale version: err = %v, want ErrConflict", err)
	}
	got, _ := store.Get(ctx, "k")
	if got.Data["v"] != "2" {
		t.Errorf("value = %v, want 2 (stale write must be rejected)", got.Data["v"])
	}
}

// 複数のクライアントが同じキーを作成しても、成功するのは 1 つだけであること。
func TestKVCreateExclusive(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	_, mount := newKVStore(t, admin)

	const n = 8
	stores := newStores(t, mount, n)
	var ok, exists atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := stores[i]
			_, err := s.Create(ctx, "lock/leader", map[string]any{"owner": i})
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, ErrAlreadyExists):
				exists.Add(1)
			default:
				t.Errorf("create: %v", err)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 || exists.Load() != n-1 {
		t.Errorf("created=%d already-exists=%d, want 1/%d", ok.Load(), exists.Load(), n-1)
	}
}

func counterOf(t *testing.T, cur map[string]any) int64 {
	t.Helper()
	if cur == nil {
		return 0
	}
	n, err := cur["count"].(json.Number).Int64()
	if err != nil {
		t.Errorf("count = %v: %v", cur["count"], err)
	}
	return n
}

// 5 クライアントが同時に加算しても、Update (CAS + 再試行) では更新が失われないこと。
func TestKVConcurrentUpdateWithCAS(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	store, mount := newKVStore(t, admin)

	const clients, each = 5, 10
	stores := newStores(t, mount, clients)
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := stores[i]
			for j := 0; j < each; j++ {
				_, err := s.Update(ctx, "counter", func(cur map[string]any) (map[string]any, error) {
					return map[string]any{"count": counterOf(t, cur) + 1}, nil
				})
				if err != nil {
					t.Errorf("update: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	got, err := store.Get(ctx, "counter")
	if err != nil {
		t.Fatal(err)
	}
	if n := counterOf(t, got.Data); n != clients*each {
		t.Errorf("count = %d, want %d (updates were lost)", n, clients*each)
	}
	t.Logf("final version = %d", got.VersionMetadata.Version)
}

// 比較用。CAS なしの読み書きでは、同時更新で更新が失われることを観察する (失敗にはしない)。
func TestKVLostUpdateWithoutCAS(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	store, mount := newKVStore(t, admin)

	const clients, each = 5, 10
	stores := newStores(t, mount, clients)
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := stores[i]
			for j := 0; j < each; j++ {
				var cur map[string]any
				if sec, err := s.Get(ctx, "counter"); err == nil {
					cur = sec.Data
				}
				if _, err := s.Put(ctx, "counter", map[string]any{"count": counterOf(t, cur) + 1}); err != nil {
					t.Errorf("put: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	got, _ := store.Get(ctx, "counter")
	t.Logf("without CAS: count = %d (expected %d if no update were lost)", counterOf(t, got.Data), clients*each)
}

// 論理削除、復元、完全破棄、全削除の動作。削除後でも Update でバージョンが続くこと。
func TestKVDeleteUndeleteDestroy(t *testing.T) {
	ctx := testCtx(t)
	store, _ := newKVStore(t, newClient(t))

	for _, v := range []string{"a", "b"} {
		if _, err := store.Put(ctx, "s", map[string]any{"v": v}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Delete(ctx, "s"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(ctx, "s"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete: err = %v, want ErrNotFound", err)
	}

	if err := store.Undelete(ctx, "s", 2); err != nil {
		t.Fatalf("undelete: %v", err)
	}
	if got, err := store.Get(ctx, "s"); err != nil || got.Data["v"] != "b" {
		t.Fatalf("get after undelete = %v, %v", got, err)
	}

	if err := store.Destroy(ctx, "s", 1); err != nil {
		t.Fatalf("destroy: %v", err)
	}
	vs, _ := store.Versions(ctx, "s")
	if len(vs) != 2 || !vs[0].Destroyed || vs[1].Destroyed {
		t.Errorf("versions after destroy = %+v", vs)
	}
	if _, err := store.GetVersion(ctx, "s", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("destroyed version: err = %v, want ErrNotFound", err)
	}

	// 最新バージョンが削除済みでも、Update はバージョン番号を引き継いで書き込める
	if err := store.Delete(ctx, "s"); err != nil {
		t.Fatal(err)
	}
	out, err := store.Update(ctx, "s", func(cur map[string]any) (map[string]any, error) {
		if cur != nil {
			t.Errorf("cur = %v, want nil for a deleted secret", cur)
		}
		return map[string]any{"v": "c"}, nil
	})
	if err != nil || out.VersionMetadata.Version != 3 {
		t.Fatalf("update after delete = %v, %v (want version 3)", out, err)
	}

	if err := store.DeleteAll(ctx, "s"); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	if _, err := store.Metadata(ctx, "s"); !errors.Is(err, ErrNotFound) {
		t.Errorf("metadata after delete all: err = %v, want ErrNotFound", err)
	}
}

func TestKVList(t *testing.T) {
	ctx := testCtx(t)
	store, _ := newKVStore(t, newClient(t))

	for _, p := range []string{"vms/web1", "vms/web2", "vms/db/primary", "other"} {
		if _, err := store.Put(ctx, p, map[string]any{"k": "v"}); err != nil {
			t.Fatal(err)
		}
	}
	keys, err := store.List(ctx, "vms")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(keys, ",") != "db/,web1,web2" {
		t.Errorf("List(vms) = %v, want [db/ web1 web2]", keys)
	}
	root, _ := store.List(ctx, "")
	if strings.Join(root, ",") != "other,vms/" {
		t.Errorf("List(root) = %v, want [other vms/]", root)
	}
	none, err := store.List(ctx, "nothing")
	if err != nil || len(none) != 0 {
		t.Errorf("List(nothing) = %v, %v; want empty", none, err)
	}
}
