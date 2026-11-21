# Grafana (grafana.labo.local) の TLS 証明書を Vault で発行する

[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) の中間CA（`pki_int`）から証明書を発行し、[81-Observability](../81-Observability/README.md) の Grafana に設定する手順です。Grafana 本体で TLS を終端するため、nginx は不要です。

| 項目 | 値 |
|---|---|
| Grafana | `grafana.labo.local`（192.168.1.11） |
| HTTPS URL | `https://grafana.labo.local:3000/` |
| Vault | `vault.labo.local`（192.168.1.251） |
| 発行元 | `pki_int` のロール `labo-local` |
| 証明書 / 秘密鍵 | `/etc/grafana/tls/grafana.labo.local.crt` / `/etc/grafana/tls/grafana.labo.local.key` |

> 本書は手順書です。実環境での発行・設定は実施していません。証明書の配置と TLS 切り替えは手動で行います。現行 Playbook の HTTP ヘルスチェックを含む、再実行時の注意点は手順 5 に記載しています。

## 1. 前提

- `pki` / `pki_int` とロール `labo-local` が作成済みであること（[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md)）。
- `grafana.labo.local` が `192.168.1.11` に解決でき、Grafana が HTTP の 3000 番で稼働していること。
- Vault CLI、`jq`、`openssl` が利用でき、`pki_int/issue/labo-local` に証明書を発行できる権限でログインできること。
- クライアントが Root CA を信頼していること（[HOWTO-4-CA-3.md](HOWTO-4-CA-3.md)）。本リポジトリでは [playbook/files/root_ca.crt](playbook/files/root_ca.crt) を使用します。CA を更新した場合は、現在の発行元に対応する Root CA を使用してください。

`labo-local` は `allowed_domains="labo.local"`、`allow_subdomains=true`、`max_ttl=720h` のため、TTL は最大 30 日です。更新を自動化しない場合も、有効期限前に必ず再発行してください。

## 2. Vault で証明書を発行する

Vault サーバーで実行します。JSON に秘密鍵が含まれるため、ファイル作成前に `umask 077` を設定します。

```bash
ssh ubuntu@vault.labo.local
vault login

umask 077
mkdir -p ~/grafana-tls
chmod 700 ~/grafana-tls
cd ~/grafana-tls

vault write -format=json pki_int/issue/labo-local \
  common_name="grafana.labo.local" ip_sans="192.168.1.11" \
  ttl=720h private_key_format=pkcs8 > grafana.json
```

  ### ws1 など、Vault サーバー以外から実行する場合

  `VAULT_ADDR` が未設定の場合、Vault CLI は実行元の `https://127.0.0.1:8200` に接続します。ws1 上で Vault が動いていなければ `connect: connection refused` になります。接続先を明示してから、状態確認・ログイン・発行を行ってください。

  ```bash
  export VAULT_ADDR="https://vault.labo.local:8200"
  vault status
  vault login

  umask 077
  mkdir -p ~/grafana-tls
  chmod 700 ~/grafana-tls
  cd ~/grafana-tls

  vault write -format=json pki_int/issue/labo-local \
    common_name="grafana.labo.local" ip_sans="192.168.1.11" \
    ttl=720h private_key_format=pkcs8 > grafana.json
  ```

  Vault の HTTPS 証明書の発行元を実行元の OS が信頼していない場合は、`vault status` の前に `export VAULT_CACERT="/path/to/vault-server-ca.crt"` を設定します。ここに指定するのは **Vault サーバー自身の HTTPS 証明書を検証する CA** です。Grafana 用の発行元 Root CA と同じとは限りません。証明書検証を無効化せず、正しい CA と接続先の DNS 名を使用してください。

  `Error loading CA File: ... permission denied` の場合は、Vault に接続する前に CA ファイルの読み込みで失敗しています。`echo "$VAULT_CACERT"` と `namei -l "$VAULT_CACERT"` で、ファイルの読み取り権限と親ディレクトリの検索権限を確認してください。さらに `type -P vault` と `snap list vault` で、Snap 版かどうかを確認します。

  **Snap 版（`/snap/bin/vault`）の場合**、通常のファイル権限で読み取れても、Snap のアクセス制限で `.ssh` や `/etc/vault-ca` の証明書を読めない場合があります。権限を緩めるのではなく、確認済みの公開 CA 証明書を Vault Snap のユーザーデータ領域へコピーしてください。

  ```bash
  mkdir -p "$HOME/snap/vault/common"
  cp /etc/vault-ca/vault-server-ca.crt "$HOME/snap/vault/common/vault-server-ca.crt"
  chmod 600 "$HOME/snap/vault/common/vault-server-ca.crt"
  export VAULT_CACERT="$HOME/snap/vault/common/vault-server-ca.crt"
  export VAULT_ADDR="https://vault.labo.local:8200"
  vault status
  ```

  コピー元が `.ssh` 配下にしかない場合は、その公開 CA 証明書をコピー元に指定してください。**Snap 版では、次の `/etc/vault-ca` への配置だけではアクセス制限を解消できません。**

  **Snap 版ではない場合**、`.ssh` 全体の権限を緩めず、公開 CA 証明書だけを次のように配置できます。

  ```bash
  # VAULT_CACERT が /home/ubuntu/.ssh/root_ca.crt を指している場合の例
  sudo install -d -m 0755 -o root -g root /etc/vault-ca
  sudo install -m 0644 -o root -g root \
    /home/ubuntu/.ssh/root_ca.crt /etc/vault-ca/vault-server-ca.crt
  export VAULT_CACERT="/etc/vault-ca/vault-server-ca.crt"
  vault status
  ```

  いずれの場合も、コピー元が **公開 CA 証明書であり、Vault の HTTPS 証明書の発行元として正しいもの** であることを確認してください。秘密鍵はコピーしません。

  この方法では証明書と秘密鍵が ws1 側に生成されます。手順 3 の転送も、Vault サーバーではなくその実行元で行います。発行が失敗した場合は、空の `grafana.json` を使って抽出処理を続けないでください。

`ip_sans` は IP アドレスでも検証できるようにする指定です。DNS 名だけで接続する場合は省略できます。`private_key_format=pkcs8` で、パスフレーズなしの PEM 秘密鍵を取得します。

証明書を先頭に置き、中間CAを含む `ca_chain` を連結します。Grafana はこのファイルをクライアントへ提示します。

```bash
jq -er '.data.certificate, (.data.ca_chain[])' grafana.json > grafana.labo.local.crt
jq -er '.data.private_key' grafana.json > grafana.labo.local.key
chmod 600 grafana.json grafana.labo.local.key

openssl x509 -in grafana.labo.local.crt -noout -subject -issuer -dates -ext subjectAltName
openssl pkey -in grafana.labo.local.key -check -noout
```

SAN に `DNS:grafana.labo.local` が含まれることを確認します。

## 3. Grafana サーバーに配置する

まず Vault サーバーから、権限を制限した一時ディレクトリへ転送します。

```bash
# Vault サーバーで実行
ssh ubuntu@grafana.labo.local 'umask 077; mkdir -p ~/grafana-tls; chmod 700 ~/grafana-tls'
scp grafana.labo.local.crt grafana.labo.local.key ubuntu@grafana.labo.local:grafana-tls/
```

リポジトリを持つマシンのリポジトリルートから、Root CA も転送します。Root CA の秘密鍵は転送しません。

```bash
scp 68-vault-single/playbook/files/root_ca.crt ubuntu@grafana.labo.local:grafana-tls/
```

Grafana サーバーにログインして配置します。

```bash
ssh ubuntu@grafana.labo.local

sudo install -d -m 0750 -o root -g grafana /etc/grafana/tls
sudo install -m 0640 -o root -g grafana \
  ~/grafana-tls/grafana.labo.local.crt /etc/grafana/tls/grafana.labo.local.crt
sudo install -m 0640 -o root -g grafana \
  ~/grafana-tls/grafana.labo.local.key /etc/grafana/tls/grafana.labo.local.key

# ローカルの HTTPS 検証と、同居する Prometheus のスクレイプに使用
sudo install -m 0644 -o root -g root \
  ~/grafana-tls/root_ca.crt /usr/local/share/ca-certificates/labo-root-ca.crt
sudo update-ca-certificates

sudo -u grafana test -r /etc/grafana/tls/grafana.labo.local.crt
sudo -u grafana test -r /etc/grafana/tls/grafana.labo.local.key
```

Grafana は `grafana` ユーザーで動くため、秘密鍵を `root:root` の `0600` にすると読み込めません。`root:grafana` の `0640` にして、Grafana のみに読み取りを許可します。

## 4. Grafana の HTTPS を有効にする

Grafana サーバーの `/etc/grafana/grafana.ini` を編集します。

```bash
sudoedit /etc/grafana/grafana.ini
```

既存の `[server]` セクションを次のように変更します。同名のセクションを追加せず、既存の認証設定などは残してください。

```ini
[server]
protocol = https
http_addr = 0.0.0.0
http_port = 3000
domain = grafana.labo.local
root_url = https://grafana.labo.local:3000/
cert_file = /etc/grafana/tls/grafana.labo.local.crt
cert_key = /etc/grafana/tls/grafana.labo.local.key
```

既存の `[security]` セクションには、次の設定を追加します。

```ini
cookie_secure = true
```

3000 番を HTTPS に切り替えるため、同じポートの HTTP は使用できなくなります。443 番へ変更する手順ではありません。ファイアウォールを使用している場合は、クライアントからの TCP 3000 番を許可してください。

```bash
sudo systemctl restart grafana-server
sudo systemctl status grafana-server --no-pager
```

Keycloak SSO を使用している場合は、クライアントの Valid redirect URIs も `https://grafana.labo.local:3000/login/generic_oauth` に合わせます（[77-keycloak/HOWTO-3-grafana.md](../77-keycloak/HOWTO-3-grafana.md)）。

## 5. Ansible と Prometheus の設定を合わせる

### Ansible に TLS 設定を残す

現行の [Grafana role](../81-Observability/playbook/roles/grafana/tasks/main.yaml) は `/etc/grafana/grafana.ini` をテンプレートで生成します。手動変更だけでは、再実行時に HTTP の設定へ戻るため、[group_vars/all.yaml](../81-Observability/playbook/group_vars/all.yaml) の既存変数も変更します。

```yaml
grafana_root_url: "https://{{ grafana_domain }}:3000/"
grafana_ini_settings:
  server:
    protocol: https
    cert_file: /etc/grafana/tls/grafana.labo.local.crt
    cert_key: /etc/grafana/tls/grafana.labo.local.key
  security:
    cookie_secure: true
```

`grafana_ini_settings` が既にある場合は、既存の OAuth 等の設定を残して上記を追加してください。証明書ファイル自体の配置・更新は、現行 role では行いません。

**再実行前に、HTTP 固定のヘルスチェックも変更する必要があります。** [Grafana role の最後のタスク](../81-Observability/playbook/roles/grafana/tasks/main.yaml) の `ansible.builtin.uri` を、次の HTTPS 設定へ変更します。既存の `register`、`retries`、`delay`、`until` は残します。

```yaml
  ansible.builtin.uri:
    url: "https://{{ grafana_domain }}:{{ grafana_http_port }}/api/health"
    ca_path: /etc/ssl/certs/ca-certificates.crt
    validate_certs: true
    status_code: 200
```

接続先に `localhost` ではなく証明書の DNS 名を使い、Grafana サーバー側で名前解決・Root CA の信頼ができることを確認してください。本書では Playbook 自体は変更していないため、この変更なしで TLS 設定を反映すると、サービスが起動していても最後のヘルスチェックが失敗します。この HTTPS 固定の変更は、TLS 有効のホスト向けです。

### Prometheus の Grafana スクレイプを HTTPS にする

Grafana の `/metrics` も HTTPS になるため、Prometheus の `grafana` ジョブを変更します。[group_vars/all.yaml](../81-Observability/playbook/group_vars/all.yaml) の `prometheus_scrape_configs` 内にある既存の `grafana` ジョブだけを次のように更新し、他のジョブは残します。

```yaml
  - job_name: grafana
    scheme: https
    tls_config:
      ca_file: /etc/ssl/certs/ca-certificates.crt
      server_name: grafana.labo.local
    static_configs:
      - targets: "{{ groups['grafana'] | map('extract', hostvars, 'ansible_host') | map('regex_replace', '$', ':3000') | list }}"
```

Prometheus が別のサーバーの場合は、そのサーバーにも Root CA を登録してください。`server_name` は、IP アドレスをスクレイプ対象にしている場合でも証明書の DNS 名で検証するための指定です。`insecure_skip_verify` は使用しません。

リポジトリを持つマシンで実行します。

```bash
cd 81-Observability
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml --tags prometheus
```

Grafana から Loki / Tempo / Prometheus へのデータソース接続は、引き続き HTTP です。Grafana の HTTPS 化だけでは、これらのサーバー間通信は TLS 化されません。

## 6. 動作を確認する

リポジトリルートで、Root CA とホスト名を検証して接続します。

```bash
curl --fail --show-error --cacert 68-vault-single/playbook/files/root_ca.crt \
  https://grafana.labo.local:3000/api/health

openssl s_client -connect grafana.labo.local:3000 \
  -servername grafana.labo.local -verify_hostname grafana.labo.local \
  -verify_return_error -CAfile 68-vault-single/playbook/files/root_ca.crt </dev/null
```

ヘルスチェックの JSON が `"database": "ok"` を返し、証明書検証が `Verify return code: 0 (ok)` になれば成功です。ブラウザーでは `https://grafana.labo.local:3000/` にアクセスします。警告が出る場合は、[HOWTO-4-CA-3.md](HOWTO-4-CA-3.md) に従ってクライアントへ Root CA を登録してください。

Prometheus の Targets 画面で `grafana` ジョブが UP になっていることも確認します。

配置と疎通確認に成功した後、Vault サーバー上の `~/grafana-tls/grafana.json`・`grafana.labo.local.key` と、Grafana サーバー上の一時コピー `~/grafana-tls/grafana.labo.local.key` を削除してください。秘密鍵をリポジトリに保存・コミットしないでください。`/etc/grafana/tls/` の運用中の秘密鍵は削除しません。

## 7. 証明書を更新する

期限前に手順 2・3 を繰り返し、証明書と秘密鍵を両方入れ替えます。設定を変えない場合も、読み込み直すために Grafana を再起動してください。

```bash
# Grafana サーバーで実行
sudo systemctl restart grafana-server
sudo openssl x509 -in /etc/grafana/tls/grafana.labo.local.crt -noout -enddate
```

再起動後に手順 6 で確認します。自動更新は本書では設定しません。ACME を利用する場合は [HOWTO-5-ACME-1.md](HOWTO-5-ACME-1.md) 以降を参照し、更新後の配置・権限設定・Grafana 再起動も自動化してください。

## 8. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `VAULT_ADDR ... unset` と `127.0.0.1:8200: connect: connection refused` | Vault サーバー以外で接続先が未設定。手順 2 のように `VAULT_ADDR` を設定するか、Vault サーバーに SSH して実行する。 |
| `common name ... not allowed by this role` | `labo-local` が `labo.local` のサブドメインを許可しているか確認する。 |
| 証明書の期限が要求より短い / 発行に失敗する | ロールの最大 TTL と中間CAの残りの有効期限を確認する。 |
| `permission denied` で Grafana が起動しない | `/etc/grafana/tls` の検索権限と、`grafana` ユーザーによる証明書・秘密鍵の読み取り権限を確認する。 |
| `private key does not match public key` | 異なる発行時の証明書と秘密鍵を組み合わせている。JSON から同じ組を取り出して再配置する。 |
| `unable to get local issuer certificate` / `unknown authority` | 証明書に `ca_chain` を連結しているか、接続元が現在の Root CA を信頼しているか確認する。 |
| `certificate is not valid for ...` | SAN と接続先が一致していない。`grafana.labo.local` で接続し、Prometheus では `server_name` を設定する。 |
| Playbook の最後だけ失敗する | HTTP 固定のヘルスチェックが残っている。手順 5 の HTTPS・CA 検証設定へ変更する。 |
| Prometheus の `grafana` ジョブが DOWN | `scheme: https`、CA の登録、`server_name` を確認する。 |
| ログイン後に HTTP へ戻る | `root_url` や SSO の redirect URI が HTTP のままになっていないか確認する。 |

Grafana の起動エラーは `sudo journalctl -u grafana-server -n 100 --no-pager` で確認できます。

参考: [Grafana 公式 HTTPS 設定手順](https://grafana.com/docs/grafana/latest/setup-grafana/set-up-https/)
