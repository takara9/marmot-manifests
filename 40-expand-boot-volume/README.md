# ブートボリュームの拡大

外部ストレージを接続するのではなく、OSが起動するブートボリュームを拡大したい場合があります。
この機能は、spec.bootVolume.spec.type="qcow2"のみ対応しています。

```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-106
    comment: データストレージを実装するケース
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    # 以下 の設定で、ブートボリュームを拡大できます。
    bootVolume:
        spec:
            size: 50
            type: qcow2
    auth:
        url: https://github.com/takara9.keys
```

## OSボリューム拡大の確認

デフォルトは、16Gのところ、上記の設定により、50Gに拡大されていることが確認できます。

```console
ubuntu@server-106:~$ lsblk
NAME    MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
sr0      11:0    1  368K  0 rom  
vda     253:0    0   50G  0 disk 
├─vda1  253:1    0   49G  0 part /
├─vda14 253:14   0    4M  0 part 
├─vda15 253:15   0  106M  0 part /boot/efi
└─vda16 259:0    0  913M  0 part /boot
```

