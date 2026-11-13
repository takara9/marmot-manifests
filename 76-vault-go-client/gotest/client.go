// Package vaultclient は、marmot から HashiCorp Vault (75-vault-cluster) を
// 機密情報の保管庫として使うための検証用コードです。
// 複数ノードへの接続と TLS、KV v2 の排他更新、Transit による暗号化、
// AppRole 認証とトークン更新を試作しています。
package vaultclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

const (
	// EnvAddrs は Vault ノードのアドレス (カンマ区切り)。省略時は 75-vault-cluster の 3 ノード。
	EnvAddrs = "VAULT_ADDRS"
	// EnvAddr は単一アドレス (Vault CLI と共通)。EnvAddrs が未設定のときに使う。
	EnvAddr = "VAULT_ADDR"
	// EnvCACert は CA 証明書ファイルのパス (Vault CLI と共通)。
	EnvCACert = "VAULT_CACERT"
	// EnvToken は認証トークン (Vault CLI と共通)。省略時は ~/.vault-token を読む。
	EnvToken = "VAULT_TOKEN"
	// EnvExpectNodes は期待するノード数。省略時はアドレスの数。
	EnvExpectNodes = "VAULT_EXPECT_NODES"
)

// DefaultAddrs は 75-vault-cluster の各ノードです。
var DefaultAddrs = []string{
	"https://192.168.1.194:8200",
	"https://192.168.1.195:8200",
	"https://192.168.1.196:8200",
}

// defaultMaxRetries は、接続後のリクエストが一時的に失敗したときの再試行回数です。
const defaultMaxRetries = 2

// ErrNoHealthyNode は、どのノードも利用可能でなかったことを表します。
var ErrNoHealthyNode = errors.New("no healthy vault node")

// Config は接続設定です。
type Config struct {
	Addrs   []string
	CACert  string
	Token   string
	Nodes   int
	Timeout time.Duration
}

// LoadConfig は環境変数から接続設定を決定します。
func LoadConfig() (Config, error) {
	cfg := Config{Addrs: DefaultAddrs, Timeout: 10 * time.Second}
	if v := os.Getenv(EnvAddrs); v != "" {
		cfg.Addrs = splitCSV(v)
	} else if v := os.Getenv(EnvAddr); v != "" {
		cfg.Addrs = []string{v}
	}
	cfg.Nodes = len(cfg.Addrs)
	if v := os.Getenv(EnvExpectNodes); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("%s=%q: %w", EnvExpectNodes, v, err)
		}
		cfg.Nodes = n
	}
	cfg.CACert = findCACert()
	cfg.Token = findToken()
	return cfg, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// findCACert は CA 証明書を VAULT_CACERT、./certs、/etc/vault-client の順に探します。
// 見つからなければ空を返し、その場合 TLS の検証には OS の CA が使われます。
func findCACert() string {
	if v := os.Getenv(EnvCACert); v != "" {
		return v
	}
	for _, c := range []string{"certs/ca.crt", "/etc/vault-client/ca.crt"} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func findToken() string {
	if v := os.Getenv(EnvToken); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, ".vault-token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// NewClientForAddr は指定した 1 ノードへ接続するクライアントを作成します (トークンは未設定)。
// スタンバイノードへのリクエストは、Vault がアクティブノードへ転送するか、
// リダイレクトされるため、どのノードに接続しても読み書きできます。
func NewClientForAddr(cfg Config, addr string) (*api.Client, error) {
	ac := api.DefaultConfig()
	if ac.Error != nil {
		return nil, ac.Error
	}
	ac.Address = addr
	ac.Timeout = cfg.Timeout
	if strings.HasPrefix(addr, "https://") {
		if err := ac.ConfigureTLS(&api.TLSConfig{CACert: cfg.CACert}); err != nil {
			return nil, fmt.Errorf("configure TLS: %w", err)
		}
	}
	c, err := api.NewClient(ac)
	if err != nil {
		return nil, err
	}
	c.ClearToken()
	return c, nil
}

// Connect は Addrs を順に試し、最初に初期化済みで unseal 済みのノードへ接続したクライアントを返します。
// 先頭のノードが停止していても、次のノードへ切り替えて接続できます。
// Vault の Go クライアントは 1 つのアドレスしか持てないため、
// ノードが落ちた場合は再度 Connect を呼んで接続し直します。
func Connect(ctx context.Context, cfg Config) (*api.Client, error) {
	var errs []error
	for _, addr := range cfg.Addrs {
		c, err := NewClientForAddr(cfg, addr)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", addr, err))
			continue
		}
		// 停止ノードの検出を早めるため、状態確認ではリトライしない
		c.SetMaxRetries(0)
		h, err := c.Sys().HealthWithContext(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", addr, err))
			continue
		}
		if !h.Initialized || h.Sealed {
			errs = append(errs, fmt.Errorf("%s: initialized=%v sealed=%v", addr, h.Initialized, h.Sealed))
			continue
		}
		c.SetMaxRetries(defaultMaxRetries)
		if cfg.Token != "" {
			c.SetToken(cfg.Token)
		}
		return c, nil
	}
	return nil, fmt.Errorf("%w: %w", ErrNoHealthyNode, errors.Join(errs...))
}
