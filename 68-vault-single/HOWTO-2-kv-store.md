# KVストアの使い方

## サブコマンド一覧
```
    delete               Deletes versions in the KV store
    destroy              Permanently removes one or more versions in the KV store
    enable-versioning    Turns on versioning for a KV store
    get                  Retrieves data from the KV store
    list                 List data or secrets
    metadata             Interact with Vault's Key-Value storage
    patch                Sets or updates data in the KV store without overwriting
    put                  Sets or updates data in the KV store
    rollback             Rolls back to a previous version of data
    undelete             Undeletes versions in the KV store
```

```
vault kv put secret/app/db username=dbuser password='S3cret!'   # 作成・更新
vault kv get secret/app/db                                       # 取得
vault kv get -field=password secret/app/db                       # 1項目だけ取得
vault kv list secret/app                                         # 一覧
vault kv delete secret/app/db                                    # 削除（バージョンは残る）
```

## KVストアへの登録
KVストアは secret/ で指定され、app1 はキーになります。
そして、usernameとpasswordは保存されるフィールド、すなわち、キー app1 で取り出す事ができる値になります。

```console
$ vault kv put secret/app1 username=dbuser password='S3cret!'
== Secret Path ==
secret/data/app1

======= Metadata =======
Key                Value
---                -----
created_time       2026-10-31T21:23:36.134013838Z
custom_metadata    <nil>
deletion_time      n/a
destroyed          false
version            1
```

## 登録データの取得
`vault kv get` にシークレットエンジン(secret/)のキー(app1)という意味の文字列を指定して、取り出す事ができます。

```console
$ vault kv get secret/app1
== Secret Path ==
secret/data/app1

======= Metadata =======
Key                Value
---                -----
created_time       2026-10-31T21:23:36.134013838Z
custom_metadata    <nil>
deletion_time      n/a
destroyed          false
version            1

====== Data ======
Key         Value
---         -----
password    S3cret!
username    dbuser
```

これではシェルスクリプトでは、環境変数などに使い難いので、フィールドを指定することで取り出す事ができます。
```console
$ vault kv get -field=username secret/app1
dbuser

$ vault kv get -field=password secret/app1
S3cret!
```

環境変数にトークンやパスワードセットするには、次の様にできます。
```
$ PASSWD=$(vault kv get -field=password secret/app1)
$ echo $PASSWD
S3cret!
```

## 階層構造でデータを管理する

キー app1 の下に、apiとcliをつけて、それぞれのアクセストークンを設定するケースを考えます。

シークレットエンジンの下に、`app1/api`と`app1/cli` というキーで、トークンを登録します。

```console
$ vault kv put secret/app1/api token='bc7e425a-d573-11f1-9c46-6b337b9e28ff'
$ vault kv put secret/app1/cli token='eac73ec8-d573-11f1-a1f8-83b392fdada6'
```

キー`app1`の下のキーをリストするのは、`vault kv list secret1/app1`として、一つ上のレベルを指定します。これで、`app1`の下のキーのリストを得る事ができます。
```console
$ vault kv list secret/app1
Keys
----
api
cli
```

トークンを取り出すには、先ほどと同じ様に、fieldを指定して、キーをセットすることで、トークンだけを得られます。
```console
$ vault kv get -field=token secret/app1/api 
bc7e425a-d573-11f1-9c46-6b337b9e28ff

$ vault kv get -field=token secret/app1/cli 
eac73ec8-d573-11f1-a1f8-83b392fdada6
```

## 鍵の削除
`app1/api` のキーを指定して、トークンを削除します。
`vault kv delete`を実行して データが取り出されなくなった事が確認できます。
```console
$ vault kv get -field=token secret/app1/api 
bc7e425a-d573-11f1-9c46-6b337b9e28ff

$ vault kv delete secret/app1/api
Success! Data deleted (if it existed) at: secret/data/app1/api

$ vault kv get -field=token secret/app1/api 
No data found at secret/data/app1/api
```
この時点では、`vault kv undelete` で復活できます。

完全に削除するには、`vault kv destroy` を使います。
```console
$ vault kv destroy -versions=1 secret/app1/api
Success! Data written to: secret/destroy/app1/api
```


## 更新と世代管理

既存のキーに tokeを `put` することで、version=2 に書き込まれる。
```console
$ vault kv put secret/app1/cli token='2f1ae4ce-d585-11f1-aa0a-331289a9c134'
```

バージョンを指定して、トークンを取り出せます。
```console
$ vault kv get -version=1 -field=token secret/app1/cli
eac73ec8-d573-11f1-a1f8-83b392fdada6
$ vault kv get -version=2 -field=token secret/app1/cli
2f1ae4ce-d585-11f1-aa0a-331289a9c134
```

バージョンの指定しないと、最新のバージョンが取り出される
```console
$ vault kv get -field=token secret/app1/cli
2f1ae4ce-d585-11f1-aa0a-331289a9c134
```

