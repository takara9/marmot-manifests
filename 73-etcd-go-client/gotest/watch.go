package etcdtest

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Event は prefix 配下のキー変更イベントです。
type Event struct {
	Type        mvccpb.Event_EventType
	Key         string
	Value       []byte
	ModRevision int64
}

// WatchPrefix は prefix 配下の変更を監視し、イベントごとに handle を呼ぶ。ctx が終わるまで戻らない。
//
//   - startRev が 0 の場合は、現在の一覧を取得して resync に渡し、その直後のリビジョンから監視する
//   - startRev が 0 以外の場合は、そのリビジョンから監視する (切断中の更新も取りこぼさない)
//   - 監視チャネルが切れた場合は、最後に処理したリビジョンの次から再接続する
//   - 履歴が compaction で失われた場合は、再度一覧を取得して resync に渡し、監視し直す
//
// handle がエラーを返すと監視を終了してそのエラーを返す。
func WatchPrefix(
	ctx context.Context,
	cli *clientv3.Client,
	prefix string,
	startRev int64,
	resync func(kvs []*mvccpb.KeyValue) error,
	handle func(Event) error,
) error {
	rev := startRev
	for {
		if rev == 0 {
			resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				if !sleepCtx(ctx, time.Second) {
					return nil
				}
				continue
			}
			if resync != nil {
				if err := resync(resp.Kvs); err != nil {
					return fmt.Errorf("resync: %w", err)
				}
			}
			rev = resp.Header.Revision + 1
		}

		next, err := watchOnce(ctx, cli, prefix, rev, handle)
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
		rev = next
		if rev == 0 {
			continue
		}
		if !sleepCtx(ctx, 500*time.Millisecond) {
			return nil
		}
	}
}

// watchOnce は 1 回分の Watch を処理し、次に再開すべきリビジョンを返す。0 は再取得が必要。
func watchOnce(ctx context.Context, cli *clientv3.Client, prefix string, rev int64, handle func(Event) error) (int64, error) {
	wctx, cancel := context.WithCancel(clientv3.WithRequireLeader(ctx))
	defer cancel()

	for wresp := range cli.Watch(wctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(rev)) {
		if wresp.CompactRevision != 0 {
			return 0, nil
		}
		if err := wresp.Err(); err != nil {
			return rev, nil
		}
		for _, ev := range wresp.Events {
			e := Event{
				Type:        ev.Type,
				Key:         string(ev.Kv.Key),
				Value:       ev.Kv.Value,
				ModRevision: ev.Kv.ModRevision,
			}
			if err := handle(e); err != nil {
				return rev, err
			}
			rev = ev.Kv.ModRevision + 1
		}
	}
	return rev, nil
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
