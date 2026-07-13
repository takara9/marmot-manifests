# ファイルストレージの作成から削除

## cephfs用のMDSデーモンを、クラスタ内に1個配置

mdsの配置
```console
FS_NAME=cephfs
ceph orch apply mds ${FS_NAME} --placement="1"
```

確認コマンド
```console
ceph orch ls mds
ceph orch ps --daemon-type mds
ceph mds stat
```

# ファイルシステム作成

cephfsの作成

```console
ceph fs volume create ${FS_NAME}
```

作成結果の確認

```console
ceph fs volume ls
ceph fs status ${FS_NAME}
```


## ceph-fs のユーザー client.vmuser の作成

```console
FS_USER_1=user1
ceph fs authorize ${FS_NAME} client.${FS_USER_1} / rw
ceph auth get client.${FS_USER_1} -o /etc/ceph/ceph.client.${FS_USER_1}.keyring
```

## client.vmuser ユーザーの情報取得

```console
ceph auth get client.${FS_USER_1}
```


## ユーザーの追加

FS_USER_2=user2
ceph fs authorize ${FS_NAME} client.${FS_USER_2} / rw
ceph auth get client.${FS_USER_2} -o /etc/ceph/ceph.client.${FS_USER_2}.keyring


## ユーザーを削除する方法

```cosnole
ceph auth del client.${FS_USER_1}
ceph auth del client.${FS_USER_2}
```


### ファイルストレージの削除

```cosnole
ceph fs set ${FS_NAME} down true
ceph config set mon mon_allow_pool_delete true
ceph fs flag set enable_multiple false --yes-i-really-mean-it
ceph fs volume rm ${FS_NAME} --yes-i-really-mean-it
```


## 確認

```cosnole
sudo ceph fs ls
sudo ceph fs volume ls
sudo ceph osd pool ls
sudo ceph -s
```


