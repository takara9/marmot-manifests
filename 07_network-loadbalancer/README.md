# ネットワークロードバランサーでサービスを負荷分散する

Network LoadBalancer は、内部ネットワーク上のサーバー群をラベル選択し、1つ以上の Listener で公開 IP へ振り分ける機能です。



以下は HTTP Listener 1 本の最小構成例です。

```yaml
apiVersion: v1
kind: ApplicationLoadBalancer
metadata:
  name: web-alb
spec:
  bindPublicIpAddress: 192.168.10.80
  internalVirtualNetwork: app-net
  listeners:
    - name: http
      protocol: HTTP
      vipPort: 80
      backendPort: 8080
      backendSelector:
        matchLabels:
          app: web
```

## 本マニフェストでの起動手順

### 1. プライベート専用ネットワークの作成

プライベートネットワークを作成する

```console
$ mactl create -f app-net.yaml
```

確認方法

```console
$ mactl get net
```

または、マニフェストを使って確認もできます。

```console
$ mactl get -f app-net.yaml
```


### 2. Webサーバーの起動

プライベートネットワークに接続するサーバーを起動する。
```console
$ mactl create -f webs.yaml
```

確認方法

```console
$ mactl get srv
```

または、マニフェストを使って確認もできます。

```console
$ mactl get -f webs.yaml
```


### 3. ロードバランサーの起動

```console
$ mactl create -f ApplicationLoadbalancer.yaml
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


