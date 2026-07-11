# シングルノードのCeph

作成途上


```console
$ mactl create -f single-ceph.yaml 
```

Ceph Dashboard is now available at:

	     URL: https://ceph-single:8443/
	    User: admin
	Password: u5o1d2bkfk


sudo mount -t ceph 10.1.3.11:6789:/ /mnt/cephfs \
  -o name=vmuser,secret=AQAKtVBqpi9yDBAA5o6P4zIpOJmLvgLxxje8dg==