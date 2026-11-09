# Vault を CA として使う方法（PKI シークレットエンジン）

Vault の **PKI シークレットエンジン**で、証明書の発行・失効・更新を API/CLI から自動化できます。本番では「ルートCA」と「中間CA」を別マウントに分け、日常の発行は中間CAだけで行う構成が基本です。

## 構成

| マウント | 役割 | 最大TTL目安 |
|---|---|---|
| `pki` | ルートCA（普段は使わない） | 10年 |
| `pki_int` | 中間CA（実際に証明書を発行） | 5年 |

## 手順

**1. PKI エンジンを有効化し、ルートCAを作成**

```bash
vault secrets enable pki
vault secrets tune -max-lease-ttl=87600h pki

vault write -field=certificate pki/root/generate/internal \
  common_name="labo.local Root CA" issuer_name="root-2026" \
  ttl=87600h > root_ca.crt

vault write pki/config/urls \
  issuing_certificates="$VAULT_ADDR/v1/pki/ca" \
  crl_distribution_points="$VAULT_ADDR/v1/pki/crl"
```

**2. 中間CAを作成し、ルートで署名**

```bash
vault secrets enable -path=pki_int pki
vault secrets tune -max-lease-ttl=43800h pki_int

# CSR 生成
vault write -format=json pki_int/intermediate/generate/internal \
  common_name="labo.local Intermediate CA" issuer_name="int-2026" \
  | jq -r '.data.csr' > pki_int.csr

# ルートで署名
vault write -format=json pki/root/sign-intermediate \
  csr=@pki_int.csr format=pem_bundle ttl=43800h \
  | jq -r '.data.certificate' > int.cert.pem

# 署名済み証明書を中間CAへ登録
vault write pki_int/intermediate/set-signed certificate=@int.cert.pem
```

**3. ロール（発行ポリシー）を定義**

```bash
vault write pki_int/roles/labo-local \
  allowed_domains="labo.local" allow_subdomains=true \
  max_ttl=720h key_type=ec key_bits=256
```

**4. 証明書を発行**

```bash
vault write pki_int/issue/labo-local \
  common_name="app.labo.local" ttl=24h
```

返却される `certificate` / `private_key` / `issuing_ca` / `ca_chain` を利用します。

**5. アクセス制御（最小権限）**

```hcl
path "pki_int/issue/labo-local" {
  capabilities = ["update"]
}
```

このポリシーを AppRole や Kubernetes 認証に紐づけます。

## 自動化の選択肢

| 用途 | 方法 |
|---|---|
| Kubernetes | cert-manager の Vault Issuer |
| ACME クライアント | Vault の ACME 対応（1.14 以降。`pki_int/config/acme` で有効化） |
| サーバー上での更新 | Vault Agent のテンプレート |

## 運用上の注意

- ルートCA鍵は発行後に Vault から外す、またはルートだけ別の Vault・オフラインで運用するのが安全です。
- 短い TTL（数時間〜数日）で自動更新し、失効に頼らない運用が推奨です。
- クライアントには `root_ca.crt` を信頼済みとして配布します。
- CRL/OCSP の URL は、実際にクライアントから到達できるアドレス（`cluster_path` / `path` の設定も含む）にしてください。
- Vault は HA 構成で Raft 等のバックエンドを使い、バックアップ（スナップショット）を取ります。
- 公式ドキュメントは [PKI secrets engine](https://developer.hashicorp.com/vault/docs/secrets/pki) です。

ラボの既存環境（例: ACME 用の step-ca）との使い分けや、cert-manager 連携の具体的な設定も必要でしたらお知らせください。