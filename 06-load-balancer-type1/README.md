# 簡易ロードバランサーを利用して、負荷分散を実施する



Webサーバー用ネットワークのマニフェスト
```yaml
apiVersion: v1
kind: VirtualNetwork
metadata:
  name: webs-net
  comment: ウェブサーバー用の仮想ネットワーク
spec:
  iPNetworkAddress: 172.16.80.0/24
  overlayMode: geneve
```


Webサーバー1 用マニフェスト  ***ラベル必要***
```yaml
apiVersion: v1
kind: Server
metadata:
    name: webserver-1
    labels:
    　　lb-enabled: "true"
    comment: Webサーバー＃１
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth:
        url: https://github.com/takara9.keys
    networkInterface:
        - networkname: webs-net
        - networkname: default
```

Webサーバー2 用マニフェスト
```yaml
apiVersion: v1
kind: Server
metadata:
    name: webserver-2
    labels:
    　　lb-enabled: "true"
    comment: Webサーバー＃２
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth:
        url: https://github.com/takara9.keys
    networkInterface:
        - networkname: webs-net
        - networkname: default
```

ロードバランサーのYMAL
```yaml
apiVersion: v1
kind: LoadBalancer
metadata:
  name: lb1
spec:
  backendMode: auto
  internalVirtualNetwork: webs-net
  serverPorts:
    - http
    - https
```

LBへパブリックIP割当るためのマニフェスト
```yaml
apiVersion: v1
kind: Gateway
metadata:
    name: lb-gw
spec:
    bindPublicIpAddress: 192.168.1.48    # パブリック側のIPアドレスでアクセスを許可
    internalServerName: lb1              # サーバーの名前
    internalVirtualNetwork: webs-net     # 内部側の仮想ネットワーク
    serverPorts:                         # リクエストを転送するポート番号のリストを以下に記述する
        - http
        - https
```