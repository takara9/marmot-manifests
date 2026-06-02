# プライベートネットワークに繋がったサーバー

marmot クラスタ環境で、ノード横断のプライベート仮想ネットワークに、サーバーを立てて、疎通を確認します。
marmot では、OVNとOVSで、クラスタ内サーバー横断の仮想ネットワークを構成します。
また、OVN/OVSで一般的なgeneve、設定が簡単で広く普及している vx-lan を選択できます。

### プライベート専用ネットワークの作成

プライベートネットワークを作成する

private-net.yaml
```yaml
apiVersion: v1
kind: VirtualNetwork
metadata:
  name: private-webs
  comment: プライベート仮想ネットワーク
spec:
  iPNetworkAddress: 172.16.12.0/24
  overlayMode: geneve   # vx-lanへ切り替えも可能
```


### Webサーバーの起動

プライベートネットワークに接続するサーバーを起動する。
プライベートネットワークで、自動的にIPアドレスをアサインします。
割当られたIPアドレスは、`mactl get srv`で確認することができます。


次の仮想マシンは、private-webs, default, host-bridgeの３つのネットワークにインタフェースを持つサーバーです。
そのため、marmotのホストが接続されたネットワーク(192.168.1.0/24)からアクセスできます。

server-60.yaml
```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-60
    comment: ネットワークの両方に接続する構成
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth:
        url: https://github.com/takara9.keys
    networkInterface:
        - networkname: private-webs
        - networkname: default
        - networkname: host-bridge
          address: 192.168.1.60
          nameservers:
            addresses:
                - 192.168.1.9
                - 8.8.8.8
                - 8.8.4.4
            search:
                - labo.local
                - lab.local
          netmasklen: 24
          routes:
            - to: default
              via: 192.168.1.1
```

次の仮想マシンは、private-webs, default の２つのネットに繋がります。

```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-61
    comment: プライベートネットワークに接続するサーバー
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth:
        url: https://github.com/takara9.keys
    networkInterface:
        - networkname: private-webs
        - networkname: default
```

## デプロイ方法

個別に作成する

```console
mactl create -f private-net.yaml
mactl create -f server-60.yaml
mactl create -f server-61.yaml
mactl create -f server-62.yaml
```

連結されたYAMLファイルで、一括で作成もできます。

```console
mactl create -f all.yaml
```
