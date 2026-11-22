# GitLab CE サーバーの構築

Ubuntu 24.04 の仮想サーバーを起動し、Ansible で GitLab CE（Omnibus パッケージ）をインストールします。
リポジトリの保存領域 `/var/opt/gitlab` は、別ディスク（LVM ボリューム 40GB、永続化）に配置されます。

## ファイル構成

| ファイル | 内容 |
|---|---|
| [server-gitlab-ce.yaml](server-gitlab-ce.yaml) | 仮想サーバーの定義（CPU 8、メモリ 16GB、固定IP、データボリューム） |
| [playbook/install.yaml](playbook/install.yaml) | データディスクのマウントと GitLab 導入のプレイブック |
| [playbook/hosts.yaml](playbook/hosts.yaml) | Ansible インベントリ |
| [playbook/roles/gitlab/vars/main.yaml](playbook/roles/gitlab/vars/main.yaml) | URL、root 初期パスワードなどの変数 |
| [playbook/roles/gitlab/tasks/main.yaml](playbook/roles/gitlab/tasks/main.yaml) | GitLab CE のインストール手順 |

## 前提事項

- marmot サーバーが稼働し、`mactl` コマンドが使えること。
- ネットワーク `host-bridge` が利用可能であること。
- 仮想サーバーが、インターネット（`packages.gitlab.com`、`github.com`）へアクセスできること。
- イメージ `ubuntu24.04` が利用できること（プレイブックは Ubuntu 24.04 以外ではエラー終了します）。
- 空きリソース：CPU 8、メモリ 16GB、ディスク（ブート + 40GB のデータボリューム）。
- Ansible の実行環境と、SSH 接続用の `/usr/local/bin/mactl-ssh`、秘密鍵 `~/.ssh/id_rsa` があること（[playbook/hosts.yaml](playbook/hosts.yaml) 参照）。
- SSH 公開鍵は `https://github.com/takara9.keys` から登録されます。**自分の GitHub アカウントの URL に変更してください**（`auth.url`）。
- 次の項目を環境に合わせて修正してください。
  - 固定IPアドレス （例 `192.168.1.241` 他のサーバーと重複しないこと）を割り当てる
  - 環境のDNSサーバー（例 `192.168.1.9`、検索ドメイン `labo.local`）環境にあわせて設定する。デフォルトGW例 `192.168.1.1`
  - `gitlab_external_url`（既定は `http://gitlab.labo.local`）。この名前が DNS で `192.168.1.241` に解決できること。DNS が無い場合は、アクセスする端末の `/etc/hosts` に追記してください。

```text
192.168.1.241 gitlab.labo.local
```

## 起動方法

```console
$ cd 61_gitlab-ce
$ mactl create -f server-gitlab-ce.yaml
```

起動の確認をします。

```console
$ mactl get server
$ mactl get server gitlab
```

仮想サーバーの起動後に Ansible が実行され、GitLab CE がインストールされます。
完了まで十数分かかります（`/-/readiness` が 200 を返すまで最大 10 分間待機します）。
うまくいかない場合は、次のコマンドで詳細を確認します。

```console
$ mactl describe server gitlab
```

## 初期状態での root ログイン

ブラウザで、`gitlab_external_url` の URL を開きます。

- URL: [http://gitlab.labo.local/](http://gitlab.labo.local/)（または `http://192.168.1.241/`）
- ユーザー名: `root`
- パスワード: [playbook/roles/gitlab/vars/main.yaml](playbook/roles/gitlab/vars/main.yaml) の `gitlab_root_password`（初期値 `Passw0rd!`）

> **注意**: 初期パスワードは公開リポジトリに記載されています。ログイン後、直ちに変更してください。
> （右上のアイコン → Edit profile → Password）
> 実運用では `ansible-vault` で暗号化するか、値を書き換えてから構築してください。

## パスワードの見つけ方

### 1. プレイブックの変数を確認する（通常はこちら）

インストール時に環境変数 `GITLAB_ROOT_PASSWORD` として `gitlab_root_password` が渡されるため、
その値が root の初期パスワードになります。この値は初回の `reconfigure` 時のみ有効です。

```console
$ grep gitlab_root_password playbook/roles/gitlab/vars/main.yaml
```

### 2. 自動生成されたパスワードファイルを確認する

`gitlab_root_password` を空にしてインストールした場合や、上記のパスワードでログインできない場合は、
サーバー内に生成されたファイルを確認します。

```console
$ ssh ubuntu@gitlab
$ sudo cat /etc/gitlab/initial_root_password
```

`Password:` の行が root の初期パスワードです。
このファイルは **インストールから 24 時間後に自動で削除**されます。

### 3. パスワードを忘れた場合（リセット）

ファイルが削除済みで、変更後のパスワードも分からない場合は、サーバー内でリセットします。

```console
$ sudo gitlab-rake "gitlab:password:reset[root]"
```

## 動作確認

Keycloak にブラウザログインの認証を委任する場合は、[GitLab の Keycloak 認証手順](../77-keycloak/HOWTO-3-GitLab.md) を参照してください。

```console
$ curl -s http://gitlab.labo.local/-/readiness
$ ssh ubuntu@gitlab sudo gitlab-ctl status
```

## 削除

```console
$ mactl del -f server-gitlab-ce.yaml
```

データボリューム（`/var/opt/gitlab`）は `persistent: true` のため、サーバー削除後も残る場合があります。
不要な場合は、ボリュームの一覧を確認して削除してください。

```console
$ mactl get vol
$ mactl del vol <name>
```
