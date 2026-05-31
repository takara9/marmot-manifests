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


### 外部ネットワークからアクセス可能なIPアドレス（パブリックIP）をアサインする

server-30 の IPアドレスに、ホストネットワークのIPアドレスをアサインします。もちろん、重複割当に注意してください。マニフェストを適用する前に、`ping 192.168.1.31` を実行して、応答があれば、別のIPアドレスを割り当てる必要があります。

public-ip-gw.yaml
```yaml
apiVersion: v1
kind: Gateway
metadata:
    name: igw
spec:
    bindPublicIpAddress: 192.168.1.31    # パブリック側のIPアドレスでアクセスを許可
    internalServerName: server-30        # サーバーの名前
    internalVirtualNetwork: net-webs     # 内部側の仮想ネットワーク
    remoteCIDR: 192.168.1.0/24           # 接続を許すリモートのIPアドレス
    serverPorts:                         # リクエストを転送するポート番号のリストを以下に記述する
        - ssh                            # sshとhttpの通信を許可、それ以外は禁止    
        - http
```


### デプロイ手順

はじめに、ネットワークをデプロイします。

```console
$ mactl create -f private-net.yaml 
リソースの作成要求が受け入れられました。ID: <nil>

$ mactl get net
NAME            NODE       BRIDGE        STATUS        AGE       IP-NET        
----            ---------  -----------   ----------    ---       --------------
default         marmot-fre  virbr0        ACTIVE        3m        -             
host-bridge     marmot-fre  br0           ACTIVE        3m        -             
private         marmot-fre  br-29d56      ACTIVE        3s        172.16.10.0/24
```

次にサーバーをデプロイします。


```console
$ mactl create -f server-30.yaml 
リソースの作成要求が受け入れられました。ID: b599a

# 処理待ちです
$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-30        marmot-fre    PENDING       1    1024     N/A              private        
                                                           N/A              default 

# 実行状態になりました。
$ mactl get srv server-30
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-30        marmot-fre    RUNNING       1    1024     172.16.10.2      private        
                                                           N/A              default   
```

パブリックIPをアサインするゲートウェイをデプロイします。

```console
$ mactl create -f public-ip-gw.yaml 
リソースの作成要求が受け入れられました。ID: <nil>

$ mactl get gw
NAME            INTERNAL-NET    PUBLIC-IP         STATUS        AGE     
----            ------------    ---------         ------        ---     
igw             private         192.168.1.31      PROVISIONING  3s      

$ mactl get gw
NAME            INTERNAL-NET    PUBLIC-IP         STATUS        AGE     
----            ------------    ---------         ------        ---     
igw             private         192.168.1.31      CONFIGURING   30s     

$ mactl get gw
NAME            INTERNAL-NET    PUBLIC-IP         STATUS        AGE     
----            ------------    ---------         ------        ---     
igw             private         192.168.1.31      ACTIVE        1m   
```

#### 割当たパブリックIPから、プライベートネットワークのサーバーへのアクセス

割当たパブリックIPを使って、プライベートネットワークに接続したサーバーへアクセスします。

```console
$ ssh ubuntu@192.168.1.31
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-117-generic x86_64)
＜途中省略＞

ubuntu@server-30:~$ hostname
server-30
```
