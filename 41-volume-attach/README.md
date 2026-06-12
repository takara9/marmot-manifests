# 永続ボリューム

marmot には、２種類のディスク（ブロックストレージ）を利用できます。
- 仮想サーバーのホストが提供するボリューム（ローカル・ブロックストレージ）
- ネットワーク上の共有ボリューム（リモート・ブロックストレージ）

これら２つの最大の違いは、パフォーマンスです。
ネットワーク経由でアクセスするリモート・ブロックストレージを利用すると、
如何にネットワークが高速でも、ローカル・ブロックストレージと比べると、速度劣化は避けられません。
しかし、リモート・ブロックストレージは、専用装置化して、可用性を高めることができます。

marmotでは、リモート・ブロックストレージとローカル・ブロックストレージは、どちらも
仮想マシンをホストするサーバー上で動かしていますから、現在の marmot では、リモート・ブロックストレージを
利用する利点はあまり無いと思います。


## 最小限の Volume 作成用のマニフェスト

以下のYAMLで、ボリュームを作成できます。
`spec.persistent = true` を設定しないと、仮想サーバーを削除すると、一緒に削除されるので注意してください。

```yaml
apiVersion: v1
kind: Volume
metadata:
    name: volume1
spec:
    size: 2   # GB
    persitent: true # サーバー削除時に、一緒に削除されることを防止するため
```

仮想サーバーから、作成したボリュームをアタッチするマニフェストです。

```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-108
    comment: 事前定義済のボリュームを名前で指定する例
spec:
    cpu: 2
    memory: 2048
    osVariant: ubuntu24.04
    storage:
        # 事前定義済のボリュームを名前で指定する例
        - metadata:
            name: volume1
```

これにより、仮想サーバーの起動時に ボリュームを接続して起動します。
ストレージは、ファイルシステムが作成されていない状態です。

OSからマウントして利用するには、以下の作業が必要となります。
- mkfs.ext4 などでファイルシステムを作成
- blkgid で ボリュームのIDを取得
- /etc/fstab を編集して、OS起動時にマウントするよう設定


## ボリュームの作成

volume3 は、まだ存在しない marmotクラスタのノードにアサインしようとしたケースです。
リトライせずにエラーになります。

```console
$ mactl create -f volume-0.yaml 
リソースの作成要求が受け入れられました。ID: ce442
$ mactl create -f volume-1.yaml 
リソースの作成要求が受け入れられました。ID: 30e58
$ mactl create -f volume-2.yaml 
リソースの作成要求が受け入れられました。ID: 2d4d2
$ mactl create -f volume-3.yaml 
リソースの作成要求が受け入れられました。ID: 34ade
$ mactl get vol
NAME          NODE        KIND  TYPE   iSCSI  SIZE(GB)  STATUS     PATH                  AGE
----          ----        ----  ----   -----  --------  ------     ----                  ---
volume0       marmot1     data  qcow2  -      2         AVAILABLE  .../data-ce442.qcow2  10s
volume1       marmot1     data  qcow2  -      2         AVAILABLE  .../data-30e58.qcow2  7s
volume2       marmot2     data  qcow2  -      2         AVAILABLE  .../data-2d4d2.qcow2  4s
volume3       marmot5     data  qcow2  -      2         PENDING    .../data-34ade.qcow2  2s
```


## ボリュームをアタッチして起動の例

```
$ mactl create -f server-108.yaml 
リソースの作成要求が受け入れられました。ID: 8c0e0

$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-108       marmot1       RUNNING       2    2048     192.168.1.108    host-bridge      13s

$ ssh 192.168.1.108
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-117-generic x86_64)
<中略>

ubuntu@server-108:~$ sudo lsblk
NAME    MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
sr0      11:0    1  368K  0 rom  
vda     253:0    0   16G  0 disk 
├─vda1  253:1    0   15G  0 part /
├─vda14 253:14   0    4M  0 part 
├─vda15 253:15   0  106M  0 part /boot/efi
└─vda16 259:0    0  913M  0 part /boot
vdb     253:16   0    2G  0 disk 
```

## スケジューリングについて

Volumeリソースは、現在、オブジェクトの作成について自動でスケジュールしません。
その代わりに、metadata.nodeName に記述したノードに、ボリュームを配置します。

そして、それをアタッチする仮想マシンも、ボリュームが存在するノードにスケジュールされます。
