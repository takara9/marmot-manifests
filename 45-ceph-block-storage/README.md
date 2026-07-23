# LVMのボリュームをアタッチする仮想サーバー



```yaml
apiVersion: v1
kind: Volume
metadata:
    name: lvm-vol1
spec:
    type: lvm
    kind: data
    size: 5
    persistent: true
```



```console
$ mactl create -f server-43.yaml 
リソースの作成要求が受け入れられました。ID: aface
リソースの作成要求が受け入れられました。ID: 02f42

$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-43        ws1           RUNNING       1    1048     192.168.1.43     host-bridge      8s

$ mactl get vol
NAME          NODE        KIND  TYPE   iSCSI  SIZE(GB)  STATUS     PATH                  AGE
----          ----        ----  ----   -----  --------  ------     ----                  ---
lvm-vol1      ws1         data  lvm    -      5         AVAILABLE  /dev/vg2/datalv-aface  12s
```



```
$ mactl get srv -o json |jq .[].spec.storage.[].spec
{
  "kind": "data",
  "logicalVolume": "datalv-aface",
  "path": "/dev/vg2/datalv-aface",
  "persistent": true,
  "size": 5,
  "type": "lvm",
  "volumeGroup": "vg2"
}
```



```console
$ mactl del -f server-43.yaml 
volume "lvm-vol1" deletion requested (accepted)
server "server-43" deletion requested (accepted)
```