package vaultclient

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSH 公開鍵に Vault の CA で署名して、短命のユーザー証明書を得られること (68-vault-single の SSH エンジンと同じ方式)。
// サーバーが CA の公開鍵だけを信頼すれば、サーバーごとの authorized_keys の管理が不要になる。
func TestSSHSignClientKey(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	mount := newMount(t, admin, "ssh", nil)

	ca, err := admin.Logical().WriteWithContext(ctx, mount+"/config/ca", map[string]any{"generate_signing_key": true})
	if err != nil {
		t.Fatalf("generate CA: %v", err)
	}
	caKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(str(t, ca, "public_key")))
	if err != nil {
		t.Fatalf("parse CA public key: %v", err)
	}

	_, err = admin.Logical().WriteWithContext(ctx, mount+"/roles/admin", map[string]any{
		"key_type":                "ca",
		"allow_user_certificates": true,
		"allowed_users":           "ubuntu,takara",
		"default_user":            "ubuntu",
		"default_extensions":      map[string]any{"permit-pty": ""},
		"ttl":                     "30m",
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	authorized := string(ssh.MarshalAuthorizedKey(sshPub))

	signed, err := admin.Logical().WriteWithContext(ctx, mount+"/sign/admin", map[string]any{
		"public_key":       authorized,
		"valid_principals": "ubuntu",
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	parsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(str(t, signed, "signed_key")))
	if err != nil {
		t.Fatalf("parse signed key: %v", err)
	}
	cert, ok := parsed.(*ssh.Certificate)
	if !ok {
		t.Fatalf("signed key is %T, want *ssh.Certificate", parsed)
	}
	if cert.CertType != ssh.UserCert || len(cert.ValidPrincipals) != 1 || cert.ValidPrincipals[0] != "ubuntu" {
		t.Errorf("certificate type=%d principals=%v", cert.CertType, cert.ValidPrincipals)
	}
	if !bytes.Equal(cert.Key.Marshal(), sshPub.Marshal()) {
		t.Error("certificate was issued for a different public key")
	}
	if d := time.Until(time.Unix(int64(cert.ValidBefore), 0)); d < 29*time.Minute || d > 31*time.Minute {
		t.Errorf("certificate expires in %s, want about 30m", d.Round(time.Second))
	}

	// sshd と同じ検証 (CA の公開鍵で署名を確認し、ユーザー名が principals に含まれること)
	checker := &ssh.CertChecker{
		IsUserAuthority: func(auth ssh.PublicKey) bool { return bytes.Equal(auth.Marshal(), caKey.Marshal()) },
	}
	if err := checker.CheckCert("ubuntu", cert); err != nil {
		t.Errorf("CheckCert(ubuntu): %v", err)
	}
	if err := checker.CheckCert("takara", cert); err == nil {
		t.Error("certificate is valid for a user that is not in valid_principals")
	}

	// ロールが許可していないユーザーの証明書は発行できない
	if _, err := admin.Logical().WriteWithContext(ctx, mount+"/sign/admin", map[string]any{
		"public_key":       authorized,
		"valid_principals": "root",
	}); err == nil {
		t.Error("signed a certificate for a user not allowed by the role")
	}
}
