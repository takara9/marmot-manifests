# etcd クラスタ (3ノード / TLS)

Ubuntu 24.04 の仮想マシン3台に、TLS (HTTPS) 有効の etcd 3.4 クラスタを構築します。

| サーバー | IPアドレス | 役割 |
|---|---|---|
| etcd-cluster-1 | 192.168.1.191 | etcd メンバー、プライベートCA |
| etcd-cluster-2 | 192.168.1.192 | etcd メンバー |
| etcd-cluster-3 | 192.168.1.193 | etcd メンバー |

## 構成

```
72-etcd-cluster/
├── etcd-cluster.yaml        # 仮想マシン3台のマニフェスト
└── playbook/
    ├── hosts.yaml           # インベントリ
    ├── install.yaml         # roles: linux → etcd
    └── roles/
        ├── linux/           # OS確認、データディスクの ext4 作成とマウント
        └── etcd/            # etcd導入、TLS証明書の発行、クラスタ設定、環境変数
```

- 各サーバーには `/var/lib/etcd` 用の 6GB ボリューム (`/dev/vdb`) が付きます。
- 証明書は `/etc/etcd/pki/` に配置されます。CA (`ca.key`, `ca.crt`) は etcd-cluster-1 に置かれ、各ノードのサーバー証明書をそこで発行します。
- クライアント通信 (2379) は HTTPS で、サーバー認証のみです。ピア通信 (2380) は相互 TLS です。

## 前提

- 内部DNS (192.168.1.9) と host-bridge ネットワークが利用できること。
- コントロールノードから 192.168.1.191〜193 へ `ubuntu` ユーザーで SSH 接続できること (`etcd-cluster.yaml` の `auth.url` に登録した公開鍵に対応する秘密鍵を使用)。
- Ansible と、コレクション `community.general`、`community.crypto`、`ansible.posix` がコントロールノードにあること。

## 起動方法

1. 仮想マシンを作成し、`RUNNING` になるまで待ちます。

    ```console
    mactl create -f etcd-cluster.yaml
    mactl server list | grep etcd-cluster
    ```

2. Ansible で疎通を確認します。

    ```console
    ansible -i playbook/hosts.yaml all -m ping
    ```

3. playbook を実行します。

    ```console
    ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
    ```

    最後に `endpoint health --cluster` の結果とメンバー一覧が表示されます。

## 動作確認

各ノードの `ubuntu` ユーザーには、次の環境変数が `~/.bashrc` の先頭に設定されます。

```
export ETCDCTL_API=3
export ETCDCTL_ENDPOINTS=https://192.168.1.191:2379,https://192.168.1.192:2379,https://192.168.1.193:2379
export ETCDCTL_CACERT=/etc/etcd/pki/ca.crt
```

そのため、ノードにログインすれば引数なしで `etcdctl` を使えます。

```console
ubuntu@etcd-cluster-1:~$ etcdctl endpoint health --cluster
ubuntu@etcd-cluster-1:~$ etcdctl endpoint status --cluster -w table
ubuntu@etcd-cluster-1:~$ etcdctl member list -w table
ubuntu@etcd-cluster-1:~$ etcdctl put key1 value1
ubuntu@etcd-cluster-1:~$ etcdctl get key1
```

別のマシンから接続するには、`/etc/etcd/pki/ca.crt` をそのマシンにコピーして `ETCDCTL_CACERT` に指定してください。

## 注意事項

- **接続方法**: `hosts.yaml` は `ansible_host` に IP を指定した直接 SSH 接続にしています。`mactl-ssh` は並列接続時に認証エラーが発生することがあり、全ノードを同時に操作する必要があるこの構成には向かないためです。
- **forks**: etcd は過半数のノードが起動するまで起動完了にならないため、全ノードで同時に起動する必要があります。Ansible の forks はノード数 (3) 以上にしてください (デフォルトの 5 で問題ありません)。
- **初回のデータ破棄**: 初回の実行時のみ、パッケージ導入で自動起動した単一ノード用のデータ (`/var/lib/etcd/default`) を削除します。クラスタ構築が成功するとマーカー `/var/lib/etcd/.etcd-bootstrapped` が作られ、以降の再実行ではデータを削除しません。構築をやり直したい場合は、全ノードでこのマーカーを削除してから実行してください。
- **再実行時の再起動**: 設定や証明書が変わると、ハンドラが全ノードを同時に再起動するため、クラスタが短時間停止します。
- **マニフェストの `ansible:`**: `etcd-cluster.yaml` の `ansible:` はコメントアウトのままにしてください。有効にするとサーバー作成時に1台ずつ実行され、他ノードが未作成のため失敗します。
- **CA の保護**: `ca.key` は etcd-cluster-1 の `/etc/etcd/pki/ca.key` (root のみ読み取り可) にあります。このノードを削除すると CA が失われるため、証明書の再発行ができなくなります。
- **TLS 設定**: クライアント証明書は要求しません。etcd 3.4 では `ETCD_TRUSTED_CA_FILE` を指定するとクライアント証明書が必須になるため、クライアント通信側には設定していません。
- **証明書の有効期間**: CA は 10 年、サーバー証明書は 825 日です。自動更新は行わないため、期限前に手動で更新してください。
- **認証**: etcd のユーザー/ロール認証は有効にしていません。閉じたネットワークでの利用が前提です。

## 削除

```console
mactl del -f etcd-cluster.yaml
```
