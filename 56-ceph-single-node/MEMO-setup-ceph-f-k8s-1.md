# K8sのためのCephのセットアップ


## RBDブロックストレージ (Ceph側の設定)

---
## プロビジョナーとプラグインの設定

### ceph-conf.yaml
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ceph-config
data:
  ceph.conf: |
    [global]
    auth_cluster_required = cephx
    auth_service_required = cephx
    auth_client_required = cephx

  # keyring is a required key and its value should be empty
  keyring: |
```

```bash
kubectl apply -f ceph-conf.yaml
```

- clusterID
- monitors
- namespace


### csi-config-map.yaml
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ceph-csi-config
data:
  config.json: |-
    [
      {
        "clusterID": "8701d0fe-9943-11f1-969a-9b462dcf4d3e",
        "rbd": {
           "nodePublishSecretRef": {
             "name": "csi-rbd-secret",
             "namespace": "default"
           }
        },
        "monitors": [
          "192.168.1.173:6789"
        ]
      }
    ]
```

```bash
kubectl apply -f csi-config-map.yaml
```

## プラグインのデプロイ

```bash
cd ~/ceph-csi/examples/cephfs
plugin-deploy.sh

cd ~/ceph-csi/examples/rbd
plugin-deploy.sh
```


---

## Secretの設定 (K8s側の設定)
- userID
- userKey
- clusterID
- pool

上で取得した鍵を `csi-rbd-secret.yaml` に設定します。
`userID:`と`userKey:`に <ceph auth print-key client.k8s の出力> をセットする。

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: csi-rbd-secret
  namespace: default
stringData:
  userID: k8s
  userKey: AQD/x4JqMQWNCBAAmT4Ge6BdfqYEn3LvA1YAjw==
```

## StorageClassの設定 (K8s側の設定)
`rbd-storageclass.yaml`にセットして、デプロイする。
`ceph status`コマンドで出力される`id`を`clusterID`にセットする。
前述で指定した RBDのプール名を `pool:` にセットする。

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-rbd-sc
provisioner: rbd.csi.ceph.com
parameters:
  clusterID: 8701d0fe-9943-11f1-969a-9b462dcf4d3e
  pool: rbdpool
  imageFeatures: layering
  csi.storage.k8s.io/provisioner-secret-name: csi-rbd-secret
  csi.storage.k8s.io/provisioner-secret-namespace: default
  csi.storage.k8s.io/controller-expand-secret-name: csi-rbd-secret
  csi.storage.k8s.io/controller-expand-secret-namespace: default
  csi.storage.k8s.io/node-stage-secret-name: csi-rbd-secret
  csi.storage.k8s.io/node-stage-secret-namespace: default
  csi.storage.k8s.io/fstype: ext4

reclaimPolicy: Delete
allowVolumeExpansion: true
mountOptions:
  - discard
```


## K8sへの反映
```bash
kubectl apply -f csi-config-map.yaml
kubectl apply -f csi-rbd-secret.yaml
kubectl apply -f rbd-storageclass.yaml
kubectl get storageclass csi-rbd-sc
```

---

## Secretのセット (K8s側の設定)
- userID
- userKey

上で取得した鍵を `csi-cephfs-secret.yaml` に設定します。
`userKey:`は <上記 print-key の出力>をセットする。

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: csi-cephfs-secret
  namespace: default
stringData:
  # Required for statically and dynamically provisioned volumes
  # The userID must not include the "client." prefix!
  userID: csi-cephfs
  userKey: AQDtnYNqRo6NGRAApWtXOdUW5sK5p3/3djfoWQ==
  # Encryption passphrase
  encryptionPassphrase: test_passphrase
```

## Storageclassのセット (K8s側の設定)
- clusterID
- fsName


```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-cephfs-sc
provisioner: cephfs.csi.ceph.com
parameters:
  clusterID: 8701d0fe-9943-11f1-969a-9b462dcf4d3e
  fsName: cephfs

  csi.storage.k8s.io/provisioner-secret-name: csi-cephfs-secret
  csi.storage.k8s.io/provisioner-secret-namespace: default
  csi.storage.k8s.io/controller-expand-secret-name: csi-cephfs-secret
  csi.storage.k8s.io/controller-expand-secret-namespace: default
  csi.storage.k8s.io/controller-publish-secret-name: csi-cephfs-secret
  csi.storage.k8s.io/controller-publish-secret-namespace: default
  csi.storage.k8s.io/node-stage-secret-name: csi-cephfs-secret
  csi.storage.k8s.io/node-stage-secret-namespace: default

reclaimPolicy: Delete
allowVolumeExpansion: true
```

## K8sへの反映
```bash
kubectl apply -f csi-rbd-secret.yaml
kubectl apply -f rbd-storageclass.yaml
kubectl get storageclass csi-cephfs-sc
```

---

## K8s オブジェクト

```bash
$ kubectl get cm
NAME                             DATA   AGE
ceph-config                      2      15h
ceph-csi-config                  1      12h
ceph-csi-encryption-kms-config   1      15h
init-scripts                     1      15h
kube-root-ca.crt                 1      15h
```

```bash
$ kubectl get secret
NAME                TYPE     DATA   AGE
csi-cephfs-secret   Opaque   3      12h
csi-rbd-secret      Opaque   2      15h
```

```bash
$ kubectl get ds
NAME               DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
csi-cephfsplugin   3         3         3       3            3           <none>          12h
csi-rbdplugin      3         3         3       3            3           <none>          15h
```

```bash
$ kubectl get deploy
NAME                           READY   UP-TO-DATE   AVAILABLE   AGE
csi-cephfsplugin-provisioner   3/3     3            3           12h
csi-rbdplugin-provisioner      3/3     3            3           15h
vault                          1/1     1            1           15h
```

```bash
$ kubectl get svc
NAME                           TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
csi-cephfsplugin-provisioner   ClusterIP   10.105.137.162   <none>        8080/TCP   12h
csi-metrics-cephfsplugin       ClusterIP   10.97.202.226    <none>        8080/TCP   12h
csi-metrics-rbdplugin          ClusterIP   10.107.167.212   <none>        8080/TCP   15h
csi-rbdplugin-provisioner      ClusterIP   10.97.144.146    <none>        8080/TCP   15h
kubernetes                     ClusterIP   10.96.0.1        <none>        443/TCP    15h
vault                          ClusterIP   None             <none>        8200/TCP   15h
```

```bash
$ kubectl get sc
NAME            PROVISIONER           RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
csi-cephfs-sc   cephfs.csi.ceph.com   Delete          Immediate           true                   11h
csi-rbd-sc      rbd.csi.ceph.com      Delete          Immediate           true                   15h
```



---



