# Python REST APIサーバーの構築


```console
export REST_API_MYSQL_USER='mysqladmin'
export REST_API_MYSQL_PASSWORD='app-password'
export REST_API_MYSQL_HOST='10.1.1.10'
export REST_API_MYSQL_DATABASE='bomdb'
mactl create -f server-60.yaml 
```

他サーバーの MySQL に接続する Python REST API サーバー

起動後は、サーバー上で次のコマンドで疎通確認できます。

```console
curl http://127.0.0.1:5000/health
curl http://127.0.0.1:5000/db
curl http://127.0.0.1:5000/products
curl 'http://127.0.0.1:5000/bom-tree?part_name=Product-39999'
```

`/db` は MySQL への接続確認を行い、`SELECT 1` の結果を返します。`status: ok` なら MySQL へのアクセスができています。

`/products` は `Product-` で始まるプロダクトの一覧 (`part_no`, `part_name`) を返します。

`/bom-tree?part_name=...` は指定した品番の下位部品一覧を返します。`part_name` を任意の品番に変えると、その品番に対応する BOM ツリーを取得できます。



