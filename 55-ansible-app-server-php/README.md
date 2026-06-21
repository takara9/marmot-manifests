# 仮想マシンをアプケーション(PHP)サーバーにする

仮想サーバーの起動後に、AnsibleでNginx,FPM,PHPを構成して、PHPのアプリケーションサーバーを
構築することができます。

## 起動方法

```console
$ mactl create -f webphp.yaml
```

## 起動の確認

上から、リスト、サーバー名を指定して個別表示、ラベルを指定して選別表示ができます。

```console
$ mactl get server 
$ mactl get server webphp
$ mactl get server -l app=webphp
```

## 確認方法

`curl`コマンドを実行することで、サーバーとクライアントのIPアドレス、ホスト名を見ることができます。

```
$ curl http://192.168.1.91/
```

## 削除

構成に使用したファイルを指定して削除、または、サーバー名を指定して削除ができます。

```console
$ mactl del -f webphp.yaml
$ mactl del server webphp
```

## エラーになった時の原因調査

テキスト形式で見やすい形式で、詳細表示します。

```console
$ mactl describe server webphp
```

JSON形式で出力して、`jq`コマンドで、エラーメッセージだけを表示できます。

```console
$ mactl get server webphp -o json |jq .[].status
```



