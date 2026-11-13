package etcdtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// prefix 配下の PUT / DELETE が、発生した順序どおりに通知され、処理できること。
func TestWatchPrefixEvents(t *testing.T) {
	p := testPrefix(t)
	cli := newClient(t)
	ctx := ctxTimeout(t, 2*opTimeout())

	marker, err := cli.Put(ctx, p+"marker", "x")
	if err != nil {
		t.Fatal(err)
	}

	wctx, stop := context.WithCancel(ctx)
	defer stop()
	events := make(chan Event, 16)
	done := make(chan error, 1)
	go func() {
		done <- WatchPrefix(wctx, newClient(t), p, marker.Header.Revision+1, nil, func(e Event) error {
			events <- e
			return nil
		})
	}()

	steps := []struct {
		op  string
		key string
	}{{"put", "k1"}, {"put", "k2"}, {"put", "k1"}, {"del", "k2"}}
	for _, s := range steps {
		if s.op == "put" {
			_, err = cli.Put(ctx, p+s.key, "v")
		} else {
			_, err = cli.Delete(ctx, p+s.key)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	cache := map[string]string{}
	var lastRev int64
	for i, s := range steps {
		select {
		case e := <-events:
			wantType := mvccpb.PUT
			if s.op == "del" {
				wantType = mvccpb.DELETE
			}
			if e.Type != wantType || e.Key != p+s.key {
				t.Fatalf("event #%d = %v %s, want %v %s", i, e.Type, e.Key, wantType, p+s.key)
			}
			if e.ModRevision < lastRev {
				t.Fatalf("event #%d is out of order", i)
			}
			lastRev = e.ModRevision
			if e.Type == mvccpb.PUT {
				cache[e.Key] = string(e.Value)
			} else {
				delete(cache, e.Key)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("event #%d not received", i)
		}
	}
	// marker は監視開始前の更新なので、キャッシュには k1 だけが残る
	if len(cache) != 1 || cache[p+"k1"] != "v" {
		t.Fatalf("cache = %v", cache)
	}
	stop()
	if err := <-done; err != nil {
		t.Fatalf("watch: %v", err)
	}
}

// 監視が途切れている間の更新を、最後に処理したリビジョンの次から再開して取りこぼさないこと。
func TestWatchResumeFromRevision(t *testing.T) {
	p := testPrefix(t)
	cli := newClient(t)
	ctx := ctxTimeout(t, 2*opTimeout())

	wctx, cancel := context.WithCancel(ctx)
	ch := cli.Watch(wctx, p, clientv3.WithPrefix())
	if _, err := cli.Put(ctx, p+"a", "1"); err != nil {
		t.Fatal(err)
	}
	var last int64
	select {
	case wr := <-ch:
		last = wr.Events[len(wr.Events)-1].Kv.ModRevision
	case <-time.After(5 * time.Second):
		t.Fatal("first event not received")
	}
	cancel() // 監視が切れた状態

	for _, k := range []string{"b", "c", "d"} {
		if _, err := cli.Put(ctx, p+k, "x"); err != nil {
			t.Fatal(err)
		}
	}

	rctx, rcancel := context.WithCancel(ctx)
	defer rcancel()
	var got []string
	for wr := range cli.Watch(rctx, p, clientv3.WithPrefix(), clientv3.WithRev(last+1)) {
		for _, ev := range wr.Events {
			got = append(got, string(ev.Kv.Key)[len(p):])
		}
		if len(got) >= 3 {
			rcancel()
		}
	}
	if fmt.Sprint(got) != "[b c d]" {
		t.Fatalf("resumed events = %v, want [b c d]", got)
	}
}

type Job struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Worker string `json:"worker,omitempty"`
}

// 更新監視と排他更新の組み合わせ: 複数のワーカーが同じ prefix を監視し、
// 各ジョブを必ず 1 つのワーカーだけが CAS で確保して処理すること。
func TestWatchJobQueueExclusiveClaim(t *testing.T) {
	p := testPrefix(t)
	producer := newStore(t, 10)
	const jobs, workers = 12, 3

	var processed atomic.Int32
	perJob := make([]atomic.Int32, jobs)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		s := newStore(t, 10)
		name := fmt.Sprintf("worker-%d", w)

		claim := func(key string) error {
			var job Job
			uctx, ucancel := context.WithTimeout(ctx, opTimeout())
			defer ucancel()
			_, err := UpdateJSON(uctx, s, key, func(j *Job) error {
				if j.Status != "pending" {
					return ErrSkip
				}
				j.Status, j.Worker = "running", name
				job = *j
				return nil
			})
			if errors.Is(err, ErrSkip) || ctx.Err() != nil {
				return nil
			}
			if err != nil {
				return err
			}
			perJob[job.ID].Add(1)
			processed.Add(1)
			time.Sleep(20 * time.Millisecond) // 処理
			_, err = UpdateJSON(uctx, s, key, func(j *Job) error { j.Status = "done"; return nil })
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WatchPrefix(ctx, s.Cli, p, 0,
				func(kvs []*mvccpb.KeyValue) error {
					for _, kv := range kvs {
						if err := claim(string(kv.Key)); err != nil {
							return err
						}
					}
					return nil
				},
				func(e Event) error {
					if e.Type != mvccpb.PUT {
						return nil
					}
					return claim(e.Key)
				})
			if err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}()
	}

	time.Sleep(500 * time.Millisecond) // ワーカーの監視開始を待つ
	for i := 0; i < jobs; i++ {
		if err := producer.CreateJSON(ctxTimeout(t, opTimeout()), fmt.Sprintf("%sjob-%03d", p, i), Job{ID: i, Status: "pending"}); err != nil {
			t.Fatal(err)
		}
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := producer.Cli.Get(ctxTimeout(t, opTimeout()), p, clientv3.WithPrefix())
		if err != nil {
			t.Fatal(err)
		}
		done := 0
		for _, kv := range resp.Kvs {
			var j Job
			if err := json.Unmarshal(kv.Value, &j); err == nil && j.Status == "done" {
				done++
			}
		}
		if done == jobs {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("jobs done = %d/%d", done, jobs)
		}
		time.Sleep(200 * time.Millisecond)
	}
	cancel()
	wg.Wait()

	for i := range perJob {
		if n := perJob[i].Load(); n != 1 {
			t.Errorf("job %d was processed %d times, want 1", i, n)
		}
	}
	if processed.Load() != jobs {
		t.Fatalf("processed = %d, want %d", processed.Load(), jobs)
	}

	byWorker := map[string]int{}
	resp, _ := producer.Cli.Get(ctxTimeout(t, opTimeout()), p, clientv3.WithPrefix())
	for _, kv := range resp.Kvs {
		var j Job
		if err := json.Unmarshal(kv.Value, &j); err == nil {
			byWorker[j.Worker]++
		}
	}
	t.Logf("jobs per worker: %v", byWorker)
}

// リース付きキーが TTL で消え、削除が監視で検知されること。KeepAlive の間は消えないこと。
func TestLeaseExpiryAndKeepAlive(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test")
	}
	p := testPrefix(t)
	cli := newClient(t)
	ctx := ctxTimeout(t, 40*time.Second)
	key := p + "ephemeral"

	lease, err := cli.Grant(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Put(ctx, key, "alive", clientv3.WithLease(lease.ID)); err != nil {
		t.Fatal(err)
	}

	kctx, kcancel := context.WithCancel(ctx)
	if _, err := cli.KeepAlive(kctx, lease.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(6 * time.Second) // TTL の 2 倍待っても KeepAlive で残る
	if resp, _ := cli.Get(ctx, key); len(resp.Kvs) != 1 {
		t.Fatal("key disappeared while KeepAlive was running")
	}

	wctx, wcancel := context.WithCancel(ctx)
	defer wcancel()
	deleted := make(chan struct{})
	ch := cli.Watch(wctx, key)
	go func() {
		for wr := range ch {
			for _, ev := range wr.Events {
				if ev.Type == mvccpb.DELETE {
					close(deleted)
					return
				}
			}
		}
	}()

	kcancel() // 更新を止める = クライアント停止を模擬
	select {
	case <-deleted:
	case <-time.After(15 * time.Second):
		t.Fatal("expiry (DELETE) was not observed after KeepAlive stopped")
	}
}

// リーダー選出: 同時に 1 つだけがリーダーになり、Resign で順に交代すること。
func TestLeaderElection(t *testing.T) {
	p := testPrefix(t)
	const candidates = 3

	leaders := make(chan string, candidates)
	resign := make([]chan struct{}, candidates)
	var elections []*concurrency.Election
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	for i := 0; i < candidates; i++ {
		s := newStore(t, 10)
		sess, err := s.Session(ctx)
		if err != nil {
			t.Fatal(err)
		}
		e := concurrency.NewElection(sess, p+"leader")
		elections = append(elections, e)
		resign[i] = make(chan struct{})
		name := fmt.Sprintf("node-%d", i)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := e.Campaign(ctx, name); err != nil {
				t.Errorf("%s campaign: %v", name, err)
				return
			}
			leaders <- name
			<-resign[i]
			if err := e.Resign(ctx); err != nil {
				t.Errorf("%s resign: %v", name, err)
			}
		}()
		time.Sleep(300 * time.Millisecond) // 立候補の順序を固定する
	}

	var order []string
	for i := 0; i < candidates; i++ {
		select {
		case name := <-leaders:
			order = append(order, name)
		case <-time.After(10 * time.Second):
			t.Fatalf("leader #%d was not elected, order so far: %v", i, order)
		}
		// 現在のリーダーは常に 1 つだけで、全員から同じ結果が見える
		for _, e := range elections {
			resp, err := e.Leader(ctx)
			if err != nil {
				t.Fatalf("leader query: %v", err)
			}
			if string(resp.Kvs[0].Value) != order[i] {
				t.Fatalf("leader seen = %s, want %s", resp.Kvs[0].Value, order[i])
			}
		}
		close(resign[i])
	}
	wg.Wait()
	if !sort.StringsAreSorted(order) {
		t.Fatalf("leaders changed in an unexpected order: %v", order)
	}
}

// compaction で履歴が失われても、WatchPrefix が一覧を取り直して継続できること。
// etcd 全体の履歴を compaction するため、ETCD_TEST_COMPACT=1 のときだけ実行する。
func TestWatchRecoversFromCompaction(t *testing.T) {
	if os.Getenv("ETCD_TEST_COMPACT") != "1" {
		t.Skip("set ETCD_TEST_COMPACT=1 to run (compacts the etcd history)")
	}
	p := testPrefix(t)
	cli := newClient(t)
	ctx := ctxTimeout(t, 2*opTimeout())

	first, err := cli.Put(ctx, p+"k", "old")
	if err != nil {
		t.Fatal(err)
	}
	last, err := cli.Put(ctx, p+"k", "new")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Compact(ctx, last.Header.Revision); err != nil {
		t.Fatalf("compact: %v", err)
	}

	wctx, stop := context.WithCancel(ctx)
	defer stop()
	resynced := make(chan string, 1)
	go func() {
		_ = WatchPrefix(wctx, cli, p, first.Header.Revision, // 失われたリビジョンから開始
			func(kvs []*mvccpb.KeyValue) error {
				resynced <- string(kvs[0].Value)
				return nil
			},
			func(Event) error { return nil })
	}()

	select {
	case v := <-resynced:
		if v != "new" {
			t.Fatalf("resync value = %s, want new", v)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("resync was not triggered after compaction")
	}
}
