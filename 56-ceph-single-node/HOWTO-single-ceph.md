# シングル構成のCephサーバーの設定

Cephには、複数の安全機構があり、それらを解除しないとシングル構成のサーバーが作れません。

```console
sudo ceph osd crush rule ls
sudo ceph osd crush rule create-replicated single-node-rule default osd
sudo ceph config set global mon_allow_pool_size_one true
sudo ceph config get mon mon_allow_pool_size_one
sudo ceph config set global mon_warn_on_pool_no_redundancy false
sudo ceph config set global osd_pool_default_size 1
sudo ceph config set global osd_pool_default_min_size 1
```


```console
sudo ceph osd pool set .mgr size 1 --yes-i-really-mean-it
sudo ceph osd pool set .mgr min_size 1
```
