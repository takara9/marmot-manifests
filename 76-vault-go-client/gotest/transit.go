package vaultclient

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/vault/api"
)

// Transit は Transit シークレットエンジンを使った「暗号化サービス」です。
// 鍵は Vault の外に出ないため、アプリケーションは暗号文だけを保存すればよく、
// 例えば etcd に保存する VM のパスワードを暗号化する用途に使えます。
type Transit struct {
	client *api.Client
	mount  string
}

// NewTransit は mount (例: "transit") の Transit を扱う Transit を返します。
func NewTransit(c *api.Client, mount string) *Transit {
	return &Transit{client: c, mount: mount}
}

// CreateKey は暗号化鍵を作成します (既存なら何もしない)。
func (t *Transit) CreateKey(ctx context.Context, name string) error {
	_, err := t.client.Logical().WriteWithContext(ctx, t.mount+"/keys/"+name, nil)
	return err
}

// Rotate は鍵の新しいバージョンを作成します。既存の暗号文は引き続き復号できます。
func (t *Transit) Rotate(ctx context.Context, name string) error {
	_, err := t.client.Logical().WriteWithContext(ctx, t.mount+"/keys/"+name+"/rotate", nil)
	return err
}

// Encrypt は平文を暗号化し、"vault:v<鍵バージョン>:..." 形式の暗号文を返します。
func (t *Transit) Encrypt(ctx context.Context, name string, plaintext []byte) (string, error) {
	sec, err := t.client.Logical().WriteWithContext(ctx, t.mount+"/encrypt/"+name, map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	})
	if err != nil {
		return "", err
	}
	return stringField(sec, "ciphertext")
}

// Decrypt は暗号文を復号します。
func (t *Transit) Decrypt(ctx context.Context, name, ciphertext string) ([]byte, error) {
	sec, err := t.client.Logical().WriteWithContext(ctx, t.mount+"/decrypt/"+name, map[string]any{
		"ciphertext": ciphertext,
	})
	if err != nil {
		return nil, err
	}
	b64, err := stringField(sec, "plaintext")
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(b64)
}

// Rewrap は、平文を取り出さずに、暗号文を最新の鍵バージョンで暗号化し直します。
func (t *Transit) Rewrap(ctx context.Context, name, ciphertext string) (string, error) {
	sec, err := t.client.Logical().WriteWithContext(ctx, t.mount+"/rewrap/"+name, map[string]any{
		"ciphertext": ciphertext,
	})
	if err != nil {
		return "", err
	}
	return stringField(sec, "ciphertext")
}

func stringField(sec *api.Secret, key string) (string, error) {
	if sec == nil || sec.Data == nil {
		return "", fmt.Errorf("empty response")
	}
	v, ok := sec.Data[key].(string)
	if !ok {
		return "", fmt.Errorf("field %q not found in response", key)
	}
	return v, nil
}
