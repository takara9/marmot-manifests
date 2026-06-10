# クラスタ横断のプライベートネットワークに繋がったサーバー

marmot クラスタ環境で、ノード横断のプライベート仮想ネットワークに、サーバーを立てて、疎通を確認します。
marmot では、OVNとOVSで、クラスタ内サーバー横断の仮想ネットワークを構成します。
また、OVN/OVSで一般的なgeneve、設定が簡単で広く普及している vx-lan を選択できます。

シングル構成でも動作しますが、クラスタ構成のmarmot環境で、試すのがお勧めです。


## プロビジョニング

クラスタ構成を確認します。

```console
$ mactl marmot cluster
NODE             HOSTID     IP              CAP_CPU CAP_MEM(MB)  TOTAL RUNNING STOPPED     VCPU  MEM(MB)    VNETS UPDATED
marmot1          61e3eba0   172.16.0.201          8      15991      0       0       0        0        0        6 2026-06-10 21:35:25
marmot2          9bffb5e3   172.16.0.202          8      15991      0       0       0        0        0        6 2026-06-10 21:35:28
marmot3          67aa9474   172.16.0.203          8      15991      0       0       0        0        0        6 2026-06-10 21:35:33
```


### 起動の確認

この結合されたYAMLファイルで、仮想ネットワーク、仮想サーバー x 3、ゲートウェイを起動できます。

```console
$ mactl create -f all.yaml 
リソースの作成要求が受け入れられました。ID: <nil>
リソースの作成要求が受け入れられました。ID: f217b
リソースの作成要求が受け入れられました。ID: 2c363
リソースの作成要求が受け入れられました。ID: 56b1c
リソースの作成要求が受け入れられました。ID: <nil>
```

### 起動の確認

create を get に変更することで、全部のオブジェクトを確認できます。

```console
ubuntu@ws1:~/marmot-manifests/05-private-cluster-network-server$ mactl get -f all.yaml 
NAME            NODE       BRIDGE        STATUS        AGE       IP-NET        
----            ---------  -----------   ----------    ---       --------------
private         marmot1    br-54a72      ACTIVE        59s       172.16.50.0/24
private         marmot2    br-54a72      ACTIVE        57s       172.16.50.0/24
private         marmot3    br-54a72      ACTIVE        57s       172.16.50.0/24

NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-50        marmot3       RUNNING       1    1024     172.16.50.3      private          59s

NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-51        marmot1       RUNNING       1    1024     172.16.50.4      private          59s

NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-52        marmot2       RUNNING       1    1024     172.16.50.2      private          59s

NAME            INTERNAL-NET    PUBLIC-IP         STATUS        AGE     
----            ------------    ---------         ------        ---     
igw             private         192.168.1.50      ACTIVE        59s  
```

サーバーに限定したリストを表示したい時は、`mactl get srv|server` を実行します。

```console
ubuntu@ws1:~/marmot-manifests/05-private-cluster-network-server$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-50        marmot3       RUNNING       1    1024     172.16.50.3      private          2m
server-51        marmot1       RUNNING       1    1024     172.16.50.4      private          2m
server-52        marmot2       RUNNING       1    1024     172.16.50.2      private          2m
```

### プライベートネットワーク上のサーバー間の疎通確認

server-50 だけが、gateway で 外部ネットワーク上にIPアドレスを持っているので、server-50を踏み台にして、
他のプライベートネットワークのサーバーにアクセスできます。

```console
ubuntu@ws1:~/marmot-manifests/05-private-cluster-network-server$ ssh 192.168.1.50 ping -c 3 172.16.50.4
Warning: Permanently added '192.168.1.50' (ED25519) to the list of known hosts.
PING 172.16.50.4 (172.16.50.4) 56(84) bytes of data.
64 bytes from 172.16.50.4: icmp_seq=1 ttl=64 time=2.35 ms
64 bytes from 172.16.50.4: icmp_seq=2 ttl=64 time=1.01 ms
64 bytes from 172.16.50.4: icmp_seq=3 ttl=64 time=3.20 ms

--- 172.16.50.4 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2003ms
rtt min/avg/max/mdev = 1.005/2.184/3.196/0.902 ms
ubuntu@ws1:~/marmot-manifests/05-private-cluster-network-server$ ssh 192.168.1.50 ping -c 3 172.16.50.2
Warning: Permanently added '192.168.1.50' (ED25519) to the list of known hosts.
PING 172.16.50.2 (172.16.50.2) 56(84) bytes of data.
64 bytes from 172.16.50.2: icmp_seq=1 ttl=64 time=1.96 ms
64 bytes from 172.16.50.2: icmp_seq=2 ttl=64 time=0.677 ms
64 bytes from 172.16.50.2: icmp_seq=3 ttl=64 time=2.43 ms

--- 172.16.50.2 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2013ms
rtt min/avg/max/mdev = 0.677/1.689/2.434/0.741 ms
```

Gatewayと紐付けられた仮想サーバーを踏み台とすることで、仮想ネットワークのサーバーにアクセスできます。

```console
$ ssh -J 192.168.1.50 172.16.50.2
```

継続的に使用したい場合は、`~/.ssh/config` に以下を加えることで、taget名でログインできるようになります。

```
Host target
    HostName target_host
    User target_user
    ProxyJump bastion_user@bastion_host
```


## クリーンナップ

`mactl del -f FILENAME` で、作成したオブジェクトを一括で削除できます。

```console
ubuntu@ws1:~/marmot-manifests/05-private-cluster-network-server$ mactl del -f all.yaml 
network "private" deletion requested (accepted) for 3 object(s)
server "server-50" deletion requested (accepted)
server "server-51" deletion requested (accepted)
server "server-52" deletion requested (accepted)
gateway "igw" deletion requested (accepted)
```

削除が完了するまでは、１〜２分を要しますが、全オブジェクトが削除されると、以下の様に表示されます。

```console
ubuntu@ws1:~/marmot-manifests/05-private-cluster-network-server$ mactl get -f all.yaml 
no network found with name "private"
no server found with name "server-50"
no server found with name "server-51"
no server found with name "server-52"
no gateway found with name "igw"
```

