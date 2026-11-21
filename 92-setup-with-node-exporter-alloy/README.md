# 既存の仮想マシンに node-exporter と Alloy をインストールする

稼働中の Ubuntu 仮想マシンに、監視用のエージェントを Ansible で導入します。

| コンポーネント | 役割 |
|---|---|
| [node-exporter](https://github.com/prometheus/node_exporter)（`prometheus-node-exporter`） | ホストのメトリクスを `9100` 番ポートで公開 |
| [Grafana Alloy](https://grafana.com/docs/alloy/latest/) | ホストのメトリクスを Prometheus へ、systemd ジャーナルを Loki へ送信 |

仮想マシンの作成時に自動適用する方法は、[57-setup-with-node-exporter-alloy](../57-setup-with-node-exporter-alloy/README.md) を参照してください。ここでは、すでに起動している仮想マシンに、手元の端末から Playbook を適用します。

## 前提

- 対象の仮想マシンは Ubuntu（apt を使用）で、実行端末から SSH で接続できること
- 実行端末に Ansible がインストールされていること
- 送信先の Prometheus（remote write 受信が有効）と Loki が稼働していること

Ansible が未導入の場合は、次のようにインストールします。

```bash
sudo apt install -y ansible
ansible --version
```

## ファイル構成

```text
92-setup-with-node-exporter-alloy/
├── README.md
└── playbook/
    ├── hosts.yaml              # インベントリ（対象サーバー）
    ├── setup.yaml              # Playbook（ロールの適用順）
    └── roles/
        ├── node_exporter/      # node-exporter のインストールと起動
        └── alloy/              # Alloy のインストール、設定、起動
```

## 1. インベントリの設定

[playbook/hosts.yaml](playbook/hosts.yaml) に、対象の仮想マシンを記述します。

```yaml
all:
  hosts:
    server:
      ansible_host: 192.168.1.8
      ansible_user: ubuntu
      ansible_ssh_private_key_file: ~/.ssh/id_ed25519
      ansible_python_interpreter: /usr/bin/python3
```

| 項目 | 説明 |
|---|---|
| `server` | インベントリ上のホスト名。任意の名前 |
| `ansible_host` | 仮想マシンの IP アドレス（または DNS 名）。自分の環境に合わせて変更する |
| `ansible_user` | SSH のログインユーザー。`become: yes` で root 権限に昇格するため、sudo が使えるユーザーにする |
| `ansible_ssh_private_key_file` | SSH の秘密鍵。仮想マシンに公開鍵が登録されていること |
| `ansible_python_interpreter` | 対象側の Python。Ubuntu 24.04 は `/usr/bin/python3` |

複数台に適用する場合は、`hosts` にエントリを追加します。Playbook は `hosts: all` なので、すべてのホストが対象になります。

```yaml
all:
  hosts:
    server-a:
      ansible_host: 192.168.1.8
      ansible_user: ubuntu
      ansible_ssh_private_key_file: ~/.ssh/id_ed25519
      ansible_python_interpreter: /usr/bin/python3
    server-b:
      ansible_host: 192.168.1.9
      ansible_user: ubuntu
      ansible_ssh_private_key_file: ~/.ssh/id_ed25519
      ansible_python_interpreter: /usr/bin/python3
```

> 実行端末の `~/.ssh` に `id_rsa` と `id_ed25519` の両方がある場合は、`ansible_ssh_private_key_file` で使う鍵を明示してください。

## 2. 接続確認

SSH 接続と Python の実行ができることを、`ping` モジュールで確認します。

```console
$ ansible -i playbook/hosts.yaml -m ping all
server | SUCCESS => {
    "changed": false,
    "ping": "pong"
}
```

`pong` が返れば準備完了です。失敗した場合の主な原因は次のとおりです。

| 症状 | 原因と対処 |
|---|---|
| `UNREACHABLE` / `Connection timed out` | `ansible_host` の IP アドレスが誤っている、または仮想マシンが停止している |
| `Permission denied (publickey)` | 秘密鍵のパスが誤っている、または仮想マシンに公開鍵が登録されていない |
| `Host key verification failed` | 初回接続で、ホスト鍵が `known_hosts` に未登録。事前に `ssh ubuntu@<IP>` で接続して登録する |
| `/usr/bin/python3: not found` | 対象に Python 3 がない。`ansible_python_interpreter` のパスを確認する |

## 3. 送信先の設定

Alloy の送信先は、[alloy ロールの defaults](playbook/roles/alloy/defaults/main.yaml) で定義されています。環境に合わせて変更してください。

| 変数 | 既定値 | 説明 |
|---|---|---|
| `alloy_loki_write_endpoint` | `http://192.168.1.9:3100/loki/api/v1/push` | ログの送信先（Loki） |
| `alloy_prometheus_remote_write_endpoint` | `http://192.168.1.9:9090/api/v1/write` | メトリクスの送信先（Prometheus の remote write） |

ファイルを編集せずに、コマンドラインから上書きもできます。

```bash
ansible-playbook -i playbook/hosts.yaml playbook/setup.yaml \
  -e alloy_loki_write_endpoint=http://192.168.1.20:3100/loki/api/v1/push \
  -e alloy_prometheus_remote_write_endpoint=http://192.168.1.20:9090/api/v1/write
```

> Prometheus 側は、remote write を受信するために `--web.enable-remote-write-receiver` を有効にしておく必要があります。

## 4. Playbook の実行

```console
$ ansible-playbook -i playbook/hosts.yaml playbook/setup.yaml
```

[playbook/setup.yaml](playbook/setup.yaml) は、次の順にロールを適用します。

### roles/node_exporter

1. `prometheus-node-exporter` パッケージを apt でインストール
2. サービスを自動起動に設定して起動

### roles/alloy

1. `/etc/apt/keyrings` を作成し、Grafana の署名鍵を配置
2. Grafana の apt リポジトリ（`https://apt.grafana.com stable main`）を追加
3. `alloy` パッケージをインストール
4. `/etc/alloy/config.alloy` を配置（設定内容は下記）
5. サービスを自動起動に設定して起動。設定ファイルが変わったときは、ハンドラで再起動

`/etc/alloy/config.alloy` の設定内容:

| コンポーネント | 動作 |
|---|---|
| `loki.source.journal` → `loki.write` | systemd journal グループの権限でジャーナルを読み、`job=systemd-journal`、`host=<ホスト名>`、`unit=<systemd unit>` のラベルを付けて Loki へ送信 |
| `prometheus.exporter.unix` → `prometheus.scrape` | Alloy 内蔵のホストメトリクスを 10 秒間隔で収集 |
| `prometheus.relabel` | ラベル `env` が `dev` のメトリクスを破棄 |
| `prometheus.remote_write` | メトリクスを Prometheus へ送信 |

> Alloy は内蔵の `prometheus.exporter.unix` でもホストメトリクスを収集します。node-exporter（`9100`）は、Prometheus から直接スクレイプする場合や、別の監視系から参照する場合のために導入します。

Playbook は冪等なので、再実行しても問題ありません。設定を変更して再度適用すると、変更のあったタスクだけが `changed` になります。

実行結果の最後に、ホストごとの集計が表示されます。`failed=0` かつ `unreachable=0` であれば成功です。

```text
PLAY RECAP ***************************************************************
server                     : ok=10   changed=8    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

## 5. 動作確認

仮想マシンにログインして確認します。

```bash
ssh ubuntu@192.168.1.8

systemctl status prometheus-node-exporter alloy
curl -s http://localhost:9100/metrics | head
curl -s http://localhost:12345/-/ready      # Alloy のヘルスチェック
journalctl -u alloy -n 20 --no-pager
```

| 確認項目 | 期待する結果 |
|---|---|
| `systemctl status` | 両サービスが `active (running)` |
| `localhost:9100/metrics` | `node_` で始まるメトリクスが出力される |
| `localhost:12345/-/ready` | Alloy が応答する（既定の待ち受けは `127.0.0.1:12345`） |
| `journalctl -u alloy` | 送信先への接続エラー（`connection refused` 等）が出ていない |

送信先側（Prometheus / Grafana）でも確認します。

- Prometheus で `up` や `node_uname_info` を検索し、対象ホストのメトリクスが届いていること
- Grafana の Explore で、Loki に対して `{job="systemd-journal"}` を検索し、ログが届いていること
- Marmot のログだけを見るには `{job="systemd-journal", unit="marmot.service"}` を検索する（`journalctl -u marmot -f` と同じユニットのログ）

## 6. 一部だけ実行する・確認する

```bash
# 実行内容の確認だけ（変更しない）
ansible-playbook -i playbook/hosts.yaml playbook/setup.yaml --check

# 特定のホストだけに適用（インベントリのホスト名を指定）
ansible-playbook -i playbook/hosts.yaml playbook/setup.yaml --limit server-a

# 詳細ログを表示
ansible-playbook -i playbook/hosts.yaml playbook/setup.yaml -v
```

> `--check` は、Grafana リポジトリ追加前に `alloy` のインストールを検証するため、初回実行では途中で失敗することがあります。

## 7. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `apt` のロック待ちで失敗する | 起動直後の `unattended-upgrades` などが apt を使用中。数分待って再実行する |
| Grafana リポジトリの取得に失敗する | 仮想マシンから `apt.grafana.com` に到達できない。DNS、プロキシ、ファイアウォールを確認する |
| Alloy は起動するが、データが届かない | `alloy_*_endpoint` の IP アドレスやポートが誤っている。`journalctl -u alloy` と、仮想マシンからの `curl` で疎通を確認する |
| Prometheus が remote write を拒否する | Prometheus に `--web.enable-remote-write-receiver` が付いていない |
| Alloy の設定を変更したのに反映されない | 設定の構文エラーで起動に失敗している。`journalctl -u alloy` で確認する |

## 削除方法

```bash
ssh ubuntu@192.168.1.8
sudo systemctl disable --now alloy prometheus-node-exporter
sudo apt purge -y alloy prometheus-node-exporter
sudo rm -f /etc/apt/sources.list.d/grafana.list /etc/apt/keyrings/grafana.asc
```
