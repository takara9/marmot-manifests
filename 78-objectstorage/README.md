# 78-objectstorage

[Garage](https://garagehq.deuxfleurs.fr/) を使って、S3互換のオブジェクトストレージをシングルノードで構築する Marmot 仮想サーバーの定義です。

## 仮想サーバー構成

[keycloak-single.yaml](keycloak-single.yaml) で定義されています（サーバー名は `garage`）。

| 項目 | 値 |
|---|---|
| サーバー名 | `garage` |
| CPU / メモリ | 2 / 4096 MB |
| イメージ | Ubuntu 24.04 |
| IPアドレス | `192.168.1.198`（社内DNS登録済み、`garage.labo.local`） |
| ネットワーク | `host-bridge`、ネットマスク `/24`、デフォルトGW `192.168.1.1` |

## Ansible構成

- [playbook/install.yaml](playbook/install.yaml): `linux` → `garage` の順にロールを適用。
- [playbook/hosts.yaml](playbook/hosts.yaml): `garage` ホストを対象に、`mactl-ssh` 経由でSSH接続。
- [playbook/group_vars/all.yaml](playbook/group_vars/all.yaml): `rpc_secret` / 管理APIトークン（ラボ用の仮値）。

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
| `garage_capacity` | `10G` | ノードに割り当てる容量。データ用ディスクの空きに合わせて変更 |
| `garage_s3_region` | `garage` | S3 リージョン名 |
| `garage_keys` | `app-key` | 作成するアクセスキー |
| `garage_buckets` | `demo`（`app-key` に read/write/owner） | 作成するバケットと権限 |

ポート: S3 API `3900` / RPC `3901`（ローカルのみ）/ Web `3902` / 管理API `3903`

## 実行方法

```bash
cd 78-objectstorage
mactl create -f keycloak-single.yaml
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
export AWS_ENDPOINT_URL=http://garage.labo.local:3900

aws s3 ls
aws s3 cp ./file.txt s3://demo/
aws s3 ls s3://demo/
```

## 注意事項

- 単一ノード・レプリカ1のため、ディスク障害でデータが失われます。ラボ用途の構成です。
- TLS は構成に含まれず HTTP で待ち受けます。本番ではリバースプロキシ等で HTTPS を終端してください。
- `garage_rpc_secret` などの既定値はラボ用の仮値です。実際に使う前に変更してください。
- [keycloak-single.yaml](keycloak-single.yaml) の `auth.users` に `keycloak` が残っています。不要であれば削除してください。
