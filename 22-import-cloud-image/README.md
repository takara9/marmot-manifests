## OSイメージのロード

現在、marmot が対応するOSイメージは、Apline 3.23, Ubuntu 22.04, Ubuntu 24.04 です。

marmot を起動すると、自動的に Ubuntu 22.04/24.04 をロードして、利用可能な状態にします。
追加で、Apline 3.23 をロードすることができます。

起動した後、自動的にイメージを作成します。

```console
$ mactl get img
NAME            STATUS     SYNCED    LV     QCOW2  AGE
----            ------     ------    ------  -----  ---
ubuntu22.04     AVAILABLE  COMPLETE  mixed   yes    22m
ubuntu24.04     AVAILABLE  COMPLETE  mixed   yes    12m
```

Alpine Linux のOSイメージのロード

```console
$ mactl create -f image-alpine3.23.yaml 
リソースの作成要求が受け入れられました。ID: f9b54
```

```console
$ mactl get img alpine3
NAME            STATUS     SYNCED    LV     QCOW2  AGE
----            ------     ------    ------  -----  ---
alpine3         AVAILABLE  COMPLETE  mixed   yes    36s
```

## Alpine Linux の仮想サーバー起動

```console
$ mactl create -f server-alpine.yaml 
リソースの作成要求が受け入れられました。ID: d72d4

$ mactl get -f server-alpine.yaml 
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
alpine51         marmot2       RUNNING       1    1024     192.168.1.51     host-bridge      21s
```

```console
ubuntu@ws1:~/marmot-manifests/22-import-cloud-image$ ssh alpine@192.168.1.51
Welcome to Alpine!

The Alpine Wiki contains a large amount of how-to guides and general
information about administrating Alpine systems.
See <https://wiki.alpinelinux.org>.

Alpine release notes:
* <https://alpinelinux.org/posts/Alpine-3.23.0-released.html>

NOTE: 'sudo' is not installed by default, please use 'doas' instead.

You may change this message by editing /etc/motd.
alpine51:~$ 
```

ネットワーク周りの設定確認

```console
alpine51:~$ ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host proto kernel_lo 
       valid_lft forever preferred_lft forever
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
    link/ether 5a:8a:a5:a5:70:67 brd ff:ff:ff:ff:ff:ff
    inet 192.168.1.51/24 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 2400:2411:bc83:3a00:588a:a5ff:fea5:7067/64 scope global dynamic mngtmpaddr proto kernel_ra 
       valid_lft 14324sec preferred_lft 12524sec
    inet6 fe80::588a:a5ff:fea5:7067/64 scope link proto kernel_ll 
       valid_lft forever preferred_lft forever
alpine51:~$ ip r
default via 192.168.1.1 dev eth0 metric 1 onlink 
192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.51 
alpine51:~$ cat /etc/resolv.conf 
search labo.local
nameserver 192.168.1.9
nameserver 8.8.8.8
nameserver 8.8.4.4
```