package etcdtest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Counter struct {
	N int `json:"n"`
}

// 接続先の構成 (メンバー数、全エンドポイントの到達性) を確認する。
func TestClusterMembers(t *testing.T) {
	tg := testTarget(t)
	cli := newClient(t)
	ctx := ctxTimeout(t, opTimeout())

	ml, err := cli.MemberList(ctx)
	if err != nil {
		t.Fatalf("member list: %v", err)
	}
	if len(ml.Members) != tg.Members {
		t.Fatalf("members = %d, want %d", len(ml.Members), tg.Members)
	}

	leaders := map[uint64]bool{}
	for _, ep := range tg.Endpoints {
		st, err := cli.Status(ctx, ep)
		if err != nil {
			t.Fatalf("status %s: %v", ep, err)
		}
		leaders[st.Leader] = true
		t.Logf("endpoint=%s version=%s leader=%x isLeader=%v", ep, st.Version, st.Leader, st.Leader == st.Header.MemberId)
	}
	if len(leaders) != 1 {
		t.Fatalf("endpoints report different leaders: %v", leaders)
	}
}

// 1 つのエンドポイントに書いた値が、全エンドポイントから (線形化可能な読み取りで) 見えること。
func TestReadFromEveryEndpoint(t *testing.T) {
	tg := testTarget(t)
	p := testPrefix(t)
	ctx := ctxTimeout(t, 3*opTimeout())

	w, err := NewClientWithEndpoints(tg, tg.Endpoints[:1])
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.Put(ctx, p+"k", "v1"); err != nil {
		t.Fatalf("put: %v", err)
	}

	for _, ep := range tg.Endpoints {
		r, err := NewClientWithEndpoints(tg, []string{ep})
		if err != nil {
			t.Fatal(err)
		}
		resp, err := r.Get(ctx, p+"k")
		_ = r.Close()
		if err != nil {
			t.Fatalf("get via %s: %v", ep, err)
		}
		if len(resp.Kvs) != 1 || string(resp.Kvs[0].Value) != "v1" {
			t.Fatalf("get via %s: unexpected %v", ep, resp.Kvs)
		}
	}
}

// 先頭のエンドポイントが落ちていても、他のエンドポイントで処理を継続できること。
func TestEndpointFailover(t *testing.T) {
	tg := testTarget(t)
	p := testPrefix(t)
	ctx := ctxTimeout(t, 3*opTimeout())

	eps := append([]string{"https://127.0.0.1:1"}, tg.Endpoints...)
	cli, err := NewClientWithEndpoints(tg, eps)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	for i := 0; i < 5; i++ {
		if _, err := cli.Put(ctx, p+"k", "v"); err != nil {
			t.Fatalf("put #%d with a dead endpoint: %v", i, err)
		}
	}
}

// 複数キーを 1 つのトランザクションで原子的に更新できること。
func TestTxnMultiKey(t *testing.T) {
	p := testPrefix(t)
	cli := newClient(t)
	ctx := ctxTimeout(t, opTimeout())
	a, b := p+"a", p+"b"

	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(a), "=", 0), clientv3.Compare(clientv3.CreateRevision(b), "=", 0)).
		Then(clientv3.OpPut(a, "1"), clientv3.OpPut(b, "1")).
		Commit()
	if err != nil || !resp.Succeeded {
		t.Fatalf("txn failed: %v succeeded=%v", err, resp != nil && resp.Succeeded)
	}
	ga, _ := cli.Get(ctx, a)
	gb, _ := cli.Get(ctx, b)
	if ga.Kvs[0].ModRevision != gb.Kvs[0].ModRevision {
		t.Fatalf("keys were not updated in one revision: %d vs %d", ga.Kvs[0].ModRevision, gb.Kvs[0].ModRevision)
	}

	// 条件を満たさない場合は、どちらも更新されない
	resp, err = cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(a), "=", 0)).
		Then(clientv3.OpPut(a, "2"), clientv3.OpPut(b, "2")).
		Commit()
	if err != nil || resp.Succeeded {
		t.Fatalf("txn must fail: err=%v", err)
	}
	gb, _ = cli.Get(ctx, b)
	if string(gb.Kvs[0].Value) != "1" {
		t.Fatalf("b was modified by a failed txn: %s", gb.Kvs[0].Value)
	}
}

// 古い ModRevision での更新は拒否されること (marmot の PutJSONCAS 相当)。
func TestStaleCASRejected(t *testing.T) {
	p := testPrefix(t)
	s := newStore(t, 10)
	ctx := ctxTimeout(t, opTimeout())
	key := p + "obj"

	if err := s.CreateJSON(ctx, key, Counter{N: 1}); err != nil {
		t.Fatal(err)
	}
	_, staleRev, err := GetJSON[Counter](ctx, s.Cli, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PutJSON(ctx, key, Counter{N: 2}); err != nil { // 別のクライアントが更新した想定
		t.Fatal(err)
	}
	if err := s.PutJSONCAS(ctx, key, staleRev, Counter{N: 99}); !errors.Is(err, ErrUpdateConflict) {
		t.Fatalf("stale CAS err = %v, want ErrUpdateConflict", err)
	}
	got, _, _ := GetJSON[Counter](ctx, s.Cli, key)
	if got.N != 2 {
		t.Fatalf("value = %d, want 2", got.N)
	}
}

// 同じキーを複数クライアントが同時に作成しても、成功するのは 1 つだけであること。
func TestCreateIfAbsentExclusive(t *testing.T) {
	p := testPrefix(t)
	key := p + "owner"
	const clients = 8

	var wins, found atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		s := newStore(t, 10)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), opTimeout())
			defer cancel()
			switch err := s.CreateJSON(ctx, key, Counter{N: i}); {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, ErrFound):
				found.Add(1)
			default:
				t.Errorf("create: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 || found.Load() != clients-1 {
		t.Fatalf("wins=%d found=%d, want 1/%d", wins.Load(), found.Load(), clients-1)
	}
}

// 複数クライアントからの同時更新 (カウンタの加算) で、更新が失われないこと。
func TestConcurrentUpdateWithCAS(t *testing.T) {
	p := testPrefix(t)
	key := p + "counter"
	const clients, perClient = 5, 20

	init := newStore(t, 10)
	if err := init.CreateJSON(ctxTimeout(t, opTimeout()), key, Counter{}); err != nil {
		t.Fatal(err)
	}

	var conflicts atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		s := newStore(t, 10)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perClient; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), opTimeout())
				c, err := UpdateJSON(ctx, s, key, func(v *Counter) error { v.N++; return nil })
				cancel()
				if err != nil {
					t.Errorf("update: %v", err)
					return
				}
				conflicts.Add(int64(c))
			}
		}()
	}
	wg.Wait()

	got, _, err := GetJSON[Counter](ctxTimeout(t, opTimeout()), init.Cli, key)
	if err != nil {
		t.Fatal(err)
	}
	if got.N != clients*perClient {
		t.Fatalf("counter = %d, want %d (lost updates)", got.N, clients*perClient)
	}
	t.Logf("counter=%d conflicts(retried)=%d", got.N, conflicts.Load())
}

// 比較のため、CAS なしの読み込み → 書き込みでは更新が失われることを観察する (失敗にはしない)。
func TestLostUpdateWithoutCAS(t *testing.T) {
	p := testPrefix(t)
	key := p + "counter"
	const clients, perClient = 5, 20

	init := newStore(t, 10)
	if err := init.CreateJSON(ctxTimeout(t, opTimeout()), key, Counter{}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		s := newStore(t, 10)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perClient; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), opTimeout())
				v, _, err := GetJSON[Counter](ctx, s.Cli, key)
				if err == nil {
					v.N++
					err = s.PutJSON(ctx, key, v)
				}
				cancel()
				if err != nil {
					t.Errorf("naive update: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	got, _, _ := GetJSON[Counter](ctxTimeout(t, opTimeout()), init.Cli, key)
	t.Logf("without CAS: counter=%d (expected %d), lost=%d", got.N, clients*perClient, clients*perClient-got.N)
}

// 分散ロックで、複数クライアントのクリティカルセクションが同時に実行されないこと。
func TestDistributedMutexExclusion(t *testing.T) {
	p := testPrefix(t)
	lockName, key := p+"lock", p+"counter"
	const clients, perClient = 4, 5

	init := newStore(t, 10)
	if err := init.CreateJSON(ctxTimeout(t, opTimeout()), key, Counter{}); err != nil {
		t.Fatal(err)
	}

	var inside, maxInside atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		s := newStore(t, 10)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perClient; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				m, err := s.Lock(ctx, lockName)
				if err != nil {
					cancel()
					t.Errorf("lock: %v", err)
					return
				}
				n := inside.Add(1)
				for {
					old := maxInside.Load()
					if n <= old || maxInside.CompareAndSwap(old, n) {
						break
					}
				}
				// ロックなしでは更新が失われる読み書き
				v, _, err := GetJSON[Counter](ctx, s.Cli, key)
				if err == nil {
					time.Sleep(10 * time.Millisecond)
					v.N++
					err = s.PutJSON(ctx, key, v)
				}
				inside.Add(-1)
				if uerr := s.Unlock(ctx, m); uerr != nil {
					t.Errorf("unlock: %v", uerr)
				}
				cancel()
				if err != nil {
					t.Errorf("critical section: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	got, _, _ := GetJSON[Counter](ctxTimeout(t, opTimeout()), init.Cli, key)
	if maxInside.Load() != 1 {
		t.Fatalf("max concurrent holders = %d, want 1", maxInside.Load())
	}
	if got.N != clients*perClient {
		t.Fatalf("counter = %d, want %d", got.N, clients*perClient)
	}
}

// ロックを保持している間、他のクライアントは待たされ、期限で諦められること。
func TestLockTimeout(t *testing.T) {
	p := testPrefix(t)
	name := p + "lock"
	a, b := newStore(t, 10), newStore(t, 10)

	m, err := a.Lock(ctxTimeout(t, opTimeout()), name)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Unlock(context.Background(), m)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if _, err := b.Lock(ctx, name); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock while held: err = %v, want deadline exceeded", err)
	}
}

// ロックを持つクライアントが異常終了 (リース失効) したら、他のクライアントがロックを取得できること。
func TestLockReleasedWhenHolderDies(t *testing.T) {
	p := testPrefix(t)
	name := p + "lock"
	holder, waiter := newStore(t, 10), newStore(t, 10)

	if _, err := holder.Lock(ctxTimeout(t, opTimeout()), name); err != nil {
		t.Fatal(err)
	}
	got := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, err := waiter.Lock(ctx, name)
		got <- err
	}()

	time.Sleep(300 * time.Millisecond)
	holder.Close() // セッションを閉じる = リースが失効し、ロックキーが消える
	if err := <-got; err != nil {
		t.Fatalf("waiter did not acquire the lock after the holder died: %v", err)
	}
}

// セッション (リース) が失効しても、Store が作り直してロックを取得できること。
// marmot の db.Database.ensureLiveSession と同じ狙い。
func TestSessionRecreatedAfterLeaseLoss(t *testing.T) {
	p := testPrefix(t)
	s := newStore(t, 10)
	ctx := ctxTimeout(t, 2*opTimeout())

	old, err := s.Session(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cli.Revoke(ctx, old.Lease()); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	select {
	case <-old.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not expire after the lease was revoked")
	}

	m, err := s.Lock(ctx, p+"lock")
	if err != nil {
		t.Fatalf("lock after lease loss: %v", err)
	}
	cur, _ := s.Session(ctx)
	if cur == old {
		t.Fatal("session was not recreated")
	}
	if err := s.Unlock(ctx, m); err != nil {
		t.Fatal(err)
	}
}
