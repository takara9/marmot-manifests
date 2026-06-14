# iSCSI Volume

iSCSIでネットワーク上のボリュームを利用できる様にする。


```yaml
apiVersion: v1
kind: Volume
metadata:
    name: vol-iscsi-1
spec:
    type: lvm
    size: 2
    iscsi: true
```

## iSCSIボリュームを利用する仮想サーバー

これまでと同じように、名前を記述する

```yaml
    storage:
        - metadata:
            name: vol-iscsi-1
```

