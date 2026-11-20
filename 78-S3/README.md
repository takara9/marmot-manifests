# 78-S3

[Garage](https://garagehq.deuxfleurs.fr/) を使って、S3互換のオブジェクトストレージをシングルノードで構築する Marmot 仮想サーバーの定義です。

## 仮想サーバー構成

[s3-single.yaml](s3-single.yaml) で定義されています（サーバー名は `s3-garage`）。

| 項目 | 値 |
|---|---|
| サーバー名 | `s3-garage` |
| CPU / メモリ | 2 / 4096 MB |
| イメージ | Ubuntu 24.04 |
| IPアドレス | `192.168.1.198`（社内DNS登録済み、`garage.labo.local`） |
| ネットワーク | `host-bridge`、ネットマスク `/24`、デフォルトGW `192.168.1.1` |
| 追加ボリューム | LVM 40GB（永続化）。`/dev/vdb` として見え、`/var/lib/garage` にマウント |

## Ansible構成

- [playbook/install.yaml](playbook/install.yaml): `linux` → `data_disk` → `garage` → `garage_ui` → `nginx_tls` の順にロールを適用。
- [playbook/hosts.yaml](playbook/hosts.yaml): `s3-garage` ホストを対象に、`mactl-ssh` 経由でSSH接続。ホスト名はマニフェストのサーバー名と一致させる。
- [playbook/group_vars/all.yaml](playbook/group_vars/all.yaml): `rpc_secret` / 管理APIトークン（ラボ用の仮値）。

### roles/data_disk

追加ボリュームを Garage のデータ領域として使えるようにします。Garage のメタデータとデータ（`/var/lib/garage/{meta,data}`）がこのボリュームに置かれます。

1. `/dev/vdb` がブロックデバイスとして存在することを確認（無ければ失敗）
2. ext4 でフォーマット（既にファイルシステムがあれば再フォーマットしない）
3. UUID を指定して `/var/lib/garage` にマウントし、`/etc/fstab` に登録

変数（[defaults/main.yaml](playbook/roles/data_disk/defaults/main.yaml)）: `data_disk_dev`（`/dev/vdb`）、`data_disk_mountpoint`（`/var/lib/garage`）、`data_disk_fstype`（`ext4`）、`data_disk_label`（`garage-data`）。

### roles/garage

1. `garage` システムユーザーとデータ用ディレクトリ（`/var/lib/garage/{meta,data}`）を作成
2. Garage のバイナリを公式サイトから取得し `/usr/local/bin/garage` へ配置
3. [garage.toml.j2](playbook/roles/garage/templates/garage.toml.j2) を `/etc/garage.toml` へ配置（`replication_factor = 1`、LMDB）
4. systemd サービス `garage` を配置して起動
5. ノードにロール（ゾーン・容量）を割り当ててクラスタレイアウトを適用（未割当のときのみ）
6. アクセスキーとバケットを作成し、権限を付与（既存ならスキップ）
7. アクセスキーIDとシークレットを画面に出力

主な変数（[defaults/main.yaml](playbook/roles/garage/defaults/main.yaml)）:

| 変数 | 既定値 | 説明 |
|---|---|---|
| `garage_version` | `v2.5.0` | Garage のバージョン |
| `garage_capacity` | `10G` | ノードに割り当てる容量。追加ボリューム（40GB）の空きに合わせて変更 |
| `garage_s3_region` | `garage` | S3 リージョン名 |
| `garage_keys` | `app-key` | 作成するアクセスキー |
| `garage_buckets` | `demo`（`app-key` に read/write/owner） | 作成するバケットと権限 |

ポート: S3 API `3900` / RPC `3901`（ローカルのみ）/ Web `3902` / 管理API `3903`

### roles/garage_ui

Garage の Web 管理UI [garage-ui](https://github.com/Noooste/garage-ui) を Docker コンテナ（ホストネットワーク、systemd 管理）で稼働させます。ログインは Garage の管理トークン（`garage_admin_token`）で行います。Garage 本体の S3 認証（アクセスキー）は変わりません。

1. Docker（`docker.io`）をインストール
2. [config.yaml.j2](playbook/roles/garage_ui/templates/config.yaml.j2) を `/etc/garage-ui/config.yaml` へ配置（Garage の管理トークンを含むため `0600`）
3. systemd サービス `garage-ui` を配置して起動し、`/health` を確認

主な変数（[defaults/main.yaml](playbook/roles/garage_ui/defaults/main.yaml)）:

| 変数 | 既定値 | 説明 |
|---|---|---|
| `garage_ui_version` | `v0.14.0` | garage-ui のイメージタグ |
| `garage_ui_port` | `8080` | UI の待ち受けポート |

### roles/nginx_tls

Garage 本体は TLS に対応していないため、前段に nginx を置いて S3 API の HTTPS（443）を終端し、`127.0.0.1:3900` へ転送します。

1. nginx と openssl をインストール
2. 証明書と秘密鍵を `/etc/nginx/tls/server.{crt,key}` に用意（`selfsigned`: 自己署名を生成 / `provided`: Ansible 実行マシンのファイルを配置）
3. [garage-s3.conf.j2](playbook/roles/nginx_tls/templates/garage-s3.conf.j2) を `/etc/nginx/conf.d/garage-s3.conf` へ配置し、`nginx -t` の後に reload
4. HTTPS で S3 API に到達できること（認証なしの `403`）を確認

主な変数（[defaults/main.yaml](playbook/roles/nginx_tls/defaults/main.yaml)）:

| 変数 | 既定値 | 説明 |
|---|---|---|
| `s3_tls_mode` | `selfsigned` | `selfsigned` または `provided` |
| `s3_tls_hostname` | `s3.labo.local` | 証明書の CN と nginx の `server_name` |
| `s3_tls_extra_names` | `garage.labo.local`、`*.s3.labo.local` | 追加の SAN / `server_name`。`*.s3.labo.local` は Garage の `root_domain` に対応 |
| `s3_tls_cert_src` / `s3_tls_key_src` | 空 | `provided` のときの証明書（中間CA連結済み）と秘密鍵（PEM、パスフレーズなし） |
| `s3_tls_ui_enabled` / `s3_tls_ui_port` | `true` / `8443` | garage-ui の HTTPS 公開。有効なとき garage-ui は `127.0.0.1:8080` のみで待ち受け（`garage_ui_bind_address`） |
| `garage_s3_bind_address`（garage ロール） | `0.0.0.0` | `127.0.0.1` にすると S3 API は HTTPS 経由のみになる |

Vault（`pki_int`）で発行した証明書を使う場合は、発行手順は [HOWTO-8-CERT-3.md](../68-vault-single/HOWTO-8-CERT-3.md) を参照し、次のように指定します。

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml \
  -e s3_tls_mode=provided \
  -e s3_tls_cert_src=$HOME/s3-tls/s3.labo.local.crt \
  -e s3_tls_key_src=$HOME/s3-tls/s3.labo.local.key
```

証明書を差し替えて再実行すると、nginx が reload されます。

## 実行方法

```bash
cd 78-S3
mactl create -f s3-single.yaml
```

Playbook のみ再実行する場合:

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml
```

トークンを変更する場合は `openssl rand -hex 32` で生成した値で上書きします。

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml \
  -e garage_rpc_secret=$(openssl rand -hex 32) \
  -e garage_admin_token=$(openssl rand -hex 32) \
  -e garage_metrics_token=$(openssl rand -hex 32)
```

> `rpc_secret` は初回起動後に変更するとCLIがノードに接続できなくなります。変更する場合は構築前に行ってください。

## 使い方

Playbook の最後に表示される `Key ID` と `Secret key` を使います。後からでも確認できます。

```bash
ssh ubuntu@garage.labo.local
sudo garage key info app-key --show-secret
sudo garage status
sudo garage bucket list
```

AWS CLI での利用例（クライアント側）:

```bash
export AWS_ACCESS_KEY_ID=<Key ID>
export AWS_SECRET_ACCESS_KEY=<Secret key>
export AWS_DEFAULT_REGION=garage
export AWS_ENDPOINT_URL=https://s3.labo.local
export AWS_CA_BUNDLE=/path/to/root_ca.crt   # 自己署名のときはサーバー証明書。OS に登録済みなら不要

aws s3 ls
aws s3 cp ./file.txt s3://demo/
aws s3 ls s3://demo/
```

`aws s3` コマンドの詳しい使い方は [HOWTO-aws-s3.md](HOWTO-aws-s3.md) を参照してください。

## garage-ui へのログイン

ブラウザで `https://s3.labo.local:8443` を開き、ログイン画面で Garage の管理トークン（[group_vars/all.yaml](playbook/group_vars/all.yaml) の `garage_admin_token`）を入力します。トークンを持つ人は全操作が可能です。既定は自己署名証明書のため、ブラウザの警告が出ます。garage-ui 本体は `127.0.0.1:8080` のみで待ち受けるため、`http://…:8080` では開けません。

> 起動順序: garage-ui は起動時に Garage の管理API（3903）へ接続できないと終了します。systemd の `Restart=on-failure` で再試行されます。

### 400 Bad Request になる場合

| 400 ページの表示 | 原因と対処 |
|---|---|
| `The plain HTTP request was sent to HTTPS port` | `http://` でアクセスしている。`https://s3.labo.local:8443` を使う（現在は自動で https へ転送される）。 |
| `Request Header Or Cookie Too Large` | クッキーはポートを区別せず共有されるため、同じホスト名（`s3.labo.local`）や親ドメイン（`labo.local`）の他サービスのクッキーでヘッダが大きくなっている。nginx は `large_client_header_buffers 4 32k` で許容している。それでも出る場合は、ブラウザで該当サイトのクッキーを削除する。 |

## 注意事項

- 単一ノード・レプリカ1のため、ディスク障害でデータが失われます。ラボ用途の構成です。
- S3 API は nginx（443）で HTTPS 化されます。既定は自己署名証明書のため、クライアントは `--ca-bundle` 等で信頼させるか、`provided` で Vault の証明書を使ってください。Garage の 3900（HTTP）は既定では外部にも開いています。閉じるには `garage_s3_bind_address: 127.0.0.1` を指定します。
- `garage_rpc_secret` などの既定値はラボ用の仮値です。実際に使う前に変更してください。
- garage-ui は nginx（8443）で HTTPS 化され、既定では 8080 が外部に開きません。`s3_tls_ui_enabled: false` にすると HTTP（8080）で外部公開に戻り、管理トークンが平文で流れます。Garage の管理 API（3903）は HTTP のままなので、外部から使う場合はファイアウォール等で制限してください。
- [s3-single.yaml](s3-single.yaml) の `auth.users` に `keycloak` が残っています。不要であれば削除してください。
