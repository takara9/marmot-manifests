# ブロックデイバイスの作成から削除まで

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

ドライブクラスを指定して、プールを作成する

```bash
POOL_NAME=marmot-ssd
DEV_CLASS=rule-ssd
ceph osd pool create ${POOL_NAME} 16 16 ${DEV_CLASS}

POOL_NAME=marmot-hdd
DEV_CLASS=rule-hdd
ceph osd pool create ${POOL_NAME} 16 16 ${DEV_CLASS}

POOL_NAME=marmot-nvme
DEV_CLASS=rule-nvme
ceph osd pool create ${POOL_NAME} 16 16 ${DEV_CLASS}

ceph osd pool ls
rbd pool init marmot-nvme
rbd pool init marmot-hdd
rbd pool init marmot-ssd
```



## アクセスユーザーの作成とアクセス件の付与

権限の存在チェック
```console
CEPH_USER=marmot-user-1
ceph auth get client.${CEPH_USER}
```

POOL_NAMEは、marmot-nvme, marmot-hdd, marmot-ssd について設定する

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

注意点: CephFSのpool名は cephfs.<fs_name>.meta / cephfs.<fs_name>.data という命名規則を前提にしています(Ceph Quincy以降のceph fs volume createのデフォルト命名)。実際のクラスタでプール名が異なる場合はceph fs lsやceph osd pool lsで確認し、必要ならタスク内のプール名を調整してください。