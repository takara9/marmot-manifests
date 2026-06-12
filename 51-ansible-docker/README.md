# 仮想マシンにdockerをインストール

仮想マシンが起動した後、自動的にansibleで docker-ce をインストールします。

```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-110
    comment: Ansibleで、dockerコマンドをインストールするサーバー
spec:
    cpu: 2
    memory: 2048
    osVariant: ubuntu24.04
    auth:
        url: https://github.com/takara9.keys
        users:
         - root      # ansible 用ユーザー
         - ubuntu    # ログイン操作用のユーザー
    ansible-playbook: playbook/install.yaml # docker をセットアップ用
    networkInterface:
        - networkname: host-bridge
          address: 192.168.1.110  # このアドレスで ssh ログイン可能
          netmasklen: 24
          nameservers:
            addresses:
                - 192.168.1.9
            search:
                - labo.local
          routes:
            - to: default
              via: 192.168.1.1
```