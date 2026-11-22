# 77-keycloak

Keycloak（認証・認可 / OIDC・SAML の IdP）を PostgreSQL と組み合わせて、シングルノードで構築する Marmot 仮想サーバーの定義です。

## 仮想サーバー構成

[keycloak-single.yaml](keycloak-single.yaml) で定義されています。

| 項目 | 値 |
|---|---|
| サーバー名 | `keycloak` |
| CPU | 2 |
| メモリ | 4096 MB |
| イメージ | Ubuntu 24.04 |
| SSH認証 | `ubuntu` / `keycloak` ユーザー、GitHub公開鍵（`takara9.keys`）を使用 |
| IPアドレス | `192.168.1.197`（社内DNS登録済み、`keycloak.labo.local`） |
| ネットワーク | `host-bridge`、ネットマスク `/24`、デフォルトGW `192.168.1.1` |

## 事前準備

- 社内DNSに `keycloak.labo.local` → `192.168.1.197` を登録しておくこと。
- 仮想サーバーが GitHub（Keycloak のリリース tarball）と Ubuntu のパッケージリポジトリにアクセスできること。
- Playbook を実行するマシンに Ansible と `community.postgresql` コレクションがあること（TLS証明書を自作する場合は `community.crypto` も必要）。

```bash
ansible-galaxy collection install community.postgresql community.crypto
```

## Ansible構成

- [playbook/install.yaml](playbook/install.yaml): エントリーポイント。`linux` → `postgresql` → `keycloak` の順にロールを適用。
- [playbook/hosts.yaml](playbook/hosts.yaml): インベントリ。`keycloak` ホストを対象に、`mactl-ssh` 経由でSSH接続。
- [playbook/group_vars/all.yaml](playbook/group_vars/all.yaml): 管理者・DB のユーザー名とパスワード。

### roles/linux

- `curl`, `jq`, `unzip`, `ca-certificates` をインストール
- `/etc/profile.d/keycloak-env.sh` を配置し、`PATH` に `/opt/keycloak/bin` を追加、`KC_URL` を設定

### roles/postgresql

- PostgreSQL をインストールして起動
- Keycloak 用のDBユーザーとデータベースを作成（`community.postgresql` モジュール使用）

### roles/keycloak

1. OpenJDK 21 をインストール
2. `keycloak` システムユーザーを作成
3. Keycloak を GitHub リリースから取得し `/opt/keycloak` へ展開
4. TLS証明書を用意（`keycloak_tls_mode` で選択。[tls_selfsigned.yaml](playbook/roles/keycloak/tasks/tls_selfsigned.yaml) / [tls_provided.yaml](playbook/roles/keycloak/tasks/tls_provided.yaml)。詳細は「HTTPS（TLS）」）
5. [keycloak.conf.j2](playbook/roles/keycloak/templates/keycloak.conf.j2) を `/opt/keycloak/conf/keycloak.conf` へ配置
6. 初期管理者を [keycloak.env.j2](playbook/roles/keycloak/templates/keycloak.env.j2) から `/etc/keycloak/keycloak.env`（mode 0640）へ配置
7. 設定変更時に `kc.sh build` を実行（`db=postgres` はビルド時オプションのため）
8. [keycloak.service.j2](playbook/roles/keycloak/templates/keycloak.service.j2) を配置し、`start --optimized` で systemd サービスとして起動・自動起動を有効化
9. HTTPS ポート待ち受けと、master realm の OpenID 設定の応答を確認

変数（[defaults/main.yaml](playbook/roles/keycloak/defaults/main.yaml)）:

| 変数 | 既定値 | 説明 |
|---|---|---|
| `keycloak_version` | `26.8.2` | インストールする Keycloak のバージョン |
| `keycloak_home` | `/opt/keycloak` | インストール先 |
| `keycloak_hostname` | `keycloak.labo.local` | Keycloak の外部ホスト名 |
| `keycloak_https_port` | `8443` | HTTPS ポート |
| `keycloak_http_enabled` | `false` | `true` で平文HTTPも待ち受ける |
| `keycloak_http_port` | `8080` | HTTP ポート（`keycloak_http_enabled: true` のときのみ） |
| `keycloak_tls_mode` | `selfsigned` | TLS証明書の入手方法（`selfsigned` / `provided`） |
| `keycloak_tls_dir` | `/etc/keycloak/tls` | `server.crt` / `server.key` / `ca.crt` の配置先 |
| `keycloak_tls_verify` | `true` | 起動後の疎通確認で証明書を検証するか |

TLS 関連の変数は「HTTPS（TLS）」を参照してください。

変数（[group_vars/all.yaml](playbook/group_vars/all.yaml)）:

| 変数 | 既定値 |
|---|---|
| `keycloak_admin_user` | `admin` |
| `keycloak_admin_password` | `admin-change-me` |
| `keycloak_db_name` | `keycloak` |
| `keycloak_db_user` | `keycloak` |
| `keycloak_db_password` | `keycloak-change-me` |

## 実行方法

`mactl` コマンドでマニフェストを適用すると、仮想サーバーの作成から Keycloak のセットアップまで一括で行われます。

```bash
cd 77-keycloak
mactl create -f keycloak-single.yaml
```

[keycloak-single.yaml](keycloak-single.yaml) の `spec.ansible` に `playbook: playbook/install.yaml` ・ `inventory: playbook/hosts.yaml` が指定されているため、仮想サーバー起動後に自動で Playbook が実行されます。

Playbook のみを手動で再実行する場合（デバッグ時など）:

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml
```

構文のみ確認する場合:

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml --syntax-check
```

パスワードを上書きして実行する場合:

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml \
  -e keycloak_admin_password='強いパスワード' -e keycloak_db_password='強いパスワード'
```

## HTTPS（TLS）

Keycloak は HTTPS（既定 `8443`）で待ち受けます。既定では HTTP は無効です。サーバー証明書の入手方法を `keycloak_tls_mode` で選択します。

| `keycloak_tls_mode` | 内容 |
|---|---|
| `selfsigned`（既定） | サーバー上にプライベートCAを作成し、サーバー証明書を自動発行する |
| `provided` | 別途用意した証明書と秘密鍵（Ansible実行マシン上のファイル）を配置する |

どちらの場合も、証明書と秘密鍵は `/etc/keycloak/tls/server.crt` と `server.key` に配置され、`hostname` は `https://<keycloak_hostname>:<keycloak_https_port>` に固定されます。

### 選択手順

1. 証明書をどう用意するかを決めます。
   - ラボ用途で手早く動かしたい → **`selfsigned`**（追加の準備は不要）
   - 社内CA（[63-ca-labo-local](../63-ca-labo-local/)）、ACME（[62_ca-acme](../62_ca-acme/)）、公的CA等で発行済みの証明書を使う → **`provided`**
2. 選んだモードに応じて「A」または「B」の設定をします。
3. `mactl create -f keycloak-single.yaml` で構築、または Playbook を再実行します（再実行時は `kc.sh build` と再起動が自動で行われます）。
4. 「動作確認」を実施します。

設定値は [group_vars/all.yaml](playbook/group_vars/all.yaml) に書くか、`-e` で指定します。

### A. 証明書を自身で作成する（`selfsigned`）

追加の設定は不要です。必要に応じて以下を変更できます。

| 変数 | 既定値 | 説明 |
|---|---|---|
| `keycloak_extra_san` | `[]` | 追加SAN（例: `["DNS:sso.labo.local"]`） |
| `keycloak_cert_days` | `825` | サーバー証明書の有効日数 |
| `keycloak_ca_days` | `3650` | CA証明書の有効日数 |
| `keycloak_ca_common_name` | `Keycloak Labo CA` | CA の CN |

- サーバー上に CA（`/etc/keycloak/ca/ca.{key,crt}`。秘密鍵は root 専用 0700）を作成し、サーバー証明書を発行します。
- 証明書の SAN: `keycloak.labo.local`、`localhost`、`127.0.0.1`、サーバーのIP
- CA 証明書は `/etc/keycloak/tls/ca.crt` に公開され、サーバー自身の信頼ストアにも登録されます（サーバー上の `curl` / `kcadm.sh` はそのまま検証可能）。
- クライアントには CA 証明書を取得して信頼させてください。

```bash
scp ubuntu@keycloak.labo.local:/etc/keycloak/tls/ca.crt ./keycloak-ca.crt
# Ubuntu で OS に信頼させる場合
sudo cp keycloak-ca.crt /usr/local/share/ca-certificates/keycloak-labo-ca.crt && sudo update-ca-certificates
```

### B. 他で作成した証明書を設定する（`provided`）

1. 証明書と秘密鍵を Ansible を実行するマシンに置きます（PEM形式）。
   - 証明書: 中間CAがある場合はサーバー証明書の後ろに連結したフルチェーンにします。
   - 秘密鍵: パスフレーズなしにします。
   - 証明書の SAN に `keycloak_hostname`（既定 `keycloak.labo.local`）が含まれていること。
2. 変数を設定します。

```yaml
# playbook/group_vars/all.yaml などに追記
keycloak_tls_mode: provided
keycloak_tls_cert_src: /path/to/keycloak.labo.local.fullchain.crt   # 必須
keycloak_tls_key_src: /path/to/keycloak.labo.local.key              # 必須
keycloak_tls_ca_src: /path/to/labo-ca.crt                           # 任意: 発行元CA証明書
```

| 変数 | 必須 | 説明 |
|---|---|---|
| `keycloak_tls_cert_src` | ○ | サーバー証明書（フルチェーン）のパス |
| `keycloak_tls_key_src` | ○ | 秘密鍵のパス |
| `keycloak_tls_ca_src` | － | 発行元CAの証明書。指定するとサーバーの信頼ストアに登録し、`/etc/keycloak/tls/ca.crt` にも配置する |
| `keycloak_tls_verify` | － | 起動後の疎通確認で証明書を検証するか |

- 社内CAなど OS が信頼していないCAで発行した証明書を使う場合は、`keycloak_tls_ca_src` を指定してください。指定しない場合は `keycloak_tls_verify: false` にしないと、構築時の疎通確認が失敗します。
- 公的CAの証明書であれば `keycloak_tls_ca_src` は不要です。
- 証明書を更新する場合は、ファイルを差し替えて Playbook を再実行します（Keycloak が再起動されます）。
- 秘密鍵は Ansible のログに出力されません（`no_log`）。リポジトリにコミットしないよう注意してください。

### 動作確認

```bash
curl --cacert keycloak-ca.crt https://keycloak.labo.local:8443/realms/master/.well-known/openid-configuration | jq .issuer
# 証明書の内容確認
openssl s_client -connect keycloak.labo.local:8443 -servername keycloak.labo.local </dev/null 2>/dev/null | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

## Keycloak の利用方法

アプリケーションとの連携手順:

- [Vault のログインを Keycloak で認証する](./HOWTO-2.md)
- [Grafana のログインを Keycloak で認証する](./HOWTO-3-grafana.md)
- [GitLab の認証を Keycloak に委任する](./HOWTO-3-GitLab.md)

ブラウザで管理コンソールを開き、管理者でログインします。

- 管理コンソール: https://keycloak.labo.local:8443/admin/
- ユーザー / パスワード: `admin` / `admin-change-me`（既定値）

サーバー上の状態確認:

```bash
ssh ubuntu@keycloak.labo.local
systemctl status keycloak
journalctl -u keycloak -f
curl -s https://127.0.0.1:8443/realms/master/.well-known/openid-configuration | jq .issuer
```

`kcadm.sh` による管理操作（`/opt/keycloak/bin` は PATH に設定済み）:

```bash
sudo /opt/keycloak/bin/kcadm.sh config credentials --server https://127.0.0.1:8443 \
  --realm master --user admin
```

## 注意事項

- 初期管理者（`KC_BOOTSTRAP_ADMIN_*`）は **DB が空の初回起動時にのみ** 作成されます。後から `keycloak_admin_password` を変更しても既存ユーザーのパスワードは変わりません。管理コンソールで変更してください。
- 既定のパスワードはラボ用の仮値です。実際に使う前に変更するか、ansible-vault 等で上書きしてください。
- `hostname-strict=true` のため、`keycloak_hostname` 以外の名前でアクセスするとリダイレクト等が正しく動きません。別名を使う場合は `keycloak_hostname` を変更してください。
- `/opt/keycloak/conf/keycloak.conf` には DB パスワードが平文で含まれます（mode 0640、`keycloak` ユーザーのみ参照可）。
