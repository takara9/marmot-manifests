# アプリケーションロードバランサーでWebサーバーを負荷分散する

Application LoadBalancer は、内部ネットワーク上のサーバー群をラベル選択し、1つ以上の Listener で公開 IP へ振り分ける機能です。


## 本マニフェストでの起動手順

### 1. プライベート専用ネットワークの作成

プライベートネットワークを作成する

```console
$ mactl create -f app-net.yaml
$ mactl get net
```

### 2. イメージを作成するためのWebサーバーを作成

プライベートネットワークに接続するサーバーを起動する。
```console
$ mactl create -f web-org.yaml
$ mactl get srv
```

### 3. イメージを作成
```console
$ mactl describe server web-org
$ mactl server createimage <ID> webserver1
$ mactl get image
```

### 4. Webサーバーをデプロイ
```console
$ mactl create -f webs.yaml
$ mactl get server -l app=web3
```

### 3. ロードバランサーの起動

```console
$ mactl create -f alb.yaml
```

確認方法

```console
$ mactl get alb
```

または、マニフェストを使って確認もできます。

```console
$ mactl get -f ApplicationLoadbalancer.yaml
```

起動と設定が完了して、動作を開始するまでに、約１分くらい時間が必要です。


