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
- Playbook を実行するマシンに Ansible と `community.postgresql` コレクションがあること。

```bash
ansible-galaxy collection install community.postgresql
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
4. [keycloak.conf.j2](playbook/roles/keycloak/templates/keycloak.conf.j2) を `/opt/keycloak/conf/keycloak.conf` へ配置
5. 初期管理者を [keycloak.env.j2](playbook/roles/keycloak/templates/keycloak.env.j2) から `/etc/keycloak/keycloak.env`（mode 0640）へ配置
6. 設定変更時に `kc.sh build` を実行（`db=postgres` はビルド時オプションのため）
7. [keycloak.service.j2](playbook/roles/keycloak/templates/keycloak.service.j2) を配置し、`start --optimized` で systemd サービスとして起動・自動起動を有効化
8. HTTP ポート待ち受けと、master realm の OpenID 設定の応答を確認

変数（[defaults/main.yaml](playbook/roles/keycloak/defaults/main.yaml)）:

| 変数 | 既定値 | 説明 |
|---|---|---|
| `keycloak_version` | `26.8.2` | インストールする Keycloak のバージョン |
| `keycloak_home` | `/opt/keycloak` | インストール先 |
| `keycloak_hostname` | `keycloak.labo.local` | Keycloak の外部ホスト名 |
| `keycloak_http_port` | `8080` | HTTP ポート |

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

## Keycloak の利用方法

ブラウザで管理コンソールを開き、管理者でログインします。

- 管理コンソール: http://keycloak.labo.local:8080/admin/
- ユーザー / パスワード: `admin` / `admin-change-me`（既定値）

サーバー上の状態確認:

```bash
ssh ubuntu@keycloak.labo.local
systemctl status keycloak
journalctl -u keycloak -f
curl -s http://127.0.0.1:8080/realms/master/.well-known/openid-configuration | jq .issuer
```

`kcadm.sh` による管理操作（`/opt/keycloak/bin` は PATH に設定済み）:

```bash
sudo /opt/keycloak/bin/kcadm.sh config credentials --server http://127.0.0.1:8080 \
  --realm master --user admin
```

## 注意事項

- 初期管理者（`KC_BOOTSTRAP_ADMIN_*`）は **DB が空の初回起動時にのみ** 作成されます。後から `keycloak_admin_password` を変更しても既存ユーザーのパスワードは変わりません。管理コンソールで変更してください。
- 既定のパスワードはラボ用の仮値です。実際に使う前に変更するか、ansible-vault 等で上書きしてください。
- TLS は構成に含まれておらず、HTTP（8080）で待ち受けます。本番利用ではリバースプロキシ等で HTTPS を終端してください（`proxy-headers=xforwarded` を設定済み）。
- `/opt/keycloak/conf/keycloak.conf` には DB パスワードが平文で含まれます（mode 0640、`keycloak` ユーザーのみ参照可）。
