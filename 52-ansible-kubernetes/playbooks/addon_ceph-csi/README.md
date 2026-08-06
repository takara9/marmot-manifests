# Ceph CSI (RBD / CephFS) 接続用アドオン

Kubernetesクラスタから、Cephクラスタ（例: [56-ceph-single-node](../../../56-ceph-single-node/README.md)）の
ブロックストレージ(RBD)とファイルストレージ(CephFS)を、CSI経由の永続ボリュームとして利用できるようにします。

構成する CSI ドライバは以下の2つです。

- `ceph-csi-rbd`  : RBDイメージをブロックデバイスとしてPodにアタッチ（RWO）
- `ceph-csi-cephfs` : CephFSをファイル共有としてPodにマウント（RWX可）

いずれも Ceph公式の Helm chart (`https://ceph.github.io/csi-charts`) をインストールします。


## 事前準備: Cephクラスタ側の作業

### 1. Cephクラスタ識別子とMONアドレスを取得

Cephノード（例: ceph-single）で以下を実行します。

```console
ceph fsid
ceph mon dump | grep addr
```

得られた fsid を `ceph_cluster_id` に、MONの `IPアドレス:6789` を `ceph_monitors` に設定します。

### 2. RBD用のプールとユーザーを作成

手順の詳細は [HOWTO-block-storage.md](../../../56-ceph-single-node/HOWTO-block-storage.md) を参照してください。

```console
POOL_NAME=k8s-rbd-pool
ceph osd pool create ${POOL_NAME} 16 16
rbd pool init ${POOL_NAME}

CEPH_USER=k8s
ceph auth get-or-create client.${CEPH_USER} \
    mon "profile rbd" \
    osd "profile rbd pool=${POOL_NAME}" \
  -o /etc/ceph/ceph.client.${CEPH_USER}.keyring

ceph auth print-key client.${CEPH_USER}
```

### 3. CephFS用のファイルシステムとユーザーを作成

手順の詳細は [HOWTO-file-storage.md](../../../56-ceph-single-node/HOWTO-file-storage.md) を参照してください。

```console
FS_NAME=cephfs
ceph orch apply mds ${FS_NAME} --placement="1"
ceph fs volume create ${FS_NAME}

FS_USER=k8s
ceph fs authorize ${FS_NAME} client.${FS_USER} / rw
ceph auth print-key client.${FS_USER}
```


## Kubernetes側の設定

`52-ansible-kubernetes/hosts` の `[all:vars]` に、上記で取得した値を追記します。

```ini
ceph_cluster_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
ceph_monitors        = ["192.168.1.230:6789"]

ceph_rbd_pool        = "k8s-rbd-pool"
ceph_rbd_user_id     = "k8s"
ceph_rbd_user_key    = "AQxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx=="

ceph_fs_name         = "cephfs"
ceph_fs_user_id      = "k8s"
ceph_fs_user_key     = "AQxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx=="
```

`playbooks/addon.yaml` には `addon_ceph-csi` ロールが追加済みなので、通常のアドオン適用と同じ手順で実行します。

```console
$ ansible-playbook -i hosts playbooks/addon.yaml
```


## 動作確認

```console
$ kubectl get pods -n ceph-csi
$ kubectl get storageclass
NAME                       PROVISIONER          RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
csi-rbd-sc                 rbd.csi.ceph.com     Delete          Immediate           true                   1m
csi-cephfs-sc              cephfs.csi.ceph.com  Delete          Immediate           true                   1m
```

PVCの動作確認例（ブロックストレージ）

```console
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: rbd-pvc-test
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: csi-rbd-sc
  resources:
    requests:
      storage: 1Gi
EOF

kubectl get pvc rbd-pvc-test
```

PVCの動作確認例（ファイルストレージ、複数Podから同時マウント可）

```console
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: cephfs-pvc-test
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: csi-cephfs-sc
  resources:
    requests:
      storage: 1Gi
EOF

kubectl get pvc cephfs-pvc-test
```


# メモ
- https://github.com/takara9/k8s-manifests/tree/main/ceph-csi

