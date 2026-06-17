# 仮想マシンにdockerの実行環境をセットアップする

`mactl create -f server-110.yaml` を実行するだけで、仮想マシンが起動した後、
自動的にansibleで docker-ce をインストールして、仮想マシン上で Docker イメージが起動できるようになります。


```console
$ mactl create -f server-110.yaml 
リソースの作成要求が受け入れられました。ID: 7f418
OS起動待機中.....
playbook 適用開始.....

PLAY [setup servers] *****************************************************************************************************

TASK [Gathering Facts] ***************************************************************************************************
ok: [server-41]

＜中略＞

PLAY RECAP ***************************************************************************************************************
server-41                  : ok=9    changed=7    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   


$ ssh 192.168.1.110
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-106-generic x86_64)

 * Documentation:  https://help.ubuntu.com

＜中略＞

ubuntu@server-110:~$ docker run hello-world
Unable to find image 'hello-world:latest' locally
latest: Pulling from library/hello-world
4f55086f7dd0: Pull complete 
d5e71e642bf5: Download complete 
Digest: sha256:96498ffd522e70807ab6384a5c0485a79b9c7c08ca79ba08623edcad1054e62d
Status: Downloaded newer image for hello-world:latest

Hello from Docker!
This message shows that your installation appears to be working correctly.

To generate this message, Docker took the following steps:
 1. The Docker client contacted the Docker daemon.
 2. The Docker daemon pulled the "hello-world" image from the Docker Hub.
    (amd64)
 3. The Docker daemon created a new container from that image which runs the
    executable that produces the output you are currently reading.
 4. The Docker daemon streamed that output to the Docker client, which sent it
    to your terminal.

To try something more ambitious, you can run an Ubuntu container with:
 $ docker run -it ubuntu bash

Share images, automate workflows, and more with a free Docker ID:
 https://hub.docker.com/

For more examples and ideas, visit:
 https://docs.docker.com/get-started/
```