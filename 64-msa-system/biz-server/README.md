# ランダム品番でBOM取得する模擬アプリ

app-63 の random-5digits API から 5 桁文字列を取得し、
`Product-xxxxx` に変換して app-62 の bom-tree API に渡す模擬アプリです。

bom-tree の結果が NotFound の場合は、再度 5 桁文字列を取得してリトライします。

```console
export REST_API_MYSQL_USER='mysqladmin'
export REST_API_MYSQL_PASSWORD='komekomekome'
export REST_API_MYSQL_HOST='10.1.1.10'
export REST_API_MYSQL_DATABASE='bomdb'
export REST_API_RANDOM_DIGITS_API_URL='http://10.1.1.15:5000/random-5digits'
export REST_API_BOM_TREE_API_URL='http://10.1.1.13:5000/bom-tree'
export REST_API_BOM_TREE_RETRY_LIMIT='20'
mactl create -f server-62.yaml
```

起動後は app-62 上で以下を実行します。

```console
curl http://127.0.0.1:5000/health
curl http://127.0.0.1:5000/mock-bom
```

`/mock-bom` は内部で次を順に実行します。

1. `http://10.1.1.15:5000/random-5digits` から 5 桁文字列を取得
2. `Product-xxxxx` を組み立てて `http://10.1.1.13:5000/bom-tree?part_name=...` を実行
3. NotFound (404 または空結果) の場合は 1 に戻ってリトライ
4. 見つかった BOM 明細を JSON で返却
