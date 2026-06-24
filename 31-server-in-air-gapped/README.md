# 外部ネットワークに繋がらない Air-Gapサーバー

このサーバーへのアクセスは、シリアル回線端末経由で接続します。
`mactl console SERVER-NAME` は、仮想マシンの模擬シリアル回線に接続して、ログインすることができます。

外部からアクセスできないサーバーで、外部へも繋げないサーバーの構築ですが、
同一のプライベートネットワークに繋がるサーバー間では、アクセスが可能です。
これによって、Air-Gapのクラスタ構成を取ることができます。


```console
$ mactl create -f server0.yaml 
リソースの作成要求が受け入れられました。ID: 2ef8f

$ mactl get server
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server0          marmot2       RUNNING       1    1024     172.16.10.3      private-net      15s
```

```console
$ mactl console server0
ubuntu@ws1:~/marmot-manifests/31-server-in-air-gapped$ mactl console server0

server0 login: root
Password: 
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-117-generic x86_64)

root@server0:~# tty
/dev/ttyS0

root@server0:~# w
 07:51:05 up 4 min,  1 user,  load average: 0.01, 0.10, 0.06
USER     TTY      FROM             LOGIN@   IDLE   JCPU   PCPU  WHAT
root     ttyS0    -                07:46    0.00s  0.07s   ?    w
```


## ネットワーク確認

IPアドレスとルーティングについて確認しておきます。

```console
root@server0:~# ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host noprefixroute 
       valid_lft forever preferred_lft forever
2: enp1s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP group default qlen 1000
    link/ether 0a:ef:03:c6:b0:e1 brd ff:ff:ff:ff:ff:ff
    inet 172.16.10.3/24 brd 172.16.10.255 scope global enp1s0
       valid_lft forever preferred_lft forever
    inet6 fe80::8ef:3ff:fec6:b0e1/64 scope link 
       valid_lft forever preferred_lft forever

root@server0:~# ip r
172.16.10.0/24 dev enp1s0 proto kernel scope link src 172.16.10.3 
```

同じ プライベートネットワークに繋がるサーバーとの疎通はできます。

```console
root@server0:~# ping -c 3 172.16.10.2
PING 172.16.10.2 (172.16.10.2) 56(84) bytes of data.
64 bytes from 172.16.10.2: icmp_seq=1 ttl=64 time=1.99 ms
64 bytes from 172.16.10.2: icmp_seq=2 ttl=64 time=2.28 ms
64 bytes from 172.16.10.2: icmp_seq=3 ttl=64 time=0.672 ms

--- 172.16.10.2 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2004ms
rtt min/avg/max/mdev = 0.672/1.644/2.275/0.697 ms
```

