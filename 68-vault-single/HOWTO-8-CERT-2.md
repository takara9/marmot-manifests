# Keycloak (keycloak.labo.local) の TLS 証明書を Vault で発行する

[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) で作成した中間CA（`pki_int`）から、Keycloak のサーバー証明書を発行して設定する手順です。GitLab 版は [HOWTO-8-CERT-1.md](HOWTO-8-CERT-1.md) を参照してください。

| 項目 | 値 |
|---|---|
| Keycloak | `keycloak.labo.local`（192.168.1.197、[77-keycloak](../77-keycloak/README.md)） |
| Vault | `vault.labo.local`（192.168.1.251） |
| 発行元 | `pki_int` のロール `labo-local` |

> 本書は手順書です。実際の環境で実行して確認した結果ではありません。本番に適用する前に、動作を確認してください。

## 1. 前提

- `pki` / `pki_int` が有効で、ロール `labo-local` が作成済みであること（[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) の手順 1〜3）。
- `keycloak.labo.local` が DNS で `192.168.1.197` に解決できること。
- クライアントに Root CA が信頼されていること（[HOWTO-4-CA-3.md](HOWTO-4-CA-3.md)）。Root CA は [playbook/files/root_ca.crt](playbook/files/root_ca.crt) です。
- [77-keycloak](../77-keycloak/README.md) が、証明書を外部から設定する `keycloak_tls_mode: provided` に対応していること。

`labo-local` ロールは `allowed_domains="labo.local"`、`allow_subdomains=true`、`max_ttl=720h` です。`keycloak.labo.local` はそのまま発行できますが、TTL は最大 30 日（720h）です。

## 2. 証明書を発行する

Vault サーバーで実行します。

```bash
ssh ubuntu@vault.labo.local
vault login   # Root Token

vault write -format=json pki_int/issue/labo-local \
  common_name="keycloak.labo.local" ttl=720h \
  private_key_format=pkcs8 > keycloak.json
```

`private_key_format=pkcs8` は、秘密鍵を PKCS#8 形式（`BEGIN PRIVATE KEY`）で出力する指定です。ロールが EC 鍵（`key_type=ec`）の場合、指定しないと `BEGIN EC PRIVATE KEY` 形式になります。Keycloak（Quarkus）が確実に読める PKCS#8 にしておきます。

### 別名でもアクセスする場合

IP アドレスや別名でもアクセスする場合は、`alt_names`（DNS 名）と `ip_sans`（IP アドレス）を追加します。ただし [77-keycloak](../77-keycloak/README.md) は `hostname-strict=true` のため、`keycloak_hostname` 以外の名前では Keycloak 側が正しく動作しません。証明書の SAN を増やすだけでなく、`keycloak_hostname` も合わせて検討してください。

```bash
vault write -format=json pki_int/issue/labo-local \
  common_name="keycloak.labo.local" alt_names="sso.labo.local" ip_sans="192.168.1.197" \
  ttl=720h private_key_format=pkcs8 > keycloak.json
```

`labo.local` から `lab.internal` へ移行中の場合は、[HOWTO-8-CERT-1.md](HOWTO-8-CERT-1.md) の「lab.internal と並行運用する場合」と同様に、ロールへドメインを追加して `alt_names` に `keycloak.lab.internal` を指定します。

## 3. 証明書と秘密鍵を取り出す

`certificate` に `ca_chain` を連結します。Keycloak が中間CAも提示するための設定です。

```bash
jq -r '.data.certificate, (.data.ca_chain[])' keycloak.json > keycloak.labo.local.crt
jq -r '.data.private_key' keycloak.json > keycloak.labo.local.key
chmod 600 keycloak.labo.local.key
```

内容を確認します。

```bash
openssl x509 -in keycloak.labo.local.crt -noout -subject -issuer -dates -ext subjectAltName
head -1 keycloak.labo.local.key   # -----BEGIN PRIVATE KEY----- であること
```

`keycloak.json` と `keycloak.labo.local.key` には秘密鍵が入っています。次の手順で配置したら、Vault サーバー上のファイルは削除してください。

```bash
shred -u keycloak.json keycloak.labo.local.key
```

## 4. Keycloak に設定する

[77-keycloak](../77-keycloak/README.md) の Playbook で配置します。Playbook を実行するマシンに証明書と秘密鍵をコピーしてから、変数を設定します。

```bash
# Playbook を実行するマシンで (Vault サーバーから取得。秘密鍵の取り扱いに注意)
mkdir -p ~/keycloak-tls && chmod 700 ~/keycloak-tls
scp ubuntu@vault.labo.local:keycloak.labo.local.crt ubuntu@vault.labo.local:keycloak.labo.local.key ~/keycloak-tls/
```

[77-keycloak/playbook/group_vars/all.yaml](../77-keycloak/playbook/group_vars/all.yaml) に追記するか、`-e` で指定します。

```yaml
keycloak_tls_mode: provided
keycloak_tls_cert_src: /home/ubuntu/keycloak-tls/keycloak.labo.local.crt
keycloak_tls_key_src: /home/ubuntu/keycloak-tls/keycloak.labo.local.key
keycloak_tls_ca_src: /home/ubuntu/marmot-manifests/68-vault-single/playbook/files/root_ca.crt
```

`keycloak_tls_ca_src` に Root CA を指定すると、Keycloak サーバーの信頼ストアにも登録されます。省略した場合は `keycloak_tls_verify: false` が必要です（構築時の疎通確認が失敗するため）。

Playbook を実行します（`kc.sh build` と Keycloak の再起動が自動で行われます）。

```bash
cd 77-keycloak
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml
```

秘密鍵をリポジトリにコミットしないでください。配置後、Playbook を実行したマシン上のコピーも削除してください。

```bash
shred -u ~/keycloak-tls/keycloak.labo.local.key
```

## 5. 動作を確認する

Root CA を指定して接続します。

```bash
curl --cacert root_ca.crt https://keycloak.labo.local:8443/realms/master/.well-known/openid-configuration | jq .issuer
openssl s_client -connect keycloak.labo.local:8443 -servername keycloak.labo.local -CAfile root_ca.crt </dev/null 2>/dev/null | grep -E "Verify return code|subject|issuer"
```

`issuer` が `https://keycloak.labo.local:8443/realms/master` で、`Verify return code: 0 (ok)` と表示されれば成功です。管理コンソールは `https://keycloak.labo.local:8443/admin/` です。

## 6. 更新する

TTL は最大 30 日です。期限前に、手順 2〜4 を繰り返して入れ替えます。Playbook は証明書ファイルが変わると Keycloak を再起動します。

期限を確認するには次のコマンドを使います。

```bash
ssh ubuntu@keycloak.labo.local sudo openssl x509 -in /etc/keycloak/tls/server.crt -noout -enddate
```

更新を自動化する場合は、ACME を使います（[HOWTO-5-ACME-1.md](HOWTO-5-ACME-1.md) 以降）。Keycloak 自体は ACME クライアントを持たないため、`certbot` や `acme.sh` で取得し、`/etc/keycloak/tls/server.{crt,key}` を差し替えて `systemctl restart keycloak` を実行する構成にします。

## 7. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `unable to get local issuer certificate` | クライアントに Root CA が登録されていない。または、証明書ファイルに `ca_chain` を連結していない（手順 3）。 |
| `common name ... not allowed by this role` | ロールの `allowed_domains` に該当ドメインがない。ロールを更新する（[HOWTO-8-CERT-1.md](HOWTO-8-CERT-1.md) 参照）。 |
| `ttl is larger than max_ttl` | `ttl` がロールの `max_ttl`（720h）を超えている。短くする。 |
| Keycloak 起動時に秘密鍵の読み込みエラー | 秘密鍵が PKCS#8 でない、またはパスフレーズ付き。`private_key_format=pkcs8` を指定して再発行する（手順 2）。 |
| Playbook の疎通確認で `certificate verify failed` | `keycloak_tls_ca_src` に Root CA を指定していない（手順 4）。 |
| 管理コンソールのリダイレクト先が違う | `hostname-strict=true` のため、`keycloak_hostname` と異なる名前でアクセスしている。`keycloak.labo.local` でアクセスする。 |
