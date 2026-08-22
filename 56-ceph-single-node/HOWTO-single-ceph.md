# シングル構成のCephサーバーの設定

Cephには、複数の安全機構があり、それらを解除しないとシングル構成のサーバーが作れません。

```bash
sudo ceph osd crush rule ls
sudo ceph osd crush rule create-replicated single-node-rule default osd
sudo ceph config set global mon_allow_pool_size_one true
sudo ceph config get mon mon_allow_pool_size_one
sudo ceph config set global mon_warn_on_pool_no_redundancy false
sudo ceph config set global osd_pool_default_size 1
sudo ceph config set global osd_pool_default_min_size 1
```


```bash
sudo ceph osd pool set .mgr size 1 --yes-i-really-mean-it
sudo ceph osd pool set .mgr min_size 1
```


ストレージプールの削除
```bash
ceph tell mon.* injectargs '--mon_allow_pool_delete=true'
ceph osd pool rm vm-pool-1 vm-pool-1 --yes-i-really-really-mean-it
ceph tell mon.* injectargs '--mon_allow_pool_delete=false'
```



```bash
# 現在の fs 一覧を確認
ceph fs ls
ceph fs status myfs

# fs を止める(down にする) - 新しいcephではvolume rmで自動的に処理されることが多い
ceph fs fail myfs

# fs 削除を許可してから、fs 自体を削除
ceph config set mon mon_allow_pool_delete true
ceph fs volume rm myfs --yes-i-really-really-mean-it
```



```bash
ceph fs fail myfs
ceph fs rm myfs --yes-i-really-really-mean-it

ceph tell mon.* injectargs '--mon_allow_pool_delete=true'
# その後にプールを削除
ceph osd pool rm cephfs.myfs.meta cephfs.myfs.meta --yes-i-really-really-mean-it
ceph osd pool rm cephfs.myfs.data cephfs.myfs.data --yes-i-really-really-mean-it
ceph tell mon.* injectargs '--mon_allow_pool_delete=false'

ceph config set mon mon_allow_pool_delete false
```

