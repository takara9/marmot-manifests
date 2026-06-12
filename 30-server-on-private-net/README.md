# プライベートネットワークに繋がったサーバー

プライベートネットワーク上に、marmotのサーバーが接続されたネットワークからアクセスできないサーバーを起動できます。


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

