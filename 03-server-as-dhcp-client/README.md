# DHCPクライアントとしての仮想サーバー

MarmotのIPアドレス管理機能を使わず、LANのDHCPサーバーからIPアドレスを割当ることができます。


```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-dhcp4
    comment: DHCPクライアントとして動作する仮想サーバー
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth:
        url: https://github.com/takara9.keys
    networkInterface:
        - networkname: host-bridge
          # 以下の設定を入れると、DHCPサーバーからアドレスを取得する
          dhcp4: true 
          dhcp6: false
```


```yaml
    networkInterface:
        - networkname: host-bridge
          # 以下の設定を入れると、DHCPサーバーからアドレスを取得する
          dhcp4: false
          dhcp6: true
```

marmot から IPアドレスを割り当てないので、表示は `N/A`となります。

```console
$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-dhcp4     marmot1       RUNNING       1    1024     N/A              N/A              4m
```

Linux カーネルのIPv4 や IPv6 を無効化していないので、表示されますが、ご了承ください。
仮想サーバーに `mactl console server-dhcp4` でログインして、IPアドレスが解れば、ssh でログインすることができます。

