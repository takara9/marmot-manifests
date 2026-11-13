// Package etcdtest は、marmot の pkg/db (etcd アクセス層) を改善するための
// 検証用コードです。71-etcd-single (単一ノード) と 72-etcd-cluster (3ノード) の
// どちらに対しても、環境変数 ETCD_TARGET で切り替えて同じテストを実行できます。
package etcdtest

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	// EnvTarget は接続先 (single | cluster)。省略時は single。
	EnvTarget = "ETCD_TARGET"
	// EnvEndpoints はエンドポイントの上書き (カンマ区切り)。
	EnvEndpoints = "ETCD_ENDPOINTS"
	// EnvCACert は CA 証明書ファイルの上書き。
	EnvCACert = "ETCD_CACERT"
	// EnvExpectMembers は期待するメンバー数の上書き。
	EnvExpectMembers = "ETCD_EXPECT_MEMBERS"
)

// Target は接続先の etcd 環境です。
type Target struct {
	Name      string
	Endpoints []string
	CACert    string
	Members   int
}

var defaultTargets = map[string]Target{
	"single": {
		Name:      "single",
		Endpoints: []string{"https://192.168.1.190:2379"},
		Members:   1,
	},
	"cluster": {
		Name: "cluster",
		Endpoints: []string{
			"https://192.168.1.191:2379",
			"https://192.168.1.192:2379",
			"https://192.168.1.193:2379",
		},
		Members: 3,
	},
}

// ResolveTarget は環境変数から接続先を決定します。
func ResolveTarget() (Target, error) {
	name := os.Getenv(EnvTarget)
	if name == "" {
		name = "single"
	}
	t, ok := defaultTargets[name]
	if !ok {
		return Target{}, fmt.Errorf("%s=%q is invalid (single | cluster)", EnvTarget, name)
	}
	if v := os.Getenv(EnvEndpoints); v != "" {
		t.Endpoints = splitCSV(v)
	}
	if v := os.Getenv(EnvExpectMembers); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Target{}, fmt.Errorf("%s=%q: %w", EnvExpectMembers, v, err)
		}
		t.Members = n
	}
	ca, err := findCACert(name)
	if err != nil {
		return Target{}, err
	}
	t.CACert = ca
	return t, nil
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

// findCACert は CA 証明書を ETCD_CACERT、./certs、/etc/etcd-client の順に探します。
func findCACert(target string) (string, error) {
	if v := os.Getenv(EnvCACert); v != "" {
		return v, nil
	}
	candidates := []string{
		fmt.Sprintf("certs/%s-ca.crt", target),
		fmt.Sprintf("/etc/etcd-client/%s-ca.crt", target),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("CA certificate for %q not found (tried %s); run `make fetch-ca` or set %s",
		target, strings.Join(candidates, ", "), EnvCACert)
}

// NewClient は Target の全エンドポイントへ接続するクライアントを作成します。
func NewClient(t Target) (*clientv3.Client, error) {
	return NewClientWithEndpoints(t, t.Endpoints)
}

// NewClientWithEndpoints は指定したエンドポイントだけを使うクライアントを作成します。
// marmot の NewDatabase は単一 URL かつ TLS 無しだが、こちらは複数エンドポイントと TLS に対応し、
// keepalive により停止ノードを早期に検出して他のエンドポイントへ切り替える。
func NewClientWithEndpoints(t Target, endpoints []string) (*clientv3.Client, error) {
	cfg := clientv3.Config{
		Endpoints:            endpoints,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    10 * time.Second,
		DialKeepAliveTimeout: 3 * time.Second,
	}
	if needTLS(endpoints) {
		tc, err := tlsConfig(t.CACert)
		if err != nil {
			return nil, err
		}
		cfg.TLS = tc
	}
	return clientv3.New(cfg)
}

func needTLS(endpoints []string) bool {
	for _, e := range endpoints {
		if strings.HasPrefix(e, "https://") {
			return true
		}
	}
	return false
}

func tlsConfig(caFile string) (*tls.Config, error) {
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("no valid certificate in " + caFile)
	}
	return &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}, nil
}
