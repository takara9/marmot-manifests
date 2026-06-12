# 永続ボリューム



ubuntu@ws1:~/marmot-manifests/41-volume-local$ mactl create -f volume-1.yaml 
リソースの作成要求が受け入れられました。ID: <nil>
ubuntu@ws1:~/marmot-manifests/41-volume-local$ mactl get vol
NAME          NODE  KIND  TYPE   iSCSI  SIZE(GB)  STATUS     PATH                  AGE
----          ----  ----  ----   -----  --------  ------     ----                  ---
boot-b67cd    marmot3  os    qcow2  -      16        AVAILABLE  .../boot-71da1.qcow2  1d
boot-b9d66    marmot1  os    qcow2  -      16        AVAILABLE  .../boot-439c1.qcow2  1d
boot-56b1c    marmot2  os    qcow2  -      16        AVAILABLE  .../boot-eaf24.qcow2  1d
boot-d1024    marmot1  os    qcow2  -      16        AVAILABLE  .../boot-05e05.qcow2  4h
boot-2ef8f    marmot2  os    qcow2  -      16        AVAILABLE  .../boot-46a19.qcow2  4h
boot-f7992    marmot3  os    qcow2  -      16        AVAILABLE  .../boot-9a7ea.qcow2  3h
volume1       marmot1  data  qcow2  -      2         AVAILABLE  .../data-2e659.qcow2  4s


ubuntu@ws1:~/marmot-manifests/41-volume-local$ mactl create -f server-108.yaml 
リソースの作成要求が受け入れられました。ID: 7a6e1
ubuntu@ws1:~/marmot-manifests/41-volume-local$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-108       marmot1       PROVISIONING  2    2048     192.168.1.108    host-bridge      3s


