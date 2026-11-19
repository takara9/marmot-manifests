# GitLab (gitlab.labo.local) の TLS 証明書を Vault で発行する

[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) で作成した中間CA（`pki_int`）から、GitLab のサーバー証明書を発行して設定する手順です。

| 項目 | 値 |
|---|---|
| GitLab | `gitlab.labo.local`（192.168.1.241、[61-gitlab-ce](../61-gitlab-ce/README.md)） |
| Vault | `vault.labo.local`（192.168.1.251） |
| 発行元 | `pki_int` のロール `labo-local` |

> 本書は手順書です。実際の環境で実行して確認した結果ではありません。本番に適用する前に、動作を確認してください。

## 1. 前提

- `pki` / `pki_int` が有効で、ロール `labo-local` が作成済みであること（[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) の手順 1〜3）。
- `gitlab.labo.local` が DNS で `192.168.1.241` に解決できること。
- クライアントに Root CA が信頼されていること（[HOWTO-4-CA-3.md](HOWTO-4-CA-3.md)）。Root CA は [playbook/files/root_ca.crt](playbook/files/root_ca.crt) です。

`labo-local` ロールは `allowed_domains="labo.local"`、`allow_subdomains=true`、`max_ttl=720h` です。`gitlab.labo.local` はそのまま発行できますが、TTL は最大 30 日（720h）です。

## 2. 証明書を発行する

Vault サーバーで実行します。

```bash
ssh ubuntu@vault.labo.local
vault login   # Root Token

vault write -format=json pki_int/issue/labo-local \
  common_name="gitlab.labo.local" ttl=720h > gitlab.json
```

### lab.internal と並行運用する場合

`labo.local` から `lab.internal` へ移行中は、同じ証明書を両方の名前で使えるようにします。ロールにドメインを追加し、`alt_names` に `lab.internal` 側の名前を入れます。

```bash
vault write pki_int/roles/labo-local \
  allowed_domains="labo.local,lab.internal" allow_subdomains=true \
  max_ttl=720h key_type=ec key_bits=256

vault write -format=json pki_int/issue/labo-local \
  common_name="gitlab.labo.local" alt_names="gitlab.lab.internal" ttl=720h > gitlab.json
```

`labo.local` を廃止するときに証明書を作り直さずに済みます。移行手順は [../79-BindDNS/MIGRATION-DOMAIN.md](../79-BindDNS/MIGRATION-DOMAIN.md) を参照してください。

## 3. 証明書と秘密鍵を取り出す

`certificate` に `ca_chain` を連結します。GitLab の NGINX が中間CAも提示するための設定です。

```bash
jq -r '.data.certificate, (.data.ca_chain[])' gitlab.json > gitlab.labo.local.crt
jq -r '.data.private_key' gitlab.json > gitlab.labo.local.key
chmod 600 gitlab.labo.local.key
```

内容を確認します。

```bash
openssl x509 -in gitlab.labo.local.crt -noout -subject -issuer -dates -ext subjectAltName
```

`gitlab.json` と `gitlab.labo.local.key` には秘密鍵が入っています。GitLab に配置したら、Vault サーバー上のファイルは削除してください。

```bash
shred -u gitlab.json gitlab.labo.local.key
```

## 4. GitLab に配置する

2 つのファイルを GitLab サーバー（192.168.1.241）にコピーし、次のパスに置きます。ファイル名は `<ホスト名>.crt` と `<ホスト名>.key` にします。GitLab が自動で検出します。

```bash
sudo mkdir -p /etc/gitlab/ssl
sudo cp gitlab.labo.local.crt /etc/gitlab/ssl/gitlab.labo.local.crt
sudo cp gitlab.labo.local.key /etc/gitlab/ssl/gitlab.labo.local.key
sudo chmod 600 /etc/gitlab/ssl/gitlab.labo.local.key
```

`/etc/gitlab/gitlab.rb` を編集します。

```ruby
external_url 'https://gitlab.labo.local'
nginx['redirect_http_to_https'] = true
letsencrypt['enable'] = false
```

`external_url` を `https://` にすると Let's Encrypt が自動で有効になり、内部CAの証明書と競合します。`letsencrypt['enable'] = false` を必ず指定してください。

反映します。

```bash
sudo gitlab-ctl reconfigure
```

## 5. 動作を確認する

Root CA を指定して接続します。

```bash
curl --cacert root_ca.crt -o /dev/null -w "%{http_code}\n" https://gitlab.labo.local/users/sign_in
openssl s_client -connect gitlab.labo.local:443 -servername gitlab.labo.local -CAfile root_ca.crt </dev/null 2>/dev/null | grep -E "Verify return code|subject|issuer"
```

`curl` が `200` を返し、`Verify return code: 0 (ok)` と表示されれば成功です。`/-/readiness` などのヘルスチェックは、既定ではローカルホスト以外から 404 になるため確認に使えません。

## 6. 更新する

TTL は最大 30 日です。期限前に、手順 2〜4 を繰り返して入れ替えます。

```bash
# 配置後は NGINX の再読み込みだけでよい
sudo gitlab-ctl hup nginx
```

更新を自動化する場合は、ACME を使います（[HOWTO-5-ACME-1.md](HOWTO-5-ACME-1.md) 以降）。GitLab 自体は ACME クライアントを持たないため、`certbot` や `acme.sh` で取得し、更新後に上のコマンドを実行する構成にします。

期限を確認するには次のコマンドを使います。

```bash
openssl x509 -in /etc/gitlab/ssl/gitlab.labo.local.crt -noout -enddate
```

## 7. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `unable to get local issuer certificate` | クライアントに Root CA が登録されていない。または、証明書ファイルに `ca_chain` を連結していない（手順 3）。 |
| `common name ... not allowed by this role` | ロールの `allowed_domains` に該当ドメインがない。ロールを更新する（手順 2）。 |
| `ttl is larger than max_ttl` | `ttl` がロールの `max_ttl`（720h）を超えている。短くする。 |
| `reconfigure` で Let's Encrypt のエラー | `letsencrypt['enable'] = false` が未設定。 |
| `https://` につながらない（接続拒否） | `external_url` が `http://` のままで、NGINX が 443 番を待ち受けていない。`gitlab.rb` を `https://` に変更して `reconfigure` する（手順 4）。 |
| `gitlab-ctl reconfigure` 後に HTTP でつながらない | `redirect_http_to_https` により HTTPS へ転送される。クライアントの URL を `https://` に変更する。 |
| Git の HTTP clone が `SSL certificate problem` で失敗する | Git クライアントが Root CA を信頼していない。`git config --global http.sslCAInfo <root_ca.crt のパス>` を設定する。SSH 形式（`git@gitlab.labo.local:...`）は証明書の影響を受けない。 |
