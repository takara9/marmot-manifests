package vaultclient

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

// サービスごとに、必要なパスだけを読めるトークンを発行できること (最小権限)。
func TestPolicyLeastPrivilege(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	adminStore, mount := newKVStore(t, admin)

	for _, p := range []string{"app1/db", "app2/db"} {
		if _, err := adminStore.Put(ctx, p, map[string]any{"password": "secret-of-" + p}); err != nil {
			t.Fatal(err)
		}
	}
	policy := newPolicy(t, admin, `
path "`+mount+`/data/app1/*"     { capabilities = ["read"] }
path "`+mount+`/metadata/app1/*" { capabilities = ["read", "list"] }
`)
	app1 := NewStore(tokenClient(t, admin, policy), mount)

	got, err := app1.Get(ctx, "app1/db")
	if err != nil || got.Data["password"] != "secret-of-app1/db" {
		t.Fatalf("app1 reads its own secret: %v, %v", got, err)
	}
	if _, err := app1.Get(ctx, "app2/db"); statusOf(err) != http.StatusForbidden {
		t.Errorf("app1 reads app2's secret: err = %v, want 403", err)
	}
	if _, err := app1.Put(ctx, "app1/db", map[string]any{"password": "hacked"}); statusOf(err) != http.StatusForbidden {
		t.Errorf("app1 writes (read-only policy): err = %v, want 403", err)
	}
	got, _ = adminStore.Get(ctx, "app1/db")
	if got.Data["password"] != "secret-of-app1/db" {
		t.Errorf("secret was modified by a read-only token: %v", got.Data)
	}
}

func newAppRole(t *testing.T, admin *api.Client, policy string, secretIDUses int) (mount, role string) {
	t.Helper()
	mount = newAuthMount(t, admin, "approle")
	role = "marmot-api"
	_, err := admin.Logical().WriteWithContext(testCtx(t), "auth/"+mount+"/role/"+role, map[string]any{
		"token_policies":     []string{policy},
		"token_ttl":          "5m",
		"token_max_ttl":      "10m",
		"secret_id_num_uses": secretIDUses,
		"secret_id_ttl":      "5m",
	})
	if err != nil {
		t.Fatalf("create approle: %v", err)
	}
	return mount, role
}

func approleIDs(t *testing.T, admin *api.Client, mount, role string) (roleID, secretID string) {
	t.Helper()
	ctx := testCtx(t)
	sec, err := admin.Logical().ReadWithContext(ctx, "auth/"+mount+"/role/"+role+"/role-id")
	if err != nil {
		t.Fatalf("read role-id: %v", err)
	}
	roleID = str(t, sec, "role_id")
	sec, err = admin.Logical().WriteWithContext(ctx, "auth/"+mount+"/role/"+role+"/secret-id", nil)
	if err != nil {
		t.Fatalf("generate secret-id: %v", err)
	}
	return roleID, str(t, sec, "secret_id")
}

// 固定のトークンを持たず、AppRole (role_id + 使い捨ての secret_id) で認証できること。
func TestAppRoleLogin(t *testing.T) {
	cfg := testConfig(t)
	ctx := testCtx(t)
	admin := newClient(t)
	adminStore, mount := newKVStore(t, admin)
	if _, err := adminStore.Put(ctx, "svc/config", map[string]any{"api_key": "k-123"}); err != nil {
		t.Fatal(err)
	}
	policy := newPolicy(t, admin, `path "`+mount+`/data/svc/*" { capabilities = ["read"] }`)
	am, role := newAppRole(t, admin, policy, 1)
	roleID, secretID := approleIDs(t, admin, am, role)

	c, auth, err := LoginAppRole(ctx, cfg, am, roleID, secretID)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	t.Cleanup(func() { _ = c.Auth().Token().RevokeSelf("") })
	if auth.Auth.LeaseDuration <= 0 || !auth.Auth.Renewable {
		t.Errorf("token lease=%ds renewable=%v", auth.Auth.LeaseDuration, auth.Auth.Renewable)
	}
	got, err := NewStore(c, mount).Get(ctx, "svc/config")
	if err != nil || got.Data["api_key"] != "k-123" {
		t.Fatalf("read with approle token: %v, %v", got, err)
	}
	if _, err := NewStore(c, mount).Put(ctx, "svc/config", map[string]any{"api_key": "x"}); statusOf(err) != http.StatusForbidden {
		t.Errorf("write with read-only approle token: err = %v, want 403", err)
	}

	// 使い捨ての secret_id は、2 回目のログインで拒否される
	if _, _, err := LoginAppRole(ctx, cfg, am, roleID, secretID); err == nil {
		t.Error("second login with a single-use secret_id succeeded")
	}
	// 新しい secret_id ならログインできるが、誤った role_id は拒否される
	_, fresh := approleIDs(t, admin, am, role)
	if _, _, err := LoginAppRole(ctx, cfg, am, "00000000-0000-0000-0000-000000000000", fresh); err == nil {
		t.Error("login with a wrong role_id succeeded")
	}
}

// WatchToken がトークンを更新し続け、最大 TTL に達したら ErrTokenExpired で再ログインを促すこと。
func TestTokenRenewWithWatcher(t *testing.T) {
	if testing.Short() {
		t.Skip("takes about 15 seconds")
	}
	admin := newClient(t)
	sec, err := admin.Auth().Token().CreateOrphanWithContext(testCtx(t), &api.TokenCreateRequest{
		Policies:       []string{"default"},
		TTL:            "6s",
		ExplicitMaxTTL: "14s",
		Renewable:      boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	c := clientForAddr(t, admin.Address())
	c.SetToken(sec.Auth.ClientToken)

	var renewals atomic.Int32
	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- WatchToken(testCtx(t), c, sec, 6, func(*api.RenewOutput) { renewals.Add(1) })
	}()
	select {
	case err := <-done:
		if !errors.Is(err, ErrTokenExpired) {
			t.Errorf("WatchToken returned %v, want ErrTokenExpired", err)
		}
	case <-time.After(40 * time.Second):
		t.Fatal("WatchToken did not return")
	}
	elapsed := time.Since(start)
	t.Logf("renewals=%d, token kept alive for %s (initial TTL 6s, max TTL 14s)", renewals.Load(), elapsed.Round(time.Second))
	if renewals.Load() < 1 {
		t.Error("token was never renewed")
	}
	if elapsed < 8*time.Second {
		t.Errorf("token lived only %s, longer than the initial 6s TTL was expected", elapsed)
	}
}

// ラップされたレスポンスは 1 回だけ取り出せること。VM の起動時などに、機密情報を安全に渡す用途。
func TestResponseWrapping(t *testing.T) {
	cfg := testConfig(t)
	ctx := testCtx(t)
	admin := newClient(t)
	store, mount := newKVStore(t, admin)
	if _, err := store.Put(ctx, "vm/bootstrap", map[string]any{"join_token": "abc123"}); err != nil {
		t.Fatal(err)
	}

	// ラップ付きで読むと、機密情報の代わりに、使い捨てのトークンだけが返る
	wrapper := newClient(t)
	wrapper.SetWrappingLookupFunc(func(string, string) string { return "60s" })
	wrapped, err := wrapper.Logical().ReadWithContext(ctx, mount+"/data/vm/bootstrap")
	if err != nil {
		t.Fatalf("read with wrapping: %v", err)
	}
	if wrapped.WrapInfo == nil || wrapped.WrapInfo.Token == "" {
		t.Fatalf("no wrap info: %+v", wrapped)
	}
	if wrapped.Data != nil {
		t.Errorf("wrapped response leaks data: %v", wrapped.Data)
	}

	// 受け取り側は、Vault のトークンを持たなくても、ラップトークンだけで取り出せる
	recv, err := NewClientForAddr(cfg, admin.Address())
	if err != nil {
		t.Fatal(err)
	}
	sec, err := recv.Logical().UnwrapWithContext(ctx, wrapped.WrapInfo.Token)
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}
	data, _ := sec.Data["data"].(map[string]any)
	if data["join_token"] != "abc123" {
		t.Errorf("unwrapped data = %v", sec.Data)
	}
	if _, err := recv.Logical().UnwrapWithContext(ctx, wrapped.WrapInfo.Token); err == nil {
		t.Error("second unwrap succeeded; a wrapping token must be single-use")
	}
}

func boolPtr(b bool) *bool { return &b }
