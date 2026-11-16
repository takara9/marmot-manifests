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

### `vault secrets tune` のオプションの意味

Vault は既定で、クライアントと PKI バックエンドの間で受け渡す HTTP ヘッダーを制限しています。ACME が使うヘッダーは明示的に許可しないと通らず、ACME クライアントが正常に動作しません。

| オプション | 意味 |
|---|---|
| `-passthrough-request-headers=If-Modified-Since` | **リクエスト**ヘッダーのうち、クライアントから PKI バックエンドへそのまま渡すものを指定します。`If-Modified-Since` は、前回取得以降に更新されたかを確認する条件付き取得に使われます。 |
| `-allowed-response-headers=Last-Modified` | PKI バックエンドが返す**レスポンス**ヘッダーのうち、クライアントへ返すことを許可するものを指定します。`Last-Modified` は最終更新日時で、`If-Modified-Since` と対になります。 |
| `-allowed-response-headers=Location` | 新規アカウントや新規オーダーの作成時に、作成されたリソースの URL を示します。ACME クライアントはこれでアカウントやオーダーの URL を知ります。 |
| `-allowed-response-headers=Replay-Nonce` | リプレイ攻撃対策用のワンタイム値（nonce）です。クライアントは次のリクエストの署名にこの値を含めるため、返されないと ACME の処理が進みません。 |
| `-allowed-response-headers=Link` | 関連リソースへのリンクを示します。ACME では `rel="index"` や、証明書の代替チェーンへのリンク（`rel="alternate"`）に使われます。 |

`-allowed-response-headers` は、許可するヘッダーごとに繰り返し指定します。

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

## Vault のリスナー証明書を pki_int で発行して差し替える

[vault.hcl](playbook/roles/vault/templates/vault.hcl) のリスナーは `/opt/vault/tls/tls.crt` と `tls.key` を使います。この証明書が自己署名のままだと、クライアントはルートCAを信頼しても Vault の TLS を検証できません。ACME クライアントは Vault への HTTPS 接続を検証するので、リスナー証明書を `pki_int` で発行したものに差し替えます。

通常の発行（`pki_int/issue/`）なので ACME は使いません。Vault サーバー上で、`pki_int/issue/labo-local` の権限を持つトークンでログインして実行します。

```bash
export VAULT_ADDR=https://127.0.0.1:8200
export VAULT_SKIP_VERIFY=true     # 差し替え前の自己署名証明書を検証しないため

# 1. 証明書を発行（接続に使う名前と IP をすべて SAN に含める）
vault write -format=json pki_int/issue/labo-local \
  common_name=vault.labo.local \
  alt_names=localhost \
  ip_sans=127.0.0.1,192.168.1.251 \
  ttl=720h > vault-tls.json

# 2. 現在の証明書を退避
sudo cp -a /opt/vault/tls /opt/vault/tls.bak

# 3. サーバー証明書 + 中間CA（ca_chain）を tls.crt に、秘密鍵を tls.key に配置
jq -r '.data.certificate, .data.ca_chain[]' vault-tls.json | sudo tee /opt/vault/tls/tls.crt > /dev/null
jq -r '.data.private_key' vault-tls.json | sudo tee /opt/vault/tls/tls.key > /dev/null
sudo chown vault:vault /opt/vault/tls/tls.crt /opt/vault/tls/tls.key
sudo chmod 0644 /opt/vault/tls/tls.crt
sudo chmod 0600 /opt/vault/tls/tls.key
shred -u vault-tls.json

# 4. 再起動せずに反映（SIGHUP でリスナーの証明書を再読み込み。シールもされない）
sudo systemctl reload vault
```

`tls.crt` には、先頭にサーバー証明書、続けて中間CAを並べます。ルートCAは含めません。中間CAを含めないと、ルートCAしか信頼していないクライアントがチェーンを組み立てられません。

### 確認

ルートCAだけを信頼させて、検証が通ることを確認します。`VAULT_SKIP_VERIFY` は使いません。

```bash
unset VAULT_SKIP_VERIFY
export VAULT_ADDR=https://vault.labo.local:8200
export VAULT_CACERT=root_ca.crt     # vault read -field=certificate pki/cert/ca で取得

vault status
openssl s_client -connect vault.labo.local:8200 -servername vault.labo.local -CAfile root_ca.crt </dev/null 2>/dev/null \
  | grep -E "Verify return code|subject=|issuer="
# Verify return code: 0 (ok) になること

echo | openssl s_client -connect vault.labo.local:8200 2>/dev/null | openssl x509 -noout -subject -issuer -enddate
```

### 注意点

- **有効期間はロールの `max_ttl`（720h）が上限です。** 約 30 日で切れるため、期限前に同じ手順で再発行します。Vault の証明書が切れると ACME を含むすべての API が使えなくなります。頻繁な更新を避けたい場合は、Vault 用に `max_ttl` の長い専用ロールを作ってください。
- **接続に使う名前と IP は SAN に入れてください。** `ip_sans` に含めていない IP アドレス（例: `https://192.168.1.251:8200`）で接続すると検証に失敗します。ロールの `allow_ip_sans=true` が必要です。
- **ルートCAの配布が前提です。** クライアントは `root_ca.crt` を OS の信頼ストアか `VAULT_CACERT` で信頼する必要があります（Ansible の例は [HOWTO-6-ACME-5.md](HOWTO-6-ACME-5.md) を参照）。
- **自動アンシール用スクリプトは影響を受けません。** [vault-auto-unseal.sh.j2](playbook/roles/vault/templates/vault-auto-unseal.sh.j2) は `127.0.0.1` へ `VAULT_SKIP_VERIFY=true` で接続します。検証を有効にする場合は、`127.0.0.1` と `localhost` を SAN に入れた上で `VAULT_CACERT` を指定します。
- **戻す場合は退避した `/opt/vault/tls.bak` を戻して `reload` します。**

## 注意点

- **証明書の有効期間はロール側で決まります。** `ttl` / `max_ttl` を設定してください（例: `ttl=72h`）。
- **certbot の更新タイミングに注意してください。** certbot は残り 30 日未満で更新するため、短寿命の証明書では実行のたびに更新されます。`renew_before_expiry` で調整するか、短寿命向けの acme.sh / lego を使います。
- **Vault の HTTPS 証明書は ACME 経由で発行できません。** Vault 自身の TLS 証明書は別途用意する必要があります（鶏と卵の問題）。`pki_int` で通常発行して差し替える手順は、上の「Vault のリスナー証明書を pki_int で発行して差し替える」を参照してください。
- **Kubernetes では cert-manager に統合できます。** ACME ではなく Vault Issuer を使うほうが、トークン管理は必要ですが構成は簡単です。

ラボには既に step-ca（`ca.labo.local`）の ACME サーバーがあるため、**クライアント側の設定は `--server` の URL を差し替えるだけで両者を切り替えられます。**