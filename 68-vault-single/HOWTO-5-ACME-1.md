# ACME で証明書を再発行する方法

**可能です。** Vault の PKI シークレットエンジンは **1.14 以降、ACME サーバー機能を標準搭載**しています（Community Edition でも利用可）。certbot、acme.sh、lego、cert-manager などの標準 ACME クライアントから自動発行・自動更新できます。

## 前提条件

| 項目 | 内容 |
|---|---|
| Vault のバージョン | 1.14 以降 |
| Vault の TLS | **HTTPS 必須**（ACME クライアントは HTTP を受け付けません）。Vault の証明書をクライアントが信頼している必要があります |
| チャレンジ | http-01 / dns-01 / tls-alpn-01 |
| 到達性 | Vault からクライアントのドメインへ、チャレンジ検証のために到達できること（http-01 なら 80番） |

## Vault 側の設定（中間CA `pki_int` の例）

```bash
# 1. ACME に必要なヘッダーを通す
vault secrets tune \
  -passthrough-request-headers=If-Modified-Since \
  -allowed-response-headers=Last-Modified \
  -allowed-response-headers=Location \
  -allowed-response-headers=Replay-Nonce \
  -allowed-response-headers=Link \
  pki_int

# 2. クラスタのパス（ACME ディレクトリの URL に使われる）
vault write pki_int/config/cluster \
  path=https://vault.labo.local:8200/v1/pki_int \
  aia_path=https://vault.labo.local:8200/v1/pki_int

# 3. ACME を有効化（発行に使うロールを固定）
vault write pki_int/config/acme \
  enabled=true \
  default_directory_policy="role:labo-local"
```

ディレクトリ URL は次のとおりです。

- `https://vault.labo.local:8200/v1/pki_int/acme/directory`（デフォルトポリシー）
- `https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory`（ロール指定）

必要に応じて、`eab_policy=always-required` で EAB（External Account Binding）を必須にして、無認証の発行を防げます。

## クライアント側（Nginx サーバーでの certbot 例）

```bash
# Vault の TLS を検証するためルートCAを指定
export REQUESTS_CA_BUNDLE=/path/to/root_ca.crt

sudo -E certbot certonly --standalone \
  --server https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory \
  -d app.labo.local
```

Nginx プラグインを使う場合は `certbot --nginx` でも設定の書き換えまで自動化できます。更新は `certbot renew`（systemd timer / cron）で行います。

## 注意点

- **証明書の有効期間はロール側で決まります。** `ttl` / `max_ttl` を設定してください（例: `ttl=72h`）。
- **certbot の更新タイミングに注意してください。** certbot は残り 30 日未満で更新するため、短寿命の証明書では実行のたびに更新されます。`renew_before_expiry` で調整するか、短寿命向けの acme.sh / lego を使います。
- **Vault の HTTPS 証明書は ACME 経由で発行できません。** Vault 自身の TLS 証明書は別途用意する必要があります（鶏と卵の問題）。
- **Kubernetes では cert-manager に統合できます。** ACME ではなく Vault Issuer を使うほうが、トークン管理は必要ですが構成は簡単です。

ラボには既に step-ca（`ca.labo.local`）の ACME サーバーがあるため、**クライアント側の設定は `--server` の URL を差し替えるだけで両者を切り替えられます。**