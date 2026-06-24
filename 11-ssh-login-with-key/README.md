# ssh-key を作成して、サーバーに公開鍵をセットしてログイン

ssh鍵ペアを作成して、仮想サーバーを構築して、ログインする方法です。


## 鍵のペアを生成

```console
$ ssh-keygen -t ed25519 -f ./vmkey -C "For marmot VMs" -N ""
Generating public/private ed25519 key pair.
Your identification has been saved in ./vmkey
Your public key has been saved in ./vmkey.pub
The key fingerprint is:
SHA256:gCiiW8LVndnFuJqn3TN/ouZ6TpLH/5GF2t+GeKOxN3A For marmot VMs
The key's randomart image is:
+--[ED25519 256]--+
|          o.     |
|   .... +...     |
|o .....+ ..      |
|+..    . .     . |
|o..     S     . .|
| +     o .o .oE o|
|.       +o.+o+.+ |
|       . .+*o+*o+|
|         .*+B*+=o|
+----[SHA256]-----+
```

## 公開鍵を表示して、マニフェストにセットする

```console
$ cat vmkey.pub 
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFOSkHRgel695I/R+TEyfCr98IcoBJtTyFLrNpb7V331 For marmot VMs
```

spec.auth.publicKey に上記の文字列をセットします。
そして、spec.auth.usersの下に、公開鍵を設定したいユーザーを列挙する

```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-91
    comment: marmotホストが繋がるネットワークに接続する仮想サーバー
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    auth:
        publicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFOSkHRgel695I/R+TEyfCr98IcoBJtTyFLrNpb7V331 For marmot VMs"
        users:
         - ubuntu
以下省略
```

## ログイン

`ssh USER@192.168.1.91` または DNSサーバーが memotd が実行するサーバーに向いていれば、`ssh USER@server-91.host-bridge`　でログインできます。USER@は、カレントユーザーと仮想サーバーに設定したユーザー名が同じであれば、省略できます。


```console
$ mactl create -f server-91.yaml 
リソースの作成要求が受け入れられました。ID: a8962


$ ssh ubuntu@10.10.0.91 -i vmkey 
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-117-generic x86_64)

 * Documentation:  https://help.ubuntu.com
 * Management:     https://landscape.canonical.com
 * Support:        https://ubuntu.com/pro

 System information as of Fri Jun 12 01:35:06 UTC 2026

  System load:             0.17
  Usage of /:              11.3% of 14.46GB
  Memory usage:            17%
  Swap usage:              0%
  Processes:               133
  Users logged in:         0
  IPv4 address for enp1s0: 192.168.1.91
  IPv6 address for enp1s0: 2400:2411:bc83:3a00:8818:b8ff:fe20:b8e

Expanded Security Maintenance for Applications is not enabled.

0 updates can be applied immediately.

Enable ESM Apps to receive additional future security updates.
See https://ubuntu.com/esm or run: sudo pro status


The list of available updates is more than a week old.
To check for new updates run: sudo apt update


The programs included with the Ubuntu system are free software;
the exact distribution terms for each program are described in the
individual files in /usr/share/doc/*/copyright.

Ubuntu comes with ABSOLUTELY NO WARRANTY, to the extent permitted by
applicable law.
```

