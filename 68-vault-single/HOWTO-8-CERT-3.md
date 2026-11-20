# Garage S3 (s3.labo.local) の TLS 証明書を Vault で発行する

[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) で作成した中間CA（`pki_int`）から、Garage S3 のサーバー証明書を発行して設定する手順です。GitLab 版は [HOWTO-8-CERT-1.md](HOWTO-8-CERT-1.md)、Keycloak 版は [HOWTO-8-CERT-2.md](HOWTO-8-CERT-2.md) を参照してください。

| 項目 | 値 |
|---|---|
| S3 サーバー | `s3.labo.local`（192.168.1.198、[78-S3](../78-S3/README.md)、サーバー名 `s3-garage`） |
| Vault | `vault.labo.local`（192.168.1.251） |
| 発行元 | `pki_int` のロール `labo-local` |

> 本書は手順書です。実際の環境で実行して確認した結果ではありません。本番に適用する前に、動作を確認してください。

## 0. Garage への証明書の設定方法について

Garage 本体（v2.x）は S3 API・Web・管理 API のいずれも TLS に対応していません（`garage.toml` に証明書を指定する項目がありません）。そのため、同じサーバー上に TLS を終端するリバースプロキシ（本書では nginx）を置き、そこに証明書を設定します。

```text
クライアント --HTTPS(443)--> nginx (証明書をここに設定) --HTTP--> Garage 127.0.0.1:3900
```

[78-S3](../78-S3/README.md) の構成では `garage.labo.local` で DNS 登録されていますが、本書は `s3.labo.local` を使います。DNS に `s3.labo.local` → `192.168.1.198` を追加してください。

## 1. 前提

- `pki` / `pki_int` が有効で、ロール `labo-local` が作成済みであること（[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) の手順 1〜3）。
- `s3.labo.local` が DNS で `192.168.1.198` に解決できること。
- クライアントに Root CA が信頼されていること（[HOWTO-4-CA-3.md](HOWTO-4-CA-3.md)）。Root CA は [playbook/files/root_ca.crt](playbook/files/root_ca.crt) です。
- [78-S3](../78-S3/README.md) の Garage が稼働していること（S3 API は `3900`）。

`labo-local` ロールは `allowed_domains="labo.local"`、`allow_subdomains=true`、`max_ttl=720h` です。`s3.labo.local` はそのまま発行できますが、TTL は最大 30 日（720h）です。

## 2. 証明書を発行する

Vault サーバーで実行します。

```bash
ssh ubuntu@vault.labo.local
vault login   # Root Token

vault write -format=json pki_int/issue/labo-local \
  common_name="s3.labo.local" alt_names="garage.labo.local" ip_sans="192.168.1.198" \
  ttl=720h private_key_format=pkcs8 > s3.json
```

- `alt_names` / `ip_sans` は、既存の名前（`garage.labo.local`）や IP アドレスでもアクセスする場合の指定です。不要であれば省略できます。
- `private_key_format=pkcs8` は、秘密鍵を PKCS#8 形式（`BEGIN PRIVATE KEY`）で出力する指定です。

### バケット名をホスト名に含める場合（virtual-hosted style）

[78-S3](../78-S3/README.md) の Garage は `root_domain = ".s3.labo.local"` を設定しており、`https://demo.s3.labo.local/` のようなバケット名をホスト名にしたアクセス（virtual-hosted style）にも対応しています。この形式で HTTPS を使う場合は、ワイルドカードを SAN に追加します。

```bash
vault write -format=json pki_int/issue/labo-local \
  common_name="s3.labo.local" alt_names="*.s3.labo.local,garage.labo.local" ip_sans="192.168.1.198" \
  ttl=720h private_key_format=pkcs8 > s3.json
```

DNS にも `*.s3.labo.local` → `192.168.1.198` を登録してください。ワイルドカード証明書が拒否される場合は、ロールの `allow_wildcard_certificates` が `true`（既定値）であることを確認します。AWS CLI などで path-style（`https://s3.labo.local/demo/`）だけを使う場合は不要です。

`labo.local` から `lab.internal` へ移行中の場合は、[HOWTO-8-CERT-1.md](HOWTO-8-CERT-1.md) の「lab.internal と並行運用する場合」と同様に、ロールへドメインを追加して `alt_names` に `s3.lab.internal` を指定します。

## 3. 証明書と秘密鍵を取り出す

`certificate` に `ca_chain` を連結します。nginx が中間CAも提示するための設定です。

```bash
jq -r '.data.certificate, (.data.ca_chain[])' s3.json > s3.labo.local.crt
jq -r '.data.private_key' s3.json > s3.labo.local.key
chmod 600 s3.labo.local.key
```

内容を確認します。

```bash
openssl x509 -in s3.labo.local.crt -noout -subject -issuer -dates -ext subjectAltName
head -1 s3.labo.local.key   # -----BEGIN PRIVATE KEY----- であること
```

## 4. S3 サーバーに配置する

[78-S3](../78-S3/README.md) の Playbook（`nginx_tls` ロール）を使う場合は、手順 4・5 の手作業は不要です。証明書と秘密鍵を Playbook を実行するマシンに置き、次のように実行します。nginx のインストール、配置、設定、reload まで自動で行われます。

```bash
cd 78-S3
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml \
  -e s3_tls_mode=provided \
  -e s3_tls_cert_src=$HOME/s3-tls/s3.labo.local.crt \
  -e s3_tls_key_src=$HOME/s3-tls/s3.labo.local.key
```

Garage の平文 HTTP を閉じる場合は `-e garage_s3_bind_address=127.0.0.1` を追加します。garage-ui も同じ証明書で `https://s3.labo.local:8443` に公開され、平文の 8080 は `127.0.0.1` のみになります（無効にするには `-e s3_tls_ui_enabled=false`）。Playbook を使わず手動で設定する場合は、以降の手順に従います（garage-ui の HTTPS 化は、手順 5 の設定を参考に `listen 8443 ssl;` と `proxy_pass http://127.0.0.1:8080;` の server ブロックを追加します）。

### 手動で配置する場合

証明書と秘密鍵を `s3-garage` へ転送します。Vault サーバーから直接転送する例です。

```bash
# Vault サーバーで実行
scp s3.labo.local.crt s3.labo.local.key ubuntu@s3.labo.local:/tmp/
```

S3 サーバーで配置します。

```bash
ssh ubuntu@s3.labo.local

sudo install -d -m 0755 /etc/nginx/tls
sudo install -m 0644 -o root -g root /tmp/s3.labo.local.crt /etc/nginx/tls/s3.labo.local.crt
sudo install -m 0600 -o root -g root /tmp/s3.labo.local.key /etc/nginx/tls/s3.labo.local.key
sudo shred -u /tmp/s3.labo.local.crt /tmp/s3.labo.local.key
```

Vault サーバー上の秘密鍵を含むファイルを削除します。

```bash
# Vault サーバーで実行
shred -u s3.json s3.labo.local.key
```

秘密鍵をリポジトリにコミットしないでください。

## 5. nginx を設定する（Garage の前段で TLS を終端）

nginx をインストールします。

```bash
sudo apt-get update
sudo apt-get install -y nginx
```

`/etc/nginx/conf.d/garage-s3.conf` を作成します。

```nginx
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name s3.labo.local garage.labo.local *.s3.labo.local;

    ssl_certificate     /etc/nginx/tls/s3.labo.local.crt;
    ssl_certificate_key /etc/nginx/tls/s3.labo.local.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    # 大きなオブジェクトのアップロードを許可し、バッファリングしない
    client_max_body_size    0;
    proxy_request_buffering off;
    proxy_buffering         off;

    location / {
        proxy_pass http://127.0.0.1:3900;
        # SigV4 署名はホスト名を含むため、元の Host をそのまま渡す
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_connect_timeout 5s;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }
}
```

> `http2 on;` は nginx 1.25.1 以降の書き方です。Ubuntu 24.04 の nginx（1.24）で `unknown directive "http2"` となる場合は、この行を削除し、`listen 443 ssl http2;` に変更してください。

設定を検査して反映します。

```bash
sudo nginx -t
sudo systemctl enable --now nginx
sudo systemctl reload nginx
```

ファイアウォール（ufw 等）を使っている場合は 443 を許可します。

```bash
sudo ufw allow 443/tcp
```

### 任意: Garage の平文 HTTP を外部に公開しない

HTTPS のみに限定する場合は、Garage の S3 API の待ち受けを `127.0.0.1` に変更します。[garage.toml.j2](../78-S3/playbook/roles/garage/templates/garage.toml.j2) の `api_bind_addr` を次のようにします。

```toml
[s3_api]
api_bind_addr = "127.0.0.1:3900"
```

```bash
sudo systemctl restart garage
```

これで `http://s3.labo.local:3900` へ外部から直接アクセスできなくなります。変更後は既存のクライアント（[78-S3/README.md](../78-S3/README.md) の `AWS_ENDPOINT_URL=http://garage.labo.local:3900`）を HTTPS へ切り替える必要があります。管理 API（3903）と garage-ui（8080）は別途 HTTP で待ち受けるため、必要に応じて同様にプロキシ化してください。

## 6. 動作を確認する

クライアントで、Root CA を指定して確認します。

```bash
openssl s_client -connect s3.labo.local:443 -servername s3.labo.local -CAfile root_ca.crt </dev/null 2>/dev/null | grep -E "Verify return code|subject|issuer"
curl --cacert root_ca.crt -i https://s3.labo.local/
```

`Verify return code: 0 (ok)` と表示されれば証明書は有効です。`curl` は認証なしのため、Garage から `403 AccessDenied` が返れば TLS 経由で Garage に到達しています。

AWS CLI での利用例です。

```bash
export AWS_ACCESS_KEY_ID=<Key ID>
export AWS_SECRET_ACCESS_KEY=<Secret key>
export AWS_DEFAULT_REGION=garage
export AWS_ENDPOINT_URL=https://s3.labo.local
export AWS_CA_BUNDLE=/path/to/root_ca.crt   # Root CA を OS に登録済みなら不要

aws s3 ls
aws s3 cp ./file.txt s3://demo/
```

## 7. 更新する

TTL は最大 30 日です。期限前に、手順 2〜4 を繰り返して証明書ファイルを入れ替え、nginx を再読み込みします。

```bash
sudo systemctl reload nginx
```

期限を確認するには次のコマンドを使います。

```bash
ssh ubuntu@s3.labo.local sudo openssl x509 -in /etc/nginx/tls/s3.labo.local.crt -noout -enddate
```

更新を自動化する場合は、ACME を使います（[HOWTO-5-ACME-1.md](HOWTO-5-ACME-1.md) 以降）。`certbot` や `acme.sh` で取得し、nginx の `ssl_certificate` / `ssl_certificate_key` をその出力先に向けて、更新後に `systemctl reload nginx` を実行する構成にします。

## 8. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `unable to get local issuer certificate` | クライアントに Root CA が登録されていない。または、証明書ファイルに `ca_chain` を連結していない（手順 3）。 |
| `common name ... not allowed by this role` | ロールの `allowed_domains` に該当ドメインがない。ロールを更新する（[HOWTO-8-CERT-1.md](HOWTO-8-CERT-1.md) 参照）。 |
| `ttl is larger than max_ttl` | `ttl` がロールの `max_ttl`（720h）を超えている。短くする。 |
| `SignatureDoesNotMatch` | プロキシが `Host` ヘッダを書き換えている。`proxy_set_header Host $http_host;` を指定する（手順 5）。 |
| `The bucket does not exist` / virtual-hosted style で接続できない | Garage の `root_domain`（`.s3.labo.local`）と SAN・DNS（`*.s3.labo.local`）が一致していない（手順 2）。 |
| 大きなファイルのアップロードで `413` | `client_max_body_size 0;` を指定する（手順 5）。 |
| `502 Bad Gateway` | Garage が停止している、または `api_bind_addr` が `127.0.0.1:3900` で待ち受けていない。`sudo systemctl status garage` で確認する。 |
| `nginx -t` で秘密鍵のエラー | 鍵と証明書の組み合わせが違う。`openssl x509 -noout -pubkey -in crt` と `openssl pkey -pubout -in key` の出力を比較する。 |
