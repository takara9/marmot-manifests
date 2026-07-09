# 仮想マシンをWeb(nginx)サーバーにする

仮想サーバーの起動後に、AnsibleでNginxを構成して、Webサーバーにすることができます。
Ansibleプレイブックを編集することで、Webページを作り変えるなど、発展させることできます。

## 起動方法

```console
$ mactl create -f server-90.yaml
```

## 起動の確認

上から、リスト、サーバー名を指定して個別表示、ラベルを指定して選別表示ができます。

```console
$ mactl get server 
$ mactl get server server-90
$ mactl get server -l app=webserver
```


## 確認方法

`curl`コマンドを実行することで、作成したWebページを見ることができます。

```
$ curl http://SERVER-IP-ADDRESS/
```

## 削除

構成に使用したファイルを指定して削除、または、サーバー名を指定して削除ができます。

```console
$ mactl del -f server-90.yaml
$ mactl del server server-90
```

## エラーになった時の原因調査

テキスト形式で見やすい形式で、詳細表示します。

```console
$ mactl describe server server-90
```

JSON形式で出力して、`jq`コマンドで、エラーメッセージだけを表示できます。

```console
$ mactl get server server-90 -o json |jq .[].status
```


