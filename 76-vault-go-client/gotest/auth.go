package vaultclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"
)

// ErrTokenExpired は、トークンがこれ以上更新できず、再ログインが必要なことを表します。
var ErrTokenExpired = errors.New("vault token can no longer be renewed; login again")

// LoginAppRole は AppRole で認証し、トークンを設定したクライアントを返します。
// marmot のサービスが、固定のトークンを持たずに、role_id と secret_id で認証する用途です。
// secret_id を使い捨て (secret_id_num_uses=1) にすれば、漏えいしても再利用できません。
func LoginAppRole(ctx context.Context, cfg Config, mount, roleID, secretID string) (*api.Client, *api.Secret, error) {
	anon := cfg
	anon.Token = ""
	c, err := Connect(ctx, anon)
	if err != nil {
		return nil, nil, err
	}
	sec, err := c.Logical().WriteWithContext(ctx, "auth/"+mount+"/login", map[string]any{
		"role_id":   roleID,
		"secret_id": secretID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("approle login: %w", err)
	}
	if sec == nil || sec.Auth == nil {
		return nil, nil, errors.New("approle login: no auth information in the response")
	}
	c.SetToken(sec.Auth.ClientToken)
	return c, sec, nil
}

// WatchToken は、トークンの有効期限が近づくたびに更新します。
// 更新できなくなった (最大 TTL に達した、失効した) 場合は ErrTokenExpired を返すので、
// 呼び出し側は LoginAppRole で再ログインします。ctx が終了すると ctx.Err() を返します。
func WatchToken(ctx context.Context, c *api.Client, auth *api.Secret, increment int, onRenew func(*api.RenewOutput)) error {
	w, err := c.NewLifetimeWatcher(&api.LifetimeWatcherInput{Secret: auth, Increment: increment})
	if err != nil {
		return err
	}
	go w.Start()
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-w.DoneCh():
			if err != nil {
				return fmt.Errorf("%w: %v", ErrTokenExpired, err)
			}
			return ErrTokenExpired
		case r := <-w.RenewCh():
			if onRenew != nil {
				onRenew(r)
			}
		}
	}
}
