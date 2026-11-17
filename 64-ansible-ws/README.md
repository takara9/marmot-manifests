# 64-ansible-ws

Ansible と Event-Driven Ansible (ansible-rulebook) を使うための「Ansible ワークステーション」仮想サーバーを作成する Marmot マニフェストです。
他のサーバー（CA、Webサーバーなど）にプレイブックを流す作業用端末として利用できます。

## 構成

| ファイル | 説明 |
|---|---|
| [ansible-ws.yaml](ansible-ws.yaml) | 仮想サーバーの定義 |
| [playbook/install.yaml](playbook/install.yaml) | 起動後に自動適用される構築用プレイブック |
| [playbook/hosts.yaml](playbook/hosts.yaml) | 構築用プレイブックのインベントリ |

### 仮想サーバー

| 項目 | 値 |
|---|---|
| サーバー名 | `ansible-ws` |
| CPU / メモリ | 1 / 2048 MB |
| イメージ | Ubuntu 24.04 |
| ネットワーク | `host-bridge` |
| ログインユーザー | `ubuntu`（公開鍵は <https://github.com/takara9.keys> から取得） |
| データボリューム | 10 GB（LVM）、名前 `/var/ansible-ws`。リポジトリの保存エリア用 |

### インストールされるもの

[install.yaml](playbook/install.yaml) により、次の順で導入されます。

1. `python3-pip`、`default-jdk`（ansible-rulebook の実行に Java が必要）
2. `ansible`（pip、`--break-system-packages` 指定）
3. `ansible-rulebook`（pip）
4. `ansible.eda` コレクション（全ユーザーから使えるよう `/usr/share/ansible/collections` に導入）

## 操作手順

### 1. 作成

```console
$ mactl create -f ansible-ws.yaml
```

作成後、Ansible プレイブックが自動で適用されます。完了まで数分かかります。

### 2. 状態確認

```console
$ mactl get server
$ mactl get server ansible-ws
```

IPアドレスが割り当てられ、ステータスが正常になれば利用できます。

### 3. ログイン

GitHub に登録した公開鍵に対応する秘密鍵で、`ubuntu` ユーザーとしてログインします。

```console
$ ssh ubuntu@ansible-ws
```

名前解決できない場合は、`mactl get server ansible-ws` で確認した IP アドレスを指定してください。

### 4. インストール結果の確認

```console
$ ansible --version
$ ansible-rulebook --version
$ java -version
$ ansible-galaxy collection list | grep ansible.eda
```

### 5. 利用例

#### 他のサーバーにプレイブックを適用する

```console
$ ansible all -i inventory.yaml -m ping
$ ansible-playbook -i inventory.yaml site.yaml
```

プレイブックやインベントリは、データボリューム `/var/ansible-ws` 以下に置くと、整理しやすくなります。マウント状態は `df -h /var/ansible-ws` で確認してください。

#### ansible-rulebook でイベント駆動の自動化を試す

Webhook でイベントを受け取り、メッセージを表示するだけの最小例です（リポジトリには含まれません）。

```yaml
# rules.yaml
- name: Webhook sample
  hosts: all
  sources:
    - ansible.eda.webhook:
        host: 0.0.0.0
        port: 5000
  rules:
    - name: Print received event
      condition: event.payload.message is defined
      action:
        debug:
          msg: "{{ event.payload.message }}"
```

```console
$ ansible-rulebook --rulebook rules.yaml -i localhost, --verbose
```

別の端末から送信して、受信を確認します。

```console
$ curl -H 'Content-Type: application/json' -d '{"message": "hello"}' http://ansible-ws:5000/endpoint
```

### 6. 削除

```console
$ mactl del -f ansible-ws.yaml
$ mactl del server ansible-ws
```

どちらか一方で削除できます。データボリュームの扱いは、マニフェストの `persistent` 設定（現在はコメントアウト）に従います。必要なデータは削除前に退避してください。

## トラブルシューティング

構築に失敗した場合は、状態を表示して原因を確認します。

```console
$ mactl describe server ansible-ws
$ mactl get server ansible-ws -o json | jq .[].status
```

- `ansible-rulebook` が起動しない場合は、`java -version` で JDK が入っているか確認してください。
- `Could not find source plugin for ansible.eda.webhook` と出る場合は、`ansible.eda` コレクションが `ubuntu` ユーザーから見えていません（以前の構築用プレイブックは root 用のパスに導入していました）。既存のサーバーでは次のいずれかで解消します。
  ```console
  $ ansible-galaxy collection install ansible.eda
  ```
  または、サーバーを作り直してください。`ansible-galaxy collection list | grep ansible.eda` で確認できます。
- `pip` のインストールで `externally-managed-environment` エラーになる場合は、`--break-system-packages` が必要です（構築用プレイブックでは指定済み）。
- `ansible-galaxy collection install` はインターネット接続が必要です。
