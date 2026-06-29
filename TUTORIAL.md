# Marmot ハンズオンチュートリアル

このチュートリアルは、このリポジトリにあるマニフェストだけを使って、Marmot の基本操作から実践構成までを段階的に学ぶための手順書です。

対象読者:
- Marmot をこれから触る人
- `mactl` の操作を一通り覚えたい人
- ネットワーク、LB、ボリューム、Ansible まで一気に試したい人

所要時間の目安:
- 初級版: 30分
- 実践版: 2時間

---

## 学習コースの分割

このチュートリアルは、以下の 2 コースに分けて進められます。

### 初級版（30分）

目的:
- Marmot の基本操作を短時間で体験する
- サーバー作成、確認、接続、削除の流れを身につける

実施順（目安時間）:
1. 環境確認（5分）
2. default ネットワークで最初のサーバー起動（10分）
3. host-bridge で外部 SSH 接続（10分）
4. DHCP クライアント起動（5分）

実行対象:
1. `01-simple-server`
2. `02-accessible-server`
3. `03-server-as-dhcp-client`

初級版のゴール:
- `mactl create/get/console/del` を一通り実行できる
- default と host-bridge と DHCP の違いを説明できる

### 実践版（2時間）

目的:
- 実運用を意識したネットワーク、公開、LB、永続化、自動構築までを通しで体験する

実施順（目安時間）:
1. プライベートネットワークと到達性設計（35分）
2. VPN / クラスタ横断ネットワーク（20分）
3. ロードバランサーと公開経路（20分）
4. 認証・イメージ運用（20分）
5. ボリューム運用（15分）
6. Ansible 自動構築（10分）

実行対象:
1. `30-server-on-private-net`
2. `31-server-in-air-gapped`
3. `32-private-network-server`
4. `04-connect-via-openvpn`
5. `05-private-cluster-network-server`
6. `06-application-loadbalancer`
7. `07-network-loadbalancer`
8. `11-ssh-login-with-key` または `12-ssh-login-with-github-key`
9. `20-save-machine-image` と `21-image-export-import`
10. `41-volume-attach`（必要に応じて `42/43/44` を追加）
11. `50-setup-ansible`（余力があれば `51/52/53/54/55`）

実践版のゴール:
- private 環境の閉域化と公開経路（GW/VPN/LB）を設計できる
- イメージとボリュームを使った再利用・永続化ができる
- Ansible を使った初期構成の自動化を実施できる

---

## 0. 前提

- `mactl` が利用できること
- Marmot クラスタまたは単一ノードが起動済みであること
- 本リポジトリを手元に配置済みであること

最初に状態確認:

```console
mactl marmot cluster
mactl get net
mactl get srv
```


## 1. `mactl` の基本操作を覚える

実際に使う主要コマンドは以下です。

```console
mactl create -f <manifest.yaml>
mactl apply -f <manifest.yaml>
mactl get <resource>
mactl get -f <manifest.yaml>
mactl console <server-name>
mactl del <resource> <name>
mactl del -f <manifest.yaml>
```

詳細は `COMMAND_REFERENCE.md` を参照してください。


## 2. 最初のサーバーを起動する（default ネットワーク）

対象ディレクトリ: `01-simple-server`

```console
cd 01-simple-server
mactl create -f server-01.yaml
mactl get srv
```

確認ポイント:
- `RUNNING` になること
- `default` ネットワーク接続のため外部から直接 SSH できないこと

ログインはコンソール経由:

```console
mactl console server-01
```

削除:

```console
mactl del srv server-01
```


## 3. 外部から SSH できるサーバーを作る

対象ディレクトリ: `02-accessible-server`

`host-bridge` 上に固定 IP を持つサーバーを作成します。

```console
cd ../02-accessible-server
mactl create -f server-20.yaml
mactl get srv
```

接続例:

```console
ssh ubuntu@<server-20 の IP>
```

この章で理解すること:
- `host-bridge` を使うと外部ネットワークに露出できる
- DNS/GW などを manifest 側で明示できる


## 4. DHCP クライアントとして起動する

対象ディレクトリ: `03-server-as-dhcp-client`

```console
cd ../03-server-as-dhcp-client
mactl create -f server-dhcp4.yaml
mactl get srv
```

補足:
- `dhcp4: true` / `dhcp6: true` で外部 DHCP から IP 取得
- Marmot 管理画面の IP 表示が `N/A` でも、ゲスト内では IP を取得している場合があります


## 5. プライベートネットワークを使う

### 5-1. private net 上に閉じたサーバー

対象ディレクトリ: `30-server-on-private-net`

```console
cd ../30-server-on-private-net
mactl create -f private-net.yaml
mactl create -f server1.yaml
mactl get net
mactl get -f server1.yaml
```

ポイント:
- private ネットワークは外部から直接到達できない
- まずは `mactl console` でログインして検証する

### 5-2. Air-Gap 構成

対象ディレクトリ: `31-server-in-air-gapped`

```console
cd ../31-server-in-air-gapped
mactl create -f private-net.yaml
mactl create -f server0.yaml
mactl console server0
```

ポイント:
- 外部ネットワークへの経路がない孤立環境を検証できる
- 同一 private net 内サーバー間通信は可能

### 5-3. private サーバーにパブリック IP を割り当てる

対象ディレクトリ: `32-private-network-server`

```console
cd ../32-private-network-server
mactl create -f private-net.yaml
mactl create -f server-30.yaml
mactl create -f public-ip-gw.yaml
mactl get gw
```

ポイント:
- Gateway リソースで private 側サーバーを外部公開
- `serverPorts` で公開ポート制御


## 6. VPN で内部ネットワークへ接続する

対象ディレクトリ: `04-connect-via-openvpn`

```console
cd ../04-connect-via-openvpn
mactl create -f net-admin.yaml
mactl create -f server-40.yaml
mactl apply -f vpn-gw.yaml
mactl get vpngw
mactl get vpngw vpn-gw -d
```

`-d` で取得した `.ovpn` プロファイルを OpenVPN クライアントに読み込んで接続し、内部アドレスに SSH します。


## 7. クラスタ横断ネットワークを試す

対象ディレクトリ: `05-private-cluster-network-server`

```console
cd ../05-private-cluster-network-server
mactl marmot cluster
mactl create -f all.yaml
mactl get -f all.yaml
```

ポイント:
- 複数ノードに跨る private ネットワークを作成
- Gateway 経由ジャンプ (`ssh -J`) で内部ノードへ到達

片付け:

```console
mactl del -f all.yaml
```


## 8. ロードバランサーを構成する

### 8-1. ALB

対象ディレクトリ: `06-application-loadbalancer`

```console
cd ../06-application-loadbalancer
mactl create -f app-net.yaml
mactl create -f webs.yaml
mactl create -f ApplicationLoadbalancer.yaml
mactl get alb
```

### 8-2. NLB

対象ディレクトリ: `07-network-loadbalancer`

```console
cd ../07-network-loadbalancer
mactl create -f nlb-net.yaml
mactl create -f webs.yaml
mactl create -f NetworkLoadbalancer.yaml
```

ポイント:
- サーバーラベルに基づくバックエンド自動選択
- LB は `ACTIVE` になるまで数十秒〜1分程度待つ


## 9. 認証設定を学ぶ

### 9-1. root パスワード変更（非推奨）

対象: `10-change-user-password`

`rootPassword` を設定できますが、README にある通り運用では非推奨です。

### 9-2. ローカル鍵を使う

対象: `11-ssh-login-with-key`

```console
cd ../11-ssh-login-with-key
ssh-keygen -t ed25519 -f ./vmkey -C "For marmot VMs" -N ""
mactl create -f server-91.yaml
ssh ubuntu@<IP> -i vmkey
```

### 9-3. GitHub 公開鍵を使う

対象: `12-ssh-login-with-github-key`

`spec.auth.url: https://github.com/<username>.keys` を使って公開鍵配布できます。


## 10. イメージを扱う

### 10-1. 稼働中サーバーからイメージ作成

対象: `20-save-machine-image`

```console
mactl server list
mactl server createimage <SERVER-ID> <IMAGE-NAME>
mactl get img
```

### 10-2. エクスポート / インポート

対象: `21-image-export-import`

```console
mactl image export <IMAGE-NAME>
mactl image import <ARCHIVE.tgz>
mactl image list
```

### 10-3. クラウドイメージをロード

対象: `22-import-cloud-image`

```console
cd ../22-import-cloud-image
mactl create -f image-alpine3.23.yaml
mactl get img alpine3
mactl create -f server-alpine.yaml
```


## 11. サーバースペック変更の注意点

対象ディレクトリ: `08-update-server-spec`

```console
cd ../08-update-server-spec
mactl create -f server-1-cpu-x1.yaml
mactl apply -f server-1-cpu-x2.yaml
mactl apply -f server-1-mem-x2.yaml
```

重要:
- README 記載の通り、CPU/メモリ変更は不安定化リスクあり
- 実運用は「再作成」が基本推奨
- 変更後に SSH 鍵再生成が必要になるケースあり


## 12. ボリュームを使う

### 12-1. ブートボリューム拡張

対象: `40-expand-boot-volume`

`bootVolume.spec.size` と `type: qcow2` を設定して拡張します。

### 12-2. 永続ボリュームを別管理

対象: `41-volume-attach`

```console
cd ../41-volume-attach
mactl create -f volume-1.yaml
mactl create -f server-108.yaml
mactl get vol
```

### 12-3. サーバー作成時にボリューム同時作成

対象: `42-volume-create-when-creating-server`

```console
cd ../42-volume-create-when-creating-server
mactl create -f server-42.yaml
mactl get vol
```

### 12-4. LVM / iSCSI

対象: `43-lvm-volume`, `44-iscsi-vol`

LVM ボリュームや iSCSI オプション付きボリュームの作成とアタッチ例があります。


## 13. Ansible で構成を自動化する

### 13-1. 起動直後に Playbook 自動適用

対象: `50-setup-ansible`

ポイント:
- `spec.ansible.playbook`, `inventory`, `extra-args` を設定
- SSH 到達可能なネットワークと鍵設定が必要

### 13-2. Docker 構築

対象: `51-ansible-docker`

```console
cd ../51-ansible-docker
mactl create -f server-110.yaml
```

### 13-3. Kubernetes クラスタ構築

対象: `52-ansible-kubernetes`

流れ:
- `cluster-net.yaml` でネットワーク作成
- `cluster-servers.yaml` でノード起動
- `ansible-playbook` で本体と addon 適用
- `kubectl` で確認

### 13-4. アプリ構築例

対象: `53-ansible-dify`, `54-ansible-webserver`, `55-ansible-app-server-php`

- Dify: docker-compose ベース構築
- Web: Nginx 自動構築
- PHP: Nginx + PHP-FPM 構築


## 14. 片付けの基本

個別削除:

```console
mactl del srv <name>
mactl del vol <name>
```

マニフェスト単位削除:

```console
mactl del -f <manifest.yaml>
```

複数リソースを作った章は、作成時と同じ YAML 単位で削除すると漏れが減ります。


## 15. コース別チェックリスト

### 初級版（30分）

1. `01-simple-server`
2. `02-accessible-server`
3. `03-server-as-dhcp-client`

### 実践版（2時間）

1. `30-server-on-private-net`
2. `32-private-network-server`
3. `04-connect-via-openvpn`
4. `06-application-loadbalancer`
5. `41-volume-attach`
6. `50-setup-ansible`

補強トピック（任意）:
- `05-private-cluster-network-server`
- `07-network-loadbalancer`
- `20-save-machine-image`, `21-image-export-import`, `22-import-cloud-image`
- `51-ansible-docker`, `52-ansible-kubernetes`, `53-ansible-dify`


## 16. つまずきやすいポイント

- `host-bridge` の IP 重複: 事前に `ping` で未使用確認
- `PENDING` が長い: `mactl get` で状態遷移を待つ
- SSH 不可: 鍵設定、ユーザー名、GW、DNS、到達経路を確認
- Ansible 失敗: README 記載の条件 (`auth.url`, `hosts: all`, SSH 鍵) を再確認
- 変更系 (`apply`) で再起動が必要: 対話確認に `Y` が必要なケースあり


## 17. 参照先

- 目次: `README.md`
- コマンドリファレンス: `COMMAND_REFERENCE.md`
- 各手順の詳細: 各ディレクトリ配下の `README.md`
