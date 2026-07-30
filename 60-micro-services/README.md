# マイクロサービスアーキテクチャのアプリケーション

## ネットワークの作成

$ mactl create -f private-nets.yaml 
リソースの作成要求が受け入れられました。ID: ae9f7
リソースの作成要求が受け入れられました。ID: 2e44d
リソースの作成要求が受け入れられました。ID: fefdb

$ mactl get net
NAME            NODE       BRIDGE        STATUS        AGE       IP-NET        
----            ---------  -----------   ----------    ---       --------------
host-bridge     hv1        br0           ACTIVE        6d        -             
default         hv1        virbr0        ACTIVE        6d        -             
ovs-network     hv1        ovsbr0        ACTIVE        6d        -             
db-net          hv1        br-ae9f7      ACTIVE        1m        172.16.90.0/24
rest-net        hv1        br-2e44d      ACTIVE        1m        172.16.100.0/24
biz-net         hv1        br-fefdb      ACTIVE        1m        172.16.120.0/24



## データベースの作成

$ mactl create -f db-server.yaml 
リソースの作成要求が受け入れられました。ID: 418bb
リソースの作成要求が受け入れられました。ID: 1d14f

$ mactl get -f db-server.yaml 
NAME                        NODE        KIND  TYPE   iSCSI  SIZE(GB)  STATUS     PATH                  AGE
----                        ----        ----  ----   -----  --------  ------     ----                  ---
data-1                      hv1         data  lvm    -      10        AVAILABLE  /dev/vg2/datalv-418bb  2m

NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
mysql-64         hv1           RUNNING       4    4096     10.1.1.10        host-bridge      2m
                                                           172.16.90.2      db-net    


## RESTサービス１のデプロイ

$ mactl create -f rest-svc-1.yaml 
リソースの作成要求が受け入れられました。ID: d6790

$ mactl get -f rest-svc-1.yaml 
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
rest-svc-1       hv1           RUNNING       1    1024     10.1.1.11        host-bridge      1m
                                                           172.16.90.3      db-net           
                                                           172.16.100.2     rest-net  


## RESTサービス２のデプロイ

$ mactl create -f rest-svc-2.yaml 
リソースの作成要求が受け入れられました。ID: ce4f8

$ mactl get -f rest-svc-2.yaml 
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
rest-svc-2       hv1           RUNNING       1    1024     10.1.1.12        host-bridge      1m
                                                           172.16.90.4      db-net           
                                                           172.16.100.3     rest-net    


## RESTサービス３のデプロイ

$ mactl create -f rest-svc-3.yaml 
リソースの作成要求が受け入れられました。ID: 1f01c

$ mactl get -f rest-svc-3.yaml 
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
rest-svc-3       hv1           RUNNING       1    1024     10.1.1.13        host-bridge      2m


## 業務サーバー

$ mactl create -f biz-server.yaml 
リソースの作成要求が受け入れられました。ID: f353e

