"failed to load cni during init, please check CRI plugin status before setting up network for pods"

error="cni config load failed: no network config found in /etc/cni/net.d: cni plugin not initialized: failed to load cni config"


Nov 20 23:19:22 master1 containerd[3969]: time="2024-11-20T23:19:22.256454066Z" level=error msg="failed to load cni during init, please check CRI plugin status before setting up network for pods" error="cni config load failed: no network config found in /etc/cni/net.d: cni plugin not initialized: failed to load cni config"

Nov 20 23:19:42 master1 containerd[3969]: time="2024-11-20T23:19:42.358735904Z" level=error msg="failed to reload cni configuration after receiving fs change event(RENAME        \"/etc/cni/net.d/.kubernetes-cni-keep.dpkg-new\")" error="cni config load failed: no network config found in /etc/cni/net.d: cni plugin not initialized: failed to load cni config"


root@master1:/home/ubuntu# cd /etc/cni
root@master1:/etc/cni# ls -la
total 12
drwxr-xr-x   3 root root 4096 Nov 20 23:29 .
drwxr-xr-x 105 root root 4096 Nov 20 23:29 ..
drwxr-xr-x   2 root root 4096 Nov 20 23:31 net.d
root@master1:/etc/cni# cd net.d/
root@master1:/etc/cni/net.d# ls -la
total 12
drwxr-xr-x 2 root root 4096 Nov 20 23:31 .
drwxr-xr-x 3 root root 4096 Nov 20 23:29 ..
-rw-r--r-- 1 root root    0 Sep 11  2023 .kubernetes-cni-keep
-rw-r--r-- 1 root root  191 Nov 20 23:31 05-cilium.conflist
root@master1:/etc/cni/net.d# cat 05-cilium.conflist 

{
  "cniVersion": "0.3.1",
  "name": "cilium",
  "plugins": [
    {
       "type": "cilium-cni",
       "enable-debug": false,
       "log-file": "/var/run/cilium/cilium-cni.log"
    }
  ]
}


level=error msg="RunPodSandbox for &PodSandboxMetadata{Name:etcd-master1,Uid:c7faf99ca0ded98e83ffa369366ce2e0,Namespace:kube-system,Attempt:0,} failed, error" error="failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown"


error="failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown"


"RunPodSandbox from runtime service failed" err="rpc error: code = Unknown desc = failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown"



Nov 21 07:25:50 master1 containerd[4006]: time="2024-11-21T07:25:50.590372815Z" level=error msg="RunPodSandbox for &PodSandboxMetadata{Name:etcd-master1,Uid:153c6847d61a86788cd0d5288d47f732,Namespace:kube-system,Attempt:0,} failed, error" error="failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown"

Nov 21 07:25:50 master1 kubelet[5721]: E1121 07:25:50.590524    5721 log.go:32] "RunPodSandbox from runtime service failed" err="rpc error: code = Unknown desc = failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown"

Nov 21 07:25:50 master1 kubelet[5721]: E1121 07:25:50.590569    5721 kuberuntime_sandbox.go:72] "Failed to create sandbox for pod" err="rpc error: code = Unknown desc = failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown" pod="kube-system/etcd-master1"

Nov 21 07:25:50 master1 kubelet[5721]: E1121 07:25:50.590580    5721 kuberuntime_manager.go:1170] "CreatePodSandbox for pod failed" err="rpc error: code = Unknown desc = failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown" pod="kube-system/etcd-master1"

Nov 21 07:25:50 master1 kubelet[5721]: E1121 07:25:50.590607    5721 pod_workers.go:1301] "Error syncing pod, skipping" err="failed to \"CreatePodSandbox\" for \"etcd-master1_kube-system(153c6847d61a86788cd0d5288d47f732)\" with CreatePodSandboxError: \"Failed to create sandbox for pod \\\"etcd-master1_kube-system(153c6847d61a86788cd0d5288d47f732)\\\": rpc error: code = Unknown desc = failed to create containerd task: cgroups: cgroup mountpoint does not exist: unknown\"" pod="kube-system/etcd-master1" podUID="153c6847d61a86788cd0d5288d47f732"