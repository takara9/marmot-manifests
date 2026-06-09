# クラスタ横断のプライベートネットワークに繋がったサーバー

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
  name: private
  comment: ウェブサーバー用の仮想ネットワーク
spec:
  iPNetworkAddress: 172.16.10.0/24
  overlayMode: geneve   # vx-lanへ切り替えも可能
```


### Webサーバーの起動

プライベートネットワークに接続するサーバーを起動する。
プライベートネットワークで、自動的にIPアドレスをアサインします。
割当られたIPアドレスは、`mactl get srv`で確認することができます。

server-30.yaml
```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-30
    comment: ネットワークの両方に接続する構成
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth:
        url: https://github.com/takara9.keys
    networkInterface:
        - networkname: private
        - networkname: default
```

