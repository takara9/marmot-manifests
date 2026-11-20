# Ceph Dashboard の TLS 証明書を Vault で発行する

[68-vault-single](../68-vault-single/README.md) の Vault（PKI シークレットエンジン）で、Ceph Dashboard のサーバー証明書を発行して設定する手順です。流れは [HOWTO-8-CERT-1.md](../68-vault-single/HOWTO-8-CERT-1.md)（GitLab 用）と同じです。

| 項目 | 値 |
|---|---|
| Ceph Dashboard | `ceph-single.host-bridge.hv0`（ポート 8443、[56-ceph-single-node](README.md)） |
| Vault | `vault.labo.local`（192.168.1.251） |
| 発行元 | `pki_int` のロール `host-bridge-hv0`（本書で新規作成） |

> 本書は手順書です。実際の環境で実行して確認した結果ではありません。本番に適用する前に、動作を確認してください。

## 1. 前提

- `pki` / `pki_int` が有効で、中間CAが設定済みであること（[HOWTO-4-CA-2.md](../68-vault-single/HOWTO-4-CA-2.md) の手順 1〜2）。
- `ceph-single.host-bridge.hv0` が DNS で Ceph サーバーに解決できること。
- クライアントに Root CA が信頼されていること（[HOWTO-4-CA-3.md](../68-vault-single/HOWTO-4-CA-3.md)）。Root CA は [root_ca.crt](../68-vault-single/playbook/files/root_ca.crt) です。

既存のロール `labo-local` は `allowed_domains="labo.local"` のため、`host-bridge.hv0` の証明書は発行できません。専用のロールを作成します。

## 2. Vault でロールを作成する

Vault サーバーで実行します。

```bash
ssh ubuntu@vault.labo.local
vault login   # Root Token

vault write pki_int/roles/host-bridge-hv0 \
  allowed_domains="host-bridge.hv0" allow_subdomains=true \
  max_ttl=720h key_type=rsa key_bits=2048
```

鍵は RSA にしています。Ceph Dashboard（Python の cherrypy）との互換性を考え、EC ではなく RSA を選びました。

## 3. 証明書を発行する

```bash
vault write -format=json pki_int/issue/host-bridge-hv0 \
  common_name="ceph-single.host-bridge.hv0" ttl=720h > ceph.json
```

IP アドレスでもアクセスする場合は、`ip_sans="<ceph-singleのIP>"` を追加します。ロールの `allow_ip_sans` は既定で true です。

`certificate` に `ca_chain` を連結して、証明書と秘密鍵を取り出します。

```bash
jq -r '.data.certificate, (.data.ca_chain[])' ceph.json > dashboard.crt
jq -r '.data.private_key' ceph.json > dashboard.key
chmod 600 dashboard.key
```

内容を確認します。

```bash
openssl x509 -in dashboard.crt -noout -subject -issuer -dates -ext subjectAltName
```

## 4. Ceph サーバーへコピーする

```bash
scp dashboard.crt dashboard.key ubuntu@ceph-single:~/
```

`ceph.json` と `dashboard.key` には秘密鍵が入っています。コピー後、Vault サーバー上のファイルは削除してください。

```bash
shred -u ceph.json dashboard.key
```

## 5. Ceph Dashboard に設定する

ceph-single で実行します。

```bash
sudo ceph dashboard set-ssl-certificate -i dashboard.crt
sudo ceph dashboard set-ssl-certificate-key -i dashboard.key

# 反映のため Dashboard モジュールを再起動
sudo ceph mgr module disable dashboard
sudo ceph mgr module enable dashboard

sudo ceph mgr services      # Dashboard の URL を確認
```

- cephadm 構成で `ceph` コマンドがコンテナー内で動く場合は、ファイルをシェルに渡します。例: `sudo cephadm shell --mount ~/dashboard.crt:/tmp/dashboard.crt --mount ~/dashboard.key:/tmp/dashboard.key` を実行し、`-i /tmp/dashboard.crt` のように `/tmp/` 側のパスを指定します。
- mgr が複数ある場合は、mgr 名を指定します。例: `ceph dashboard set-ssl-certificate mgr.<名前> -i dashboard.crt`
- 設定後、ceph-single 上の `~/dashboard.key` は削除します。

## 6. 動作を確認する

Root CA を指定して接続します。

```bash
openssl s_client -connect ceph-single.host-bridge.hv0:8443 \
  -servername ceph-single.host-bridge.hv0 -CAfile root_ca.crt </dev/null 2>/dev/null \
  | grep -E "Verify return code|subject|issuer"
```

`Verify return code: 0 (ok)` と表示されれば成功です。ブラウザーでは `https://ceph-single.host-bridge.hv0:8443/` を開きます。

## 7. 更新する

TTL は最大 30 日（720h）です。期限前に、手順 3〜5 を繰り返して入れ替えます。期限は次のコマンドで確認できます。

```bash
openssl x509 -in dashboard.crt -noout -enddate
```

## 8. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `common name ... not allowed by this role` | ロールの `allowed_domains` に該当ドメインがない。ロールを更新する（手順 2）。 |
| `ttl is larger than max_ttl` | `ttl` がロールの `max_ttl`（720h）を超えている。短くする。 |
| `unable to get local issuer certificate` | クライアントに Root CA が登録されていない。または、証明書ファイルに `ca_chain` を連結していない（手順 3）。 |
| 設定後も古い証明書が表示される | Dashboard モジュールを再起動していない。`disable` → `enable` を実行する（手順 5）。 |
| 短縮名 `ceph-single` でアクセスしたい | ロールに `allow_bare_domains=true` を追加し、発行時に `alt_names="ceph-single"` を指定する。 |
