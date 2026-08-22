# K8sのためのCephのセットアップ


## RBDブロックストレージ (Ceph側の設定)

RBDプールの作成
```bash
ceph osd pool create rbdpool 16 16 single-node-rule
ceph osd pool application enable rbdpool rbd
rbd pool init rbdpool

# シングルノードなのでプールサイズを1に設定
ceph osd pool set rbdpool size 1 --yes-i-really-mean-it
ceph osd pool set rbdpool min_size 1
```

Ceph設定の確認コマンド

```bash
# プール一覧を確認
ceph osd pool ls

# 詳細情報（サイズ、min_size、crush_ruleなど）も見る場合
ceph osd pool ls detail

# rbdpool の中身（RBDイメージ一覧）を見たい場合は次のようにプールを指定
rbd ls -p rbdpool
```

RBDブロックストレージ専用ユーザーの作成 (Ceph側の設定)

```bash
RBD_USER=rbduser1
ceph auth get-or-create client.${RBD_USER} \
    mon "profile rbd" \
    osd "profile rbd pool=rbdpool" \
  -o /etc/ceph/ceph.client.${RBD_USER}.keyring

# 鍵だけを表示
ceph auth print-key client.${RBD_USER}
```

確認
```bash
ceph auth ls | grep -A3 client.${RBD_USER}
```

---

## Ceph File systems を作成 (Ceph側の設定)

Ceph FS の作成
```bash
FS_NAME=cephfs
ceph orch apply mds ${FS_NAME} --placement="1"
ceph fs volume create ${FS_NAME}
ceph fs subvolumegroup create ${FS_NAME} csi
```

確認
```bash
ceph orch ps --daemon-type mds
ceph fs volume ls
ceph fs status ${FS_NAME}
ceph fs subvolumegroup ls ${FS_NAME}
```


CSI専用ユーザーを作成 (Ceph側の設定)
```bash
FS_USER=fsuser
SUB_VOL=csi
ceph auth get-or-create client.${FS_USER} \
  mgr "allow rw" \
  osd "allow rwx tag cephfs metadata=${FS_NAME}, allow rw tag cephfs data=${FS_NAME}" \
  mds "allow r fsname=${FS_NAME} path=/volumes, allow rws fsname=${FS_NAME} path=/volumes/${SUB_VOL}" \
  mon "allow r fsname=${FS_NAME}"
```

鍵だけを表示ex
```bash
ceph auth print-key client.${FS_USER}
```


確認
```bash
ceph auth ls | grep -A3 client.${FS_USER}
```
