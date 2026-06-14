# marmot コマンドリファレンスマニュアル

このドキュメントは、このリポジトリのサンプルで実際に使用している `mactl` コマンドを整理したリファレンスです。

## 前提

- CLI: `mactl`
- マニフェスト適用ディレクトリ: 各サンプル配下
- 基本形式: `mactl <subcommand> [options] [args]`

## よく使う操作フロー

1. マニフェスト作成: `mactl create -f <manifest.yaml>`
2. 状態確認: `mactl get <resource>` または `mactl get -f <manifest.yaml>`
3. サーバー接続: `mactl console <server-name>`
4. 削除: `mactl del <resource> <name>` または `mactl del -f <manifest.yaml>`

## コマンド一覧

### create

- 目的: マニフェストからリソースを作成
- 構文: `mactl create -f <manifest.yaml>`
- 例:
  - `mactl create -f server-01.yaml`
  - `mactl create -f private-net.yaml`
  - `mactl create -f app-net.yaml`
  - `mactl create -f webs.yaml`
  - `mactl create -f ApplicationLoadbalancer.yaml`
  - `mactl create -f image-alpine3.23.yaml`

### apply

- 目的: マニフェストの適用（更新を含む適用系操作）
- 構文: `mactl apply -f <manifest.yaml>`
- 例:
  - `mactl apply -f vpn-gw.yaml`

### get

- 目的: リソース状態の参照
- 構文:
  - `mactl get <resource>`
  - `mactl get <resource> <name>`
  - `mactl get -f <manifest.yaml>`
  - `mactl get <resource> -d`
- 例:
  - `mactl get srv`
  - `mactl get server`
  - `mactl get srv server-30`
  - `mactl get net`
  - `mactl get gw`
  - `mactl get vol`
  - `mactl get alb`
  - `mactl get vpngw`
  - `mactl get vpngw vpn-gw -d`
  - `mactl get img`
  - `mactl get img alpine3`
  - `mactl get -f app-net.yaml`

### console

- 目的: 仮想サーバーのコンソールへ接続
- 構文: `mactl console <server-name>`
- 例:
  - `mactl console server0`
  - `mactl console server1`

### del

- 目的: リソース削除
- 構文:
  - `mactl del <resource> <name>`
  - `mactl del -f <manifest.yaml>`
- 例:
  - `mactl del srv server-01`
  - `mactl del vol for-vg2`
  - `mactl del -f server-42.yaml`
  - `mactl del -f server-43.yaml`
  - `mactl del -f all.yaml`

### image

- 目的: イメージの一覧・入出力・削除
- 構文:
  - `mactl image list`
  - `mactl image export <image-name>`
  - `mactl image import <archive.tgz>`
  - `mactl image delete <image-id>`
- 例:
  - `mactl image list`
  - `mactl image export srv110`
  - `mactl image import marmot-machine-image-srv110.tgz`
  - `mactl image delete 07781`

### server

- 目的: サーバー関連の補助操作
- 構文:
  - `mactl server list`
  - `mactl server createimage <server-id> <image-name>`
- 例:
  - `mactl server list`
  - `mactl server createimage 1d91a srv110`

### marmot cluster

- 目的: クラスタ情報の確認
- 構文: `mactl marmot cluster`
- 例:
  - `mactl marmot cluster`

## リソース短縮名

`get` / `del` で使われる主な短縮名です。

- `srv` / `server`: Server
- `net`: Network
- `vol`: Volume
- `gw`: Gateway
- `alb`: ApplicationLoadbalancer
- `vpngw`: VPN Gateway
- `img`: Image

## サンプル別の対応先

- サーバー基本: `01-simple-server`
- ネットワーク: `04-connect-via-openvpn`, `30-server-on-private-net`, `32-private-network-server`
- ロードバランサー: `06-application-loadbalancer`, `07-network-loadbalancer`
- イメージ: `20-save-machine-image`, `21-image-export-import`, `22-import-cloud-image`
- ボリューム: `41-volume-attach`, `42-volume-create-when-creating-server`, `43-lvm-volume`, `44-iscsi-vol`
- Ansible: `50-setup-ansible`, `51-ansible-docker`

## 補足

- 一部サンプルでは `create` と `apply` の両方が使われています。運用ポリシーに合わせて統一してください。
- `mactl get srv -o json | jq ...` のような JSON パイプ処理は、詳細確認や自動化に有効です。
