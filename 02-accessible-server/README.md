## ホストネットワーク上にIPをアドレスを持ちアクセス可能なサーバー

ネットワーク host-birdgeを、仮想サーバーのマニフェストに指定すると、
marmotが稼働するサーバーが接続されたネットワークに接続できます。

IPアドレス、デフォルトゲートウェイ、DNSサーバーなどをマニフェストで指定することで、
パソコンが繋がるネットワーク上にサーバーが起動することになります。

## サンプルマニフェスト

このマニフェストで、ブロードバンドルーターが管理する 192.168.1.0/24 の IPネットワークに
接続して、192.168.1.20のIPアドレスを持つ仮想サーバーが起動します。

```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-20
    comment: marmotホストが繋がるネットワークに接続する仮想サーバー
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth: # 利用者の公開鍵に変更してください。
        url: https://github.com/takara9.keys
    networkInterface:
        - networkname: host-bridge  # marmot のサーバーが接続されるネットワーク
          address: 192.168.1.20     # IPアドレスを手動設定（IPアドレスの重複使用に注意)
          nameservers:              # DNSサーバー
            addresses:
                - 192.168.1.9
                - 8.8.8.8
                - 8.8.4.4
            search:                 # ドメイン名を省略可能なドメインをセット
                - labo.local
                - lab.local
          netmasklen: 24            # ネットマスク
          routes:                   # デフォルトGW ルーターのアドレスを指定
            - to: default
              via: 192.168.1.1
```


## 仮想サーバーの起動方法

```console
$ ls -la
total 16
drwxrwxr-x 2 ubuntu ubuntu 4096 May 30 04:46 .
drwxrwxr-x 5 ubuntu ubuntu 4096 May 30 04:37 ..
-rw-rw-r-- 1 ubuntu ubuntu   97 May 30 04:37 README.md
-rw-rw-r-- 1 ubuntu ubuntu  660 May 30 04:46 server-20.yaml
```

```console
$ mactl create -f server-20.yaml 
リソースの作成要求が受け入れられました。ID: 52e99

$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-20        marmot-fre    PENDING       1    1024     192.168.1.20     host-bridge    

$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-20        marmot-fre    RUNNING       1    1024     192.168.1.20     host-bridge  
```

## ログイン

仮想サーバーのIPアドレスを指定してログインできます。
仮想サーバーのsshの公開鍵は、マニフェストで指定したアドレスから取得した鍵がセットされて起動してきます。
初期のユーザーは、ubuntuが設定されています。

```
$ ssh ubuntu@192.168.1.20
Warning: Permanently added '192.168.1.20' (ED25519) to the list of known hosts.
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-117-generic x86_64)
<途中省略>
$ 
```

marmotd には、DNSサーバーの機能があり、リゾルバーのDNSサーバーの指定を、marmotdのIPアドレスに向けることで、
DNS名でのログインもできます。DNS名は、`サーバー名.仮想ネットワーク名` となります。

```
$ ssh ubuntu@server-20.host-bridge
Warning: Permanently added 'server-20.host-bridge' (ED25519) to the list of known hosts.
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-117-generic x86_64)
<途中省略>
$ 
```

## サーバーの削除

サーバーの削除は、`mactl del srv サーバー名` を実行することで、削除処理が完了すれば、リストから消えます。

