# MySQLサーバーの構築
10.1.0.0/16, 192.168.1.0/24, localhost からアクセスを受け入れる MySQL サーバー


```console
export MYSQL_REMOTE_ADMIN_PASSWORD='komekomekome'
mactl create -f server-58.yaml 
```

## ログイン
mysql -h 10.1.1.10 -u mysqladmin -pkomekomekome

## 部品表データ投入スクリプトの実行
スクリプトの管理場所（リポジトリ内）は `playbook/roles/mysql_server/files/` です。

`playbook/setup.yaml` 実行時に、以下の 2 つのスクリプトは仮想サーバー内 `/opt/bom-tools` へ自動配置されます。

- `generate_bom_mock_data.py`
- `stress_bomdb_transactions.py`

そのため、スクリプトはサーバーへ SSH ログインして実行してください。

この環境は externally managed environment (PEP 668) のため、
仮想環境を利用します。
`playbook/setup.yaml` 実行時に `/opt/bom-tools/.venv` の作成と
`mysql-connector-python` のインストールまで自動で実施されます。

```console
ssh ubuntu@10.1.1.10
cd /opt/bom-tools
. .venv/bin/activate
```

### スキーマも含めて投入
```console
.venv/bin/python generate_bom_mock_data.py --host 127.0.0.1 --user mysqladmin --password komekomekome --database bomdb --part-count 1000 --create-schema
```

### I/O 負荷を高めるために二次インデックスを減らして投入
`--io-heavy-indexes` を付けると、`idx_part_name`, `idx_parent_effective` を削除してから投入します。
`idx_child_part` は外部キー維持に必要なため削除しません。

```console
.venv/bin/python generate_bom_mock_data.py --host 127.0.0.1 --user mysqladmin --password komekomekome --database bomdb --part-count 1000 --create-schema --io-heavy-indexes
```

### 既存スキーマに投入だけ
```console
.venv/bin/python generate_bom_mock_data.py --host 127.0.0.1 --user mysqladmin --password komekomekome --database bomdb --part-count 1000
```

