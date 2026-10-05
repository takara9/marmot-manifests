## OSイメージのロード

現在、marmot が対応するOSイメージは、Apline 3.23, Ubuntu 22.04, Ubuntu 24.04 です。

marmot を起動すると、自動的に Ubuntu 22.04/24.04 をロードして、利用可能な状態にします。
追加で、Apline 3.23 をロードすることができます。

起動した後、自動的にイメージを作成します。

```console
$ mactl get img
NAME            STATUS     SYNCED    LV     QCOW2  AGE
----            ------     ------    ------  -----  ---
ubuntu22.04     AVAILABLE  COMPLETE  mixed   yes    22m
ubuntu24.04     AVAILABLE  COMPLETE  mixed   yes    12m
```

Alpine Linux のOSイメージのロード

```console
$ mactl create -f image-alpine3.23.yaml 
リソースの作成要求が受け入れられました。ID: f9b54
```

```console
$ mactl get img alpine3
NAME            STATUS     SYNCED    LV     QCOW2  AGE
----            ------     ------    ------  -----  ---
alpine3         AVAILABLE  COMPLETE  mixed   yes    36s
```

## Alpine 3 Linux の仮想サーバー起動

Alpine Linuxは、ED25519 鍵が使えません。そのため、RSA 鍵を作成して利用します。

```
$ ssh-keygen -t rsa
```
生成された公開鍵を Alpine Linuxを起動するYAMLにセットする。

Alpine Linux のデフォルトユーザーは、`alpine` です。そして、rootのデフォルトパスワードも、`alpine`になっています。

```console
$ mactl create -f server-alpine.yaml 
リソースの作成要求が受け入れられました。ID: d72d4

$ mactl get -f server-alpine.yaml 
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
srv-alpine       marmot2       RUNNING       1    1024     192.168.1.51     host-bridge      21s
```

Alpine Linux のでデフォルトユーザーは、`alpine` なので、サーバー名の前に `alpine@` をセットします。

```console
$ mactl ssh alpine@srv-alpine
Welcome to Alpine!

```

## Rocky 9 Linux の起動

Rocky Linux は、起動に少し時間がかかるので、`mactl console SERVER-NAME` コマンドで、コンソールを見ると、
起動の過程を表示することができるので、`mactl ssh rocky@SERVER-NAME` を実行するタイミングが分かると思います。

Rocky Linux のデフォルトユーザーは、`rocky` です。そして、rootのデフォルトパスワードも、`rocky`になっています。

```console
$ mactl create -f server-rocky9.yaml
```

起動過程を見るには、以下のコマンドです。コンソールから抜けるには、`Ctrl+]` を実行します。
```console
$ mactl console srv-rocky9
```

ログインするには
```console
$ mactl ssh rocky@srv-rocky9
Last login: Mon Oct  5 21:49:03 2026 from 192.168.1.8
[rocky@srv-rocky9 ~]$ 
```

## Ubuntu 22.04/24.04/26.04 の起動

デフォルトユーザーは、`ubuntu` です。本サンプルの実行ユーザーも`ubuntu` なので、ユーザーIDを省略しています。

```console
$ mactl create -f server-ubuntu22.yaml 
リソースの作成要求が受け入れられました。ID: 0aafd

$ mactl get srv srv-ubuntu22
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
srv-ubuntu22     hv0           RUNNING       1    1024     192.168.1.178    host-bridge      8s

$ mactl ssh srv-ubuntu22
Welcome to Ubuntu 22.04.5 LTS (GNU/Linux 5.15.0-187-generic x86_64)
```


