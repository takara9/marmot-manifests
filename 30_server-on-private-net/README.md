# プライベートネットワークに繋がったサーバー

プライベートクラウド Marmot 上で、図の様なシステム構成を構築することができます。

## 構成について

<IMG WIRDH="500" SRC="image/pic30-1.png">

このシステムには、以下のネットワークがあります。
- `host-bridge`は、MarmotのPCが接続されたネットワークで、このネットワークまでが外部ネットワークといえます。
- `pri-net-bastion`は、踏み台サーバーへの経路で、開発や保守のために使用できるセキュアな経路です。
- `pri-net-web`は、Webサーバーのアプリケーションにアクセスするためのプライベートアドレスの経路です。
- `pri-net-db`は、Webサーバーのアプリケーションがデータベースやキャッシュにアクセスするための経路です。

外部からの入り口は、GW(ゲートウェイ）とALB(アプリケーションロードバランサー)の二つです。
- `GW`は、アクセス元範囲を限定できるゲートウェイ機能で、この構成では`GW`が公開するIPアドレスにアクセスすると、`bastion`踏み台サーバーにログインできます。これを踏み台として、WebサーバーやDBサーバなどにアクセスできます。
- `ALB`は、HTTPを負荷分散するためのL7ロードバランサーで、HTTPセッションを補足して、振り先サーバーを固定する機能を有しています。

web1〜web3サーバーは、Webアプリケーションサーバーであり、host-bridgeより向こう側の外部ネットワークからの攻撃は、ALB,GWによって遮断される形となります。web1〜web3サーバーは、`pri-net-db`で経由して、データベースやキャッシュへアクセスすることができます。

dbサーバーは、bastionやweb1〜web3の背後にあるプライベートネットワーク`pri-net-db`に接続されるサーバーで外部からは容易に侵入できない深部に配置されています。dbサーバーは、データ領域としてLVMで作成された拡張ディスクを２本持っています。


## 構築手順

### プライベートネットワークの構築

```console
$ mactl create -f pri-networks.yaml
$ mactl get network -l case=30
```

### 仮想サーバーの起動

```console
$ mactl create -f servers.yaml
$ mactl get servers -l case=30
```

### ロードバランサーとゲートウェイの起動

```console
$ mactl create -f gateways
$ mactl get alb
$ mactl get gw
```


##



## プライベートネットワークを作成

172.16.10.0/24 の ネットワークを marmot のクラスタ上に作成します。
このネットワークに直接アクセスすることはできません。

```console
$ mactl create -f private-net.yaml 
リソースの作成要求が受け入れられました。ID: <nil>
```

```console
$ mactl get net
NAME            NODE       BRIDGE        STATUS        AGE       IP-NET        
----            ---------  -----------   ----------    ---       --------------
host-bridge     marmot3    br0           ACTIVE        1d        -             
default         marmot1    virbr0        ACTIVE        16h       -             
host-bridge     marmot1    br0           ACTIVE        45m       -             
default         marmot2    virbr0        ACTIVE        45m       -             
host-bridge     marmot2    br0           ACTIVE        7m        -             
default         marmot3    virbr0        ACTIVE        7m        -             
private-net     marmot1    br-14b63      ACTIVE        2s        172.16.10.0/24
private-net     marmot2    br-14b63      WAITING       1s        172.16.10.0/24
private-net     marmot3    br-14b63      WAITING       1s        172.16.10.0/24
```

## プライベートネットワークに接続されたサーバーを起動

`mactl create -f MANIFEST-FILE`でサーバーを起動できます。
IPアドレスは、自動で割当られるので、マニフェストには、ネットワーク名だけが記述されています。

```console
$ mactl create -f server1.yaml 
リソースの作成要求が受け入れられました。ID: 29130

$ mactl get -f server1.yaml 
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server1          marmot1       RUNNING       1    1024     172.16.10.2      private-net      22s
```

試しに、`ping` コマンドで疎通できないことを確認してみます。

```console
$ ping -c 1 172.16.10.2
PING 172.16.10.2 (172.16.10.2) 56(84) bytes of data.
From 10.0.0.1 icmp_seq=1 Destination Host Unreachable

--- 172.16.10.2 ping statistics ---
1 packets transmitted, 0 received, +1 errors, 100% packet loss, time 0ms
```

## サーバーにログイン

ssh を使ってネットワーク経由でサーバーには、ログインできないので、シリアルコンソール経由でネットワークに接続します。
コンソールに入るには、`mactl console SERVER_NAME` を実行します。コマンド実行後に、Enterキーを一回押すと、ログインプロンプトが
再出力されて、画面でみることができます。

```console
$ mactl console server1

server1 login: root
Password: 
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-117-generic x86_64)
<中略>
```

サーバーの利用を開始する前に、`apt-get update` を実行してリポジトリを更新できます。

```console
root@server1:~# apt-get update
Hit:1 http://archive.ubuntu.com/ubuntu noble InRelease
Get:2 http://archive.ubuntu.com/ubuntu noble-updates InRelease [126 kB]
Get:3 http://archive.ubuntu.com/ubuntu noble-backports InRelease [126 kB]
<中略>

Fetched 42.9 MB in 8s (5430 kB/s)                                              
Reading package lists... Done
root@server1:~# 
```

