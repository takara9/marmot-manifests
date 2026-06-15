# Kubernetesクラスタの構築

```console
$ mactl create -f cluster-servers.yaml 
リソースの作成要求が受け入れられました。ID: 8a9eb
リソースの作成要求が受け入れられました。ID: a1293
リソースの作成要求が受け入れられました。ID: 0cba0
```


```console
ubuntu@hv0:~/marmot-manifests/52-ansible-kubernetes$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
master           hv0           RUNNING       4    8192     172.16.20.220    cluster-net      51s
                                                           192.168.1.220    host-bridge      
node1            hv0           RUNNING       2    4192     172.16.20.221    cluster-net      51s
                                                           192.168.1.221    host-bridge      
node2            hv0           RUNNING       2    4192     172.16.20.222    cluster-net      51s
                                                           192.168.1.222    host-bridge     
```

```console
$ ansible -i hosts -m ping all
master | SUCCESS => {
    "ansible_facts": {
        "discovered_interpreter_python": "/usr/bin/python3"
    },
    "changed": false,
    "ping": "pong"
}
node1 | SUCCESS => {
    "ansible_facts": {
        "discovered_interpreter_python": "/usr/bin/python3"
    },
    "changed": false,
    "ping": "pong"
}
node2 | SUCCESS => {
    "ansible_facts": {
        "discovered_interpreter_python": "/usr/bin/python3"
    },
    "changed": false,
    "ping": "pong"
}
```

```console
$ ansible-playbook -i hosts playbooks/install-node.yaml 
```

