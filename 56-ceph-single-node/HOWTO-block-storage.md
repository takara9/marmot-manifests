# ブロックデイバイスの作成から削除まで
プールは、VM用専用に作成して、それぞれアクセス権を付与する想定


## 物理ディスクの情報
Disk /dev/sda: 931.51 GiB, 1000204886016 bytes, 1953525168 sectors  boot
Disk model: SanDisk SDSSDH3 

Disk /dev/sdb: 931.51 GiB, 1000204886016 bytes, 1953525168 sectors
Disk model: WDC WD10EZRX-00A

Disk /dev/sdc: 931.51 GiB, 1000204886016 bytes, 1953525168 sectors
Disk model: SanDisk SDSSDH3 

Disk /dev/nvme0n1: 931.51 GiB, 1000204886016 bytes, 1953525168 sectors
Disk model: CSSD-M2B1TPG3VNF                        


```console
ubuntu@hv4:~$ df -h -x tmpfs -x efivarfs
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda2       915G   15G  854G   2% /
/dev/sda1       1.1G  6.2M  1.1G   1% /boot/efi
```

```console
ubuntu@hv4:~$ sudo pvs
  PV           VG  Fmt  Attr PSize   PFree  
  /dev/nvme0n1 vg2 lvm2 a--  931.51g 931.51g 高速  vg-nvme
  /dev/sdc     vg1 lvm2 a--  931.51g 915.51g 中速  vg-sdd
  /dev/sdb     vg3 lvm2 a--  931.51g 931.51g 低速  vg-hdd
```

```console
ubuntu@hv4:~$ sudo vgs
  VG  #PV #LV #SN Attr   VSize   VFree  
  vg1   1   1   0 wz--n- 931.51g 915.51g
  vg2   1   0   0 wz--n- 931.51g 931.51g
  vg3   1   0   0 wz--n- 931.51g 931.51g
```

ボリュームグループでディスクのクラス別けも可能かもしれない。
vg3  HDD     低速Blockストレージ  vg-hdd
vg2  NVMe　　高速Blockストレージ  vg-nvme
vg1  SSD　　 中速Blockストレージ  vg-sdd
 
※しかし、 Cephをホストにインストールする場合は、直接デバイスを見せた方が良いと思われる。


## デバイスのリスト

```console
ceph orch device ls
```

## OSDのリスト

デバイスのリスト表示

```console
ceph osd df tree
```

クラスを独自設定

```console
ceph osd crush rm-device-class osd.0
ceph osd crush set-device-class hdd osd.0
ceph osd crush rm-device-class osd.1
ceph osd crush set-device-class ssd osd.1
ceph osd crush rm-device-class osd.2
ceph osd crush set-device-class nvme osd.2
```

３種類の速度の異なるストレージが実装されている場合

```console
root@ceph-single:/home/ubuntu# ceph osd tree
ID  CLASS  WEIGHT   TYPE NAME             STATUS  REWEIGHT  PRI-AFF
-1         0.29306  root default                                   
-3         0.29306      host ceph-single                           
 0    hdd  0.09769          osd.0             up   1.00000  1.00000
 2   nvme  0.09769          osd.2             up   1.00000  1.00000
 1    ssd  0.09769          osd.1             up   1.00000  1.00000
```


## デバイスクラスごとのCRUSHルールを作成

OSDのクラスを使用するルールを作成

```console
ceph osd crush rule create-replicated rule-hdd default osd hdd
ceph osd crush rule create-replicated rule-ssd default osd ssd
ceph osd crush rule create-replicated rule-nvme default osd nvme
```

ルールをリストする

```console
ceph osd crush rule ls
replicated_rule
single-node-rule
rule-hdd
rule-ssd
rule-nvme
```



## プールの作成とブロックデバイスの作成

SSDのドライブクラスを指定して、プールを作成する

```console
POOL_NAME=vm-pool-1
DEV_CLASS=rule-ssd
ceph osd pool create ${POOL_NAME} 16 16 ${DEV_CLASS}
ceph osd pool ls

rbd pool init ${POOL_NAME}

rbd create ${POOL_NAME}/disk-1 --size 2G
rbd ls ${POOL_NAME}

rbd create ${POOL_NAME}/disk-2 --size 3G
rbd ls ${POOL_NAME}
```


## ユーザーの作成とアクセス件の付与

権限の存在チェック
```console
CEPH_USER=user-1
ceph auth get client.${CEPH_USER}
```

ユーザーと権限のセット
```console
ceph auth get-or-create client.${CEPH_USER} \
    mon "profile rbd" \
    osd "profile rbd pool=${POOL_NAME}" \
  -o /etc/ceph/ceph.client.${CEPH_USER}.keyring
```

登録内容の確認
```console
ceph auth ls | grep -A 3 client.${CEPH_USER}
```

キーの表示
```console
ceph auth print-key client.${CEPH_USER}
```

## Cephクライアントからのアクセス・テストを実施する

手順はXX参照のこと


## ユーザーの削除

削除の実行
```console
ceph auth del client.${CEPH_USER}
```

実行後の確認
```console
ceph auth ls | grep client.${CEPH_USER}
```


## ブロックデバイスの削除

ブロックデバイスのリスト
```console
rbd ls ${POOL_NAME} 
```

ステータスのチェック
```console
rbd status ${POOL_NAME}/disk-1
rbd status ${POOL_NAME}/disk-2
```

スナップショットのリスト
```console
rbd snap ls ${POOL_NAME}/disk-1
rbd snap ls ${POOL_NAME}/disk-2
```

スナップショットの削除と本体の削除
```console
rbd snap purge ${POOL_NAME}/disk-1
rbd rm ${POOL_NAME}/disk-1
rbd snap purge ${POOL_NAME}/disk-2
rbd rm ${POOL_NAME}/disk-2
```


## プールの削除

プールの削除を許可
```console
ceph config get mon mon_allow_pool_delete
ceph config set global mon_allow_pool_delete true
```

プールを指定して削除
```console
ceph osd pool delete ${POOL_NAME} ${POOL_NAME} --yes-i-really-really-mean-it
ceph osd pool ls
```


