package vaultclient

import (
	"bytes"
	"crypto/rand"
	"net/http"
	"strings"
	"testing"
)

// 暗号化・復号、鍵のローテーション、Rewrap、古い鍵バージョンの無効化ができること。
func TestTransitEncryptDecrypt(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	mount := newMount(t, admin, "transit", nil)
	tr := NewTransit(admin, mount)
	if err := tr.CreateKey(ctx, "vm-password"); err != nil {
		t.Fatalf("create key: %v", err)
	}

	binary := make([]byte, 1024)
	_, _ = rand.Read(binary)
	for name, plain := range map[string][]byte{
		"text":     []byte("p@ssw0rd!"),
		"japanese": []byte("機密情報です"),
		"binary":   binary,
		"empty":    []byte(""),
	} {
		ct, err := tr.Encrypt(ctx, "vm-password", plain)
		if err != nil {
			t.Fatalf("%s: encrypt: %v", name, err)
		}
		if !strings.HasPrefix(ct, "vault:v1:") {
			t.Errorf("%s: ciphertext = %q, want vault:v1: prefix", name, ct)
		}
		got, err := tr.Decrypt(ctx, "vm-password", ct)
		if err != nil || !bytes.Equal(got, plain) {
			t.Errorf("%s: decrypt = %q, %v", name, got, err)
		}
	}

	// 同じ平文でも、暗号文は毎回異なる (nonce)
	a, _ := tr.Encrypt(ctx, "vm-password", []byte("same"))
	b, _ := tr.Encrypt(ctx, "vm-password", []byte("same"))
	if a == b {
		t.Error("two encryptions of the same plaintext are identical")
	}

	// ローテーション後も古い暗号文は復号でき、Rewrap で最新の鍵バージョンになる
	old, _ := tr.Encrypt(ctx, "vm-password", []byte("rotate-me"))
	if err := tr.Rotate(ctx, "vm-password"); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	fresh, _ := tr.Encrypt(ctx, "vm-password", []byte("rotate-me"))
	if !strings.HasPrefix(fresh, "vault:v2:") {
		t.Errorf("after rotate: ciphertext = %q, want vault:v2:", fresh)
	}
	if got, err := tr.Decrypt(ctx, "vm-password", old); err != nil || string(got) != "rotate-me" {
		t.Errorf("decrypt old ciphertext after rotate = %q, %v", got, err)
	}
	rewrapped, err := tr.Rewrap(ctx, "vm-password", old)
	if err != nil || !strings.HasPrefix(rewrapped, "vault:v2:") {
		t.Fatalf("rewrap = %q, %v", rewrapped, err)
	}
	if got, _ := tr.Decrypt(ctx, "vm-password", rewrapped); string(got) != "rotate-me" {
		t.Errorf("decrypt rewrapped = %q", got)
	}

	// v1 の復号を禁止すると、Rewrap していない古い暗号文は復号できなくなる
	_, err = admin.Logical().WriteWithContext(ctx, mount+"/keys/vm-password/config", map[string]any{"min_decryption_version": 2})
	if err != nil {
		t.Fatalf("set min_decryption_version: %v", err)
	}
	if _, err := tr.Decrypt(ctx, "vm-password", old); err == nil {
		t.Error("v1 ciphertext still decrypts after min_decryption_version=2")
	}
	if got, err := tr.Decrypt(ctx, "vm-password", rewrapped); err != nil || string(got) != "rotate-me" {
		t.Errorf("rewrapped ciphertext after min_decryption_version=2: %q, %v", got, err)
	}
}

// 暗号化だけを許可したトークンでは復号できないこと。書き込み専用のサービスを作れる。
func TestTransitEncryptOnlyToken(t *testing.T) {
	ctx := testCtx(t)
	admin := newClient(t)
	mount := newMount(t, admin, "transit", nil)
	if err := NewTransit(admin, mount).CreateKey(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	policy := newPolicy(t, admin, `path "`+mount+`/encrypt/k" { capabilities = ["update"] }`)
	tr := NewTransit(tokenClient(t, admin, policy), mount)

	ct, err := tr.Encrypt(ctx, "k", []byte("write-only"))
	if err != nil {
		t.Fatalf("encrypt with encrypt-only token: %v", err)
	}
	if _, err := tr.Decrypt(ctx, "k", ct); statusOf(err) != http.StatusForbidden {
		t.Errorf("decrypt with encrypt-only token: err = %v, want 403", err)
	}
	if got, err := NewTransit(admin, mount).Decrypt(ctx, "k", ct); err != nil || string(got) != "write-only" {
		t.Errorf("admin decrypt = %q, %v", got, err)
	}
}
