package vaultclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

func parseCertPEM(t *testing.T, s string) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		t.Fatalf("no PEM block in %q", s)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return cert
}

// Vault を CA として、短命のサーバー証明書を発行できること。
// ルート CA で検証でき、SAN と有効期限が指定どおりで、許可していないドメインは発行できない。
func TestPKIIssueCertificate(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	mount := newMount(t, admin, "pki", nil)
	if err := admin.Sys().TuneMountWithContext(ctx, mount, api.MountConfigInput{MaxLeaseTTL: "24h"}); err != nil {
		t.Fatalf("tune: %v", err)
	}

	root, err := admin.Logical().WriteWithContext(ctx, mount+"/root/generate/internal", map[string]any{
		"common_name": "vault-go-client test CA",
		"ttl":         "24h",
	})
	if err != nil {
		t.Fatalf("generate root: %v", err)
	}
	caCert := parseCertPEM(t, str(t, root, "certificate"))
	if !caCert.IsCA {
		t.Error("generated root is not a CA")
	}

	_, err = admin.Logical().WriteWithContext(ctx, mount+"/roles/web", map[string]any{
		"allowed_domains":  "labo.local",
		"allow_subdomains": true,
		"allow_ip_sans":    true,
		"max_ttl":          "1h",
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	issued, err := admin.Logical().WriteWithContext(ctx, mount+"/issue/web", map[string]any{
		"common_name": "vm1.labo.local",
		"alt_names":   "vm1.dev.labo.local",
		"ip_sans":     "192.168.1.50",
		"ttl":         "10m",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	certPEM, keyPEM := str(t, issued, "certificate"), str(t, issued, "private_key")
	cert := parseCertPEM(t, certPEM)

	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: roots, DNSName: "vm1.dev.labo.local", KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}); err != nil {
		t.Errorf("verify against the issuing CA: %v", err)
	}
	if len(cert.IPAddresses) != 1 || !cert.IPAddresses[0].Equal(net.ParseIP("192.168.1.50")) {
		t.Errorf("IP SANs = %v", cert.IPAddresses)
	}
	if d := time.Until(cert.NotAfter); d < 9*time.Minute || d > 11*time.Minute {
		t.Errorf("certificate expires in %s, want about 10m", d.Round(time.Second))
	}
	if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
		t.Errorf("private key does not match the certificate: %v", err)
	}

	// 別の CA で検証すると失敗する
	other := x509.NewCertPool()
	if _, err := cert.Verify(x509.VerifyOptions{Roots: other}); err == nil {
		t.Error("certificate verified without its CA")
	}
	// ロールが許可していないドメインと、最大 TTL を超える期間
	if _, err := admin.Logical().WriteWithContext(ctx, mount+"/issue/web", map[string]any{"common_name": "evil.example.com"}); err == nil {
		t.Error("issued a certificate for a domain not allowed by the role")
	}
	long, err := admin.Logical().WriteWithContext(ctx, mount+"/issue/web", map[string]any{"common_name": "vm2.labo.local", "ttl": "48h"})
	if err == nil {
		if d := time.Until(parseCertPEM(t, str(t, long, "certificate")).NotAfter); d > 61*time.Minute {
			t.Errorf("ttl was not capped by role max_ttl: expires in %s", d)
		}
	}

	// 失効できること
	if _, err := admin.Logical().WriteWithContext(ctx, mount+"/revoke", map[string]any{"serial_number": str(t, issued, "serial_number")}); err != nil {
		t.Errorf("revoke: %v", err)
	}
}
