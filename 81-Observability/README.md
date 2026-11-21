# Observability: Prometheus / Loki / Tempo / Grafana

Ansible で、メトリクス (Prometheus)、ログ (Loki)、トレース (Tempo)、可視化 (Grafana) を構築します。プロダクトごとに role を分けており、他のプレイブックからも再利用できます。

| プロダクト | 導入方法 | ポート | 役割 |
|---|---|---|---|
| Prometheus | 公式リリースのバイナリ (SHA-256 検証) | 9090 | メトリクス保存。remote write 受信 (Alloy / Tempo から) |
| Loki | Grafana apt リポジトリ | 3100 | ログ保存 |
| Tempo | Grafana apt リポジトリ | 3200, 4317 (OTLP gRPC), 4318 (OTLP HTTP) | トレース保存。span-metrics / service-graphs を Prometheus へ書き込み |
| Grafana | Grafana apt リポジトリ | 3000 | 可視化。3種類のデータソースを自動登録 |

## 使い方

実行元には `ansible.posix` と `community.general` コレクションが必要です。

```bash
ansible-galaxy collection install ansible.posix community.general
```

```
marmot server create -f observability.yaml
# または手動実行
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
# 一部のプロダクトのみ
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml --tags loki,tempo
```

構築後は `http://grafana.labo.local:3000` にアクセスします。初期ユーザーは `admin`、パスワードは `admin-change-me` です (ラボ用の値。`grafana_admin_password` を ansible-vault 等で上書きしてください。初回起動時のみ反映されます)。

Grafana を HTTPS に切り替える手順は [Vault による Grafana 証明書の発行・設定](../68-vault-single/HOWTO-8-CERT-4.md) を参照してください。現行 Playbook のヘルスチェックと Prometheus スクレイプの変更点も記載しています。

## 構成

```
playbook/
├── install.yaml          # グループごとに role を適用
├── hosts.yaml            # inventory (プロダクトごとのグループ)
├── group_vars/all.yaml   # プロダクト間の接続先を inventory から導出
└── roles/
    ├── data_disk         # 追加ボリュームの ext4 作成・UUID マウント・fstab 登録
    ├── grafana_apt_repo  # Grafana apt リポジトリの登録 (loki / tempo / grafana が依存)
    ├── prometheus
    ├── loki
    ├── tempo
    └── grafana
```

- 各 role の変数は `roles/<role>/defaults/main.yaml` にあります。ポート、保持期間、データ保存先、バージョンなどを上書きできます。
- 設定ファイルは `roles/<role>/templates/` のテンプレートから生成し、変更時にサービスを再起動します。
- 各 role は最後にヘルスチェック (`/ready`、`/-/ready`、`/api/health`) を待ちます。
- Loki / Tempo のデータディレクトリと設定ファイルには、パッケージが作成したサービスユーザーのプライマリグループを使用します。同名のグループがなくても動作します。

## 1台構成と複数台構成

[hosts.yaml](playbook/hosts.yaml) は、1台のホスト `observability` を4つのグループ (`prometheus` / `loki` / `tempo` / `grafana`) に割り当てています。複数台に分ける場合は、ホストを追加してグループの割り当てだけ変更します。

```yaml
  hosts:
    prom01: { ansible_host: 192.168.1.12 }
    obs01:  { ansible_host: 192.168.1.11 }
  children:
    prometheus: { hosts: { prom01: } }
    loki:       { hosts: { obs01: } }
    tempo:      { hosts: { obs01: } }
    grafana:    { hosts: { obs01: } }
```

[group_vars/all.yaml](playbook/group_vars/all.yaml) が、各グループの先頭ホストから次の接続先を導出します。

- Grafana のデータソース URL
- Tempo の Prometheus remote write 先
- Prometheus のスクレイプ対象 (Loki / Tempo / Grafana のメトリクス)

## 他のプレイブックでの再利用

role を単体で呼び出せます。データソースを持たない Loki のみのサーバー、といった使い方ができます。

```yaml
- hosts: logserver
  become: true
  roles:
    - role: loki
      vars:
        loki_retention_period: 720h
```

`grafana` role の `grafana_ini_settings` で `grafana.ini` の任意の設定を追加できます (例: [Keycloak との SSO](../77-keycloak/HOWTO-3-grafana.md))。データソースの URL を空文字にすると、そのデータソースを登録しません。

## 他サーバーからの送信

[57-setup-with-node-exporter-alloy](../57-setup-with-node-exporter-alloy) の Alloy の送信先を、このサーバーに向けます。

| 種別 | 送信先 |
|---|---|
| ログ (Loki) | `http://192.168.1.11:3100/loki/api/v1/push` |
| メトリクス (Prometheus remote write) | `http://192.168.1.11:9090/api/v1/write` |
| トレース (OTLP) | `192.168.1.11:4317` (gRPC) / `http://192.168.1.11:4318` (HTTP) |

## 注意点

- Tempo は 3.x の単一バイナリ構成 (Kafka 不要、ローカルディスク保存) です。Tempo 3.x の既定の保存先が `/var/tempo` のため、設定で `tempo_data_dir` 配下に変更しています。
- [observability.yaml](observability.yaml) の追加ボリュームは、各プロダクトの導入前に次のようにマウントします。VM 内のデバイス名が異なる場合は、[group_vars/all.yaml](playbook/group_vars/all.yaml) の `*_data_device` を変更してください。複数台構成では各ホストのデバイスに合わせて host_vars で上書きします。

  | デバイス | マウント先 | 変数 |
  |---|---|---|
  | `/dev/vdb` | `/var/lib/prometheus` | `prometheus_data_device` |
  | `/dev/vdc` | `/var/lib/loki` | `loki_data_device` |
  | `/dev/vdd` | `/var/lib/tempo` | `tempo_data_device` |

- ホスト側の LVM ボリュームは VM 内ではブロックデバイスとして扱います。未フォーマットの場合は ext4 を作成し、UUID で `/etc/fstab` に登録して再起動後もマウントします。既存の ext4 は再フォーマットしません。他のファイルシステム、パーティションのあるディスク、別用途にマウントされたディスクはエラーになります。
- 新規マウント時にサービスが稼働中、または保存先に既存データがある場合はエラーにします。サービス停止・データ移行は自動では行いません。停止・移行後に再実行してください。既に指定のボリュームがマウントされている場合は、そのデータを維持します。`--check` ではファイルシステム作成を予測しますが、UUID の取得と実際のマウントは実行しません。
- Grafana は `/var/lib/grafana` をルートディスク上で使用します。
- 認証 (Loki / Tempo / Prometheus) と TLS は有効にしていません。信頼できるネットワーク内での利用を想定したラボ構成です。
- Prometheus のバージョンは `prometheus_version`、Loki / Tempo / Grafana は `*_package_state: present` (最新) です。固定する場合は `loki_package_name: loki=3.7.10` のように指定します。
