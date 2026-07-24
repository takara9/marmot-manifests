# MySQLサーバーの構築
10.1.0.0/16, 192.168.1.0/24, localhost からアクセスを受け入れる MySQL サーバー


```console
export MYSQL_REMOTE_ADMIN_PASSWORD='komekomekome'
mactl create -f server-58.yaml 
```

## ログイン
mysql -h 10.1.1.10 -u mysqladmin -pkomekomekome

## 部品表データ投入スクリプトの実行
この環境は externally managed environment (PEP 668) のため、
システム Python への pip install は失敗します。
仮想環境を使って実行してください。

```console
python3 -m venv .venv
. .venv/bin/activate
python -m pip install mysql-connector-python
```

### スキーマも含めて投入
```console
.venv/bin/python generate_bom_mock_data.py --host 10.1.1.10 --user mysqladmin --password komekomekome --database bomdb --part-count 1000 --create-schema
```

### 既存スキーマに投入だけ
```console
.venv/bin/python generate_bom_mock_data.py --host 10.1.1.10 --user mysqladmin --password komekomekome --database bomdb --part-count 1000
```



mysql> show databases;
