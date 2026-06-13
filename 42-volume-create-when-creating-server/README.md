# サーバーのマニフェストにボリューム作成も含める

仮想サーバーをデプロイするマニフェストに、ボリューム生成を入れることもできます。


```console
$ mactl create -f server-42.yaml 
リソースの作成要求が受け入れられました。ID: d271c

$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-42        ws1           PROVISIONING  1    1048     192.168.1.42     host-bridge      4s

$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-42        ws1           RUNNING       1    1048     192.168.1.42     host-bridge      22s

$ mactl get vol
NAME          NODE        KIND  TYPE   iSCSI  SIZE(GB)  STATUS     PATH                  AGE
----          ----        ----  ----   -----  --------  ------     ----                  ---
var           ws1         data  qcow2  -      2         AVAILABLE  .../data-5556c.qcow2  23s
for-vg1       ws1         data  qcow2  -      1         AVAILABLE  .../data-28569.qcow2  23s
for-vg2       ws1         data  qcow2  -      1         AVAILABLE  .../data-275ce.qcow2  23s
```

サーバーを削除する。 `spec.pesistent = true` がセットされたボリュームだけ、消されずに残る。

```
$ mactl del -f server-42.yaml 
server "server-42" deletion requested (accepted)
```

`spec.pesistent = true` がセットされた`for-vg2` だけが残存した。

```
ubuntu@ws1:~/marmot-manifests/42-volume-create-when-creating-server$ mactl get vol
NAME          NODE        KIND  TYPE   iSCSI  SIZE(GB)  STATUS     PATH                  AGE
----          ----        ----  ----   -----  --------  ------     ----                  ---
for-vg2       ws1         data  qcow2  -      1         AVAILABLE  .../data-275ce.qcow2  5m
```


`for-vg2`を削除する
```
$ mactl del vol for-vg2
volume "for-vg2" deletion requested (accepted)

$ mactl get vol
NAME          NODE        KIND  TYPE   iSCSI  SIZE(GB)  STATUS     PATH                  AGE
----          ----        ----  ----   -----  --------  ------     ----                  ---
```
