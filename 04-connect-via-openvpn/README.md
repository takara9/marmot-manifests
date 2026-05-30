## PCにインストールした VPN クライアント経由で、marmot内部のプライベートネットワークへ接続する

#### 管理用ネットワークをデプロイ

```console
$ mactl create -f net-admin.yaml 
リソースの作成要求が受け入れられました。ID: <nil>

$ mactl get net
NAME            NODE       BRIDGE        STATUS        AGE       IP-NET        
----            ---------  -----------   ----------    ---       --------------
private         marmot-fre  br-29d56      ACTIVE        53m       172.16.10.0/24
default         marmot-fre  virbr0        ACTIVE        29m       -             
host-bridge     marmot-fre  br0           ACTIVE        29m       -             
admin           marmot-fre  br-23024      ACTIVE        5s        172.16.20.0/24
```

#### 管理対象サーバーをデプロイ

```console
$ mactl create -f server-40.yaml 
リソースの作成要求が受け入れられました。ID: dd1d4

$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-40        marmot-fre    RUNNING       1    1024     N/A              default        
                                                           172.16.20.2      admin  
```

#### VPNゲートウェイをデプロイ

```console
$ mactl apply -f vpn-gw.yaml 
リソースが作成されました。ID: <nil>

$ mactl get vpngw
NAME            INTERNAL-NET    PUBLIC-IP         STATUS        AGE     
----            ------------    ---------         ------        ---     
vpn-gw          admin           192.168.1.40      ACTIVE        1m    
```

#### PCにインストールする OpenVPNクライアント

OpenVPNのクライアントであれば、どれでも良いと思います。
開発時に検証したOpenVPNクライアントは、以下の２種類です。

- MacOS: TuunelBlick (https://tunnelblick.net/)
- Windows: OpenVPN Connect (https://openvpn.net/client/)

#### アクセステスト

OpenVPNの接続ファイルをダウンロードして、OpenVPNクライアントに読み込ませます。

```console
$ mactl get vpngw vpn-gw -d
vpn profile downloaded: /home/ubuntu/marmot-manifests/04-connect-via-openvpn/vpn-gw.ovpn
```

OpenVPNで接続して、ターミナルを開いて、`ssh ubuntu@172.16.20.2` をアクセスすることで、
プライベートネットワークの仮想サーバーへ、リーチすることができます。
