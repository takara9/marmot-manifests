package etcdtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrFound            = errors.New("found")
	ErrUpdateConflict   = errors.New("update conflict")
	ErrTooManyConflicts = errors.New("too many update conflicts")
	// ErrSkip は UpdateJSON の mutate から返すと、更新せずに正常終了する。
	ErrSkip = errors.New("skip update")
)

const maxUpdateRetries = 50

// Store は marmot の db.Database を改善するための試作です。
//
//   - 排他更新は ModRevision を使った CAS で行い、競合時は再読込して再試行する (UpdateJSON)
//   - 新規作成は存在しない場合のみ成功する (CreateJSON)
//   - 分散ロックは呼び出しごとに Mutex を返す (db.Database は共有フィールド d.Mutex を上書きする)
//   - Session が失効していたら再生成する
type Store struct {
	Cli *clientv3.Client
	ttl int

	mu      sync.Mutex
	session *concurrency.Session
}

// NewStore は Store を作成します。ttl はロック用セッションのリース TTL (秒)。
func NewStore(cli *clientv3.Client, ttl int) *Store {
	return &Store{Cli: cli, ttl: ttl}
}

// Close はセッションを閉じます。クライアントは閉じません。
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		_ = s.session.Close()
		s.session = nil
	}
}

// Session は有効なセッションを返す。失効していれば作り直す。
func (s *Store) Session(ctx context.Context) (*concurrency.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		select {
		case <-s.session.Done():
		default:
			return s.session, nil
		}
	}
	sess, err := concurrency.NewSession(s.Cli, concurrency.WithTTL(s.ttl), concurrency.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("create etcd session: %w", err)
	}
	// 失効前のセッションは破棄 (Close はリース失効済みでも問題ない)
	if s.session != nil {
		_ = s.session.Close()
	}
	s.session = sess
	return sess, nil
}

// Lock は分散ロックを取得する。ctx の期限までロックを待つ。
func (s *Store) Lock(ctx context.Context, name string) (*concurrency.Mutex, error) {
	sess, err := s.Session(ctx)
	if err != nil {
		return nil, err
	}
	m := concurrency.NewMutex(sess, name)
	if err := m.Lock(ctx); err != nil {
		return nil, fmt.Errorf("acquire lock %s: %w", name, err)
	}
	return m, nil
}

// Unlock はロックを解放する。
func (s *Store) Unlock(ctx context.Context, m *concurrency.Mutex) error {
	return m.Unlock(ctx)
}

// GetJSON は JSON を読み込み、値と ModRevision を返す。
func GetJSON[T any](ctx context.Context, cli *clientv3.Client, key string) (T, int64, error) {
	var v T
	resp, err := cli.Get(ctx, key)
	if err != nil {
		return v, 0, fmt.Errorf("etcd get: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return v, 0, ErrNotFound
	}
	if err := json.Unmarshal(resp.Kvs[0].Value, &v); err != nil {
		return v, 0, fmt.Errorf("json unmarshal: %w", err)
	}
	return v, resp.Kvs[0].ModRevision, nil
}

// PutJSON は無条件に上書きする (排他が不要な場合のみ使う)。
func (s *Store) PutJSON(ctx context.Context, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	if _, err := s.Cli.Put(ctx, key, string(b)); err != nil {
		return fmt.Errorf("etcd put: %w", err)
	}
	return nil
}

// CreateJSON は key が存在しない場合のみ作成する。存在すれば ErrFound。
func (s *Store) CreateJSON(ctx context.Context, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	resp, err := s.Cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, string(b))).
		Commit()
	if err != nil {
		return fmt.Errorf("etcd txn: %w", err)
	}
	if !resp.Succeeded {
		return ErrFound
	}
	return nil
}

// PutJSONCAS は ModRevision が expectedRev の時だけ更新する。違えば ErrUpdateConflict。
func (s *Store) PutJSONCAS(ctx context.Context, key string, expectedRev int64, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	resp, err := s.Cli.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", expectedRev)).
		Then(clientv3.OpPut(key, string(b))).
		Commit()
	if err != nil {
		return fmt.Errorf("etcd txn: %w", err)
	}
	if !resp.Succeeded {
		return ErrUpdateConflict
	}
	return nil
}

// UpdateJSON は読み込み → mutate → CAS 書き込みを、競合しなくなるまで繰り返す。
// 戻り値は競合による再試行回数。mutate が ErrSkip を返すと更新せずに ErrSkip を返す。
func UpdateJSON[T any](ctx context.Context, s *Store, key string, mutate func(*T) error) (int, error) {
	conflicts := 0
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		v, rev, err := GetJSON[T](ctx, s.Cli, key)
		if err != nil {
			return conflicts, err
		}
		if err := mutate(&v); err != nil {
			return conflicts, err
		}
		err = s.PutJSONCAS(ctx, key, rev, v)
		if err == nil {
			return conflicts, nil
		}
		if !errors.Is(err, ErrUpdateConflict) {
			return conflicts, err
		}
		conflicts++
		// 同時に再試行して競合し続けないよう、少しずらす
		select {
		case <-ctx.Done():
			return conflicts, ctx.Err()
		case <-time.After(time.Duration(rand.IntN(5)) * time.Millisecond):
		}
	}
	return conflicts, ErrTooManyConflicts
}
