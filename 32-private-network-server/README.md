# プライベートネットワークに繋がったサーバーにパブリックIPを設定する

#### プライベート専用ネットワークの作成

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
```


#### Webサーバーの起動

プライベートネットワークに接続するサーバーを起動する。
プライベートネットワークで、自動的にIPアドレスをアサインするので、指定しなくても良い。

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


#### 外部ネットワークからアクセス可能なIPアドレス（パブリックIP）をアサインする

server-30 の IPアドレスに、ホストネットワークのIPアドレスをアサインします。もちろん、重複割当に注意してください。マニフェストを適用する前に、`ping 10.10.0.31` を実行して、応答があれば、別のIPアドレスを割り当てる必要があります。

public-ip-gw.yaml
```yaml
apiVersion: v1
kind: Gateway
metadata:
    name: igw
spec:
    bindPublicIpAddress: 10.10.0.31      # パブリック側のIPアドレスでアクセスを許可
    internalServerName: server-30        # サーバーの名前
    internalVirtualNetwork: net-webs     # 内部側の仮想ネットワーク
    remoteCIDR: 10.10.0.0/24             # 接続を許すリモートのIPアドレス
    serverPorts:                         # リクエストを転送するポート番号のリストを以下に記述する
        - ssh                            # sshとhttpの通信を許可、それ以外は禁止    
        - http
```


#### デプロイ

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
