package vaultclient

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

var (
	// ErrNotFound は、シークレットが存在しない (または最新バージョンが削除済み) ことを表します。
	ErrNotFound = api.ErrSecretNotFound
	// ErrConflict は、CAS (check-and-set) が現在のバージョンと一致しなかったことを表します。
	ErrConflict = errors.New("secret was modified concurrently")
	// ErrAlreadyExists は、Create の対象がすでに存在することを表します。
	ErrAlreadyExists = errors.New("secret already exists")
)

// maxUpdateRetries は Update が競合したときの再試行回数です。
const maxUpdateRetries = 50

// Store は KV v2 シークレットエンジンの 1 つのマウントを扱います。
type Store struct {
	client *api.Client
	mount  string
	kv     *api.KVv2
}

// NewStore は mount (例: "secret") の KV v2 を扱う Store を返します。
func NewStore(c *api.Client, mount string) *Store {
	return &Store{client: c, mount: mount, kv: c.KVv2(mount)}
}

// Put は無条件で新しいバージョンを書き込みます。
func (s *Store) Put(ctx context.Context, path string, data map[string]any) (*api.KVSecret, error) {
	return s.kv.Put(ctx, path, data)
}

// Get は最新バージョンを読み出します。存在しない、または削除済みなら ErrNotFound を返します。
func (s *Store) Get(ctx context.Context, path string) (*api.KVSecret, error) {
	return notFoundIfDeleted(s.kv.Get(ctx, path))
}

// GetVersion は指定バージョンを読み出します。存在しない、削除済み、破棄済みなら ErrNotFound を返します。
func (s *Store) GetVersion(ctx context.Context, path string, version int) (*api.KVSecret, error) {
	return notFoundIfDeleted(s.kv.GetVersion(ctx, path, version))
}

// notFoundIfDeleted は、削除済み・破棄済みのバージョンを ErrNotFound にそろえる。
// vault/api の KVv2 は、削除済みでもエラーにせず、Data が nil のシークレットを返すため、
// 呼び出し側が空のシークレットと取り違えないようにする。
func notFoundIfDeleted(sec *api.KVSecret, err error) (*api.KVSecret, error) {
	if err != nil {
		return nil, err
	}
	if sec.Data == nil {
		return nil, fmt.Errorf("%w: deleted or destroyed", ErrNotFound)
	}
	return sec, nil
}

// Create はキーが存在しない場合だけ書き込みます (cas=0)。
// 複数のクライアントが同時に作成しても、成功するのは 1 つだけです。
// 最新バージョンが削除済みでもバージョンは残っているため、ErrAlreadyExists になります。
func (s *Store) Create(ctx context.Context, path string, data map[string]any) (*api.KVSecret, error) {
	sec, err := s.kv.Put(ctx, path, data, api.WithCheckAndSet(0))
	if isCASConflict(err) {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, path)
	}
	return sec, err
}

// PutCAS は現在のバージョンが cas と一致する場合だけ書き込みます。
// 一致しなければ ErrConflict を返します。
func (s *Store) PutCAS(ctx context.Context, path string, data map[string]any, cas int) (*api.KVSecret, error) {
	sec, err := s.kv.Put(ctx, path, data, api.WithCheckAndSet(cas))
	if isCASConflict(err) {
		return nil, fmt.Errorf("%w: %s (cas=%d)", ErrConflict, path, cas)
	}
	return sec, err
}

// Update は、読み出し → fn による変更 → CAS 書き込みを、競合しなくなるまで繰り返します。
// キーが存在しない場合、fn には nil が渡され、新規作成します。
// 複数のクライアントが同じシークレットを更新しても、更新が失われません。
func (s *Store) Update(ctx context.Context, path string, fn func(cur map[string]any) (map[string]any, error)) (*api.KVSecret, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		var cur map[string]any
		version := 0
		sec, err := s.kv.Get(ctx, path)
		switch {
		case err == nil:
			cur = sec.Data
			version = sec.VersionMetadata.Version
		case errors.Is(err, ErrNotFound):
			// 最新バージョンが削除済みの場合、バージョン番号は残っているため、メタデータから取得する
			if version, err = s.currentVersion(ctx, path); err != nil {
				return nil, err
			}
		default:
			return nil, err
		}

		next, err := fn(cur)
		if err != nil {
			return nil, err
		}
		out, err := s.PutCAS(ctx, path, next, version)
		if err == nil {
			return out, nil
		}
		if !errors.Is(err, ErrConflict) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(1+rand.IntN(5*(attempt+1))) * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("%w: %s (gave up after %d retries)", ErrConflict, path, maxUpdateRetries)
}

func (s *Store) currentVersion(ctx context.Context, path string) (int, error) {
	md, err := s.kv.GetMetadata(ctx, path)
	if errors.Is(err, ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return md.CurrentVersion, nil
}

// Metadata はバージョン一覧などのメタデータを返します。
func (s *Store) Metadata(ctx context.Context, path string) (*api.KVMetadata, error) {
	return s.kv.GetMetadata(ctx, path)
}

// Versions は、各バージョンの作成・削除時刻を、バージョン順に返します。
func (s *Store) Versions(ctx context.Context, path string) ([]api.KVVersionMetadata, error) {
	return s.kv.GetVersionsAsList(ctx, path)
}

// SetMaxVersions は、保持するバージョン数を設定します。古いバージョンは自動的に破棄されます。
func (s *Store) SetMaxVersions(ctx context.Context, path string, n int) error {
	return s.kv.PutMetadata(ctx, path, api.KVMetadataPutInput{MaxVersions: n})
}

// Delete は最新バージョンを論理削除します (Undelete で復元できる)。
func (s *Store) Delete(ctx context.Context, path string) error {
	return s.kv.Delete(ctx, path)
}

// Undelete は論理削除したバージョンを復元します。
func (s *Store) Undelete(ctx context.Context, path string, versions ...int) error {
	return s.kv.Undelete(ctx, path, versions)
}

// Destroy は指定バージョンのデータを完全に破棄します (復元できない)。
func (s *Store) Destroy(ctx context.Context, path string, versions ...int) error {
	return s.kv.Destroy(ctx, path, versions)
}

// DeleteAll は、全バージョンとメタデータを完全に削除します。
func (s *Store) DeleteAll(ctx context.Context, path string) error {
	return s.kv.DeleteMetadata(ctx, path)
}

// List は path 直下のキーを返します。下位階層は末尾に "/" が付きます。
func (s *Store) List(ctx context.Context, path string) ([]string, error) {
	sec, err := s.client.Logical().ListWithContext(ctx, s.mount+"/metadata/"+strings.Trim(path, "/"))
	if err != nil {
		return nil, err
	}
	if sec == nil || sec.Data == nil {
		return nil, nil
	}
	raw, _ := sec.Data["keys"].([]any)
	keys := make([]string, 0, len(raw))
	for _, k := range raw {
		if ks, ok := k.(string); ok {
			keys = append(keys, ks)
		}
	}
	return keys, nil
}

// isCASConflict は、CAS の不一致によるエラーかどうかを判定します。
func isCASConflict(err error) bool {
	var re *api.ResponseError
	if !errors.As(err, &re) || re.StatusCode != http.StatusBadRequest {
		return false
	}
	for _, e := range re.Errors {
		if strings.Contains(e, "check-and-set") {
			return true
		}
	}
	return false
}
