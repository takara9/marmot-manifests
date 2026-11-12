# etcd シングルノード (TLS)

Ubuntu 24.04 の仮想マシン1台に、TLS (HTTPS) 有効の etcd 3.4 を構築します。

| サーバー | IPアドレス | 役割 |
|---|---|---|
| etcd-single | 192.168.1.190 | etcd、プライベートCA |

## 構成

```
71-etcd-single/
├── etcd-single.yaml         # 仮想マシンのマニフェスト
└── playbook/
    ├── hosts.yaml           # インベントリ
    ├── install.yaml         # roles: linux → etcd
    └── roles/
        ├── linux/           # OS確認、データディスクの ext4 作成とマウント
        └── etcd/            # etcd導入、TLS証明書の発行、etcd設定、環境変数
```

- サーバーには `/var/lib/etcd` 用の 6GB ボリューム (`/dev/vdb`) が付きます。
- CA と、それで署名したサーバー証明書は `/etc/etcd/pki/` に作成されます。
- クライアント通信 (2379) は HTTPS で、サーバー認証のみです。ピア通信 (2380) は 127.0.0.1 の HTTP です。

## 前提

- 内部DNS (192.168.1.9) と host-bridge ネットワークが利用できること。
- `mactl` と `mactl-ssh` が使えること (`hosts.yaml` は `mactl-ssh` で接続します)。
- Ansible と、コレクション `community.general`、`community.crypto`、`ansible.posix` がコントロールノードにあること。

## 起動方法

`etcd-single.yaml` に `ansible:` が設定されているため、仮想マシンの作成時に playbook が自動で実行されます。

```console
mactl create -f etcd-single.yaml
mactl server list | grep etcd-single
```

playbook を単独で実行する場合 (再実行や設定変更の反映) は、次のとおりです。

```console
ansible -i playbook/hosts.yaml all -m ping
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
```

最後に `etcdctl endpoint health` の結果が表示されます。

## 動作確認

`ubuntu` ユーザーには、次の環境変数が `~/.bashrc` の先頭に設定されます。

```
export ETCDCTL_API=3
export ETCDCTL_ENDPOINTS=https://192.168.1.190:2379
export ETCDCTL_CACERT=/etc/etcd/pki/ca.crt
```

そのため、サーバーにログインすれば引数なしで `etcdctl` を使えます。

```console
ubuntu@etcd-single:~$ etcdctl endpoint health
ubuntu@etcd-single:~$ etcdctl put key1 value1
ubuntu@etcd-single:~$ etcdctl get key1
ubuntu@etcd-single:~$ etcdctl del key1
```

別のマシンから接続するには、`/etc/etcd/pki/ca.crt` をそのマシンにコピーして `ETCDCTL_CACERT` に指定してください。証明書の SAN には `etcd.labo.local`、`etcd-single`、`localhost`、`127.0.0.1`、`192.168.1.190` が含まれます。

## 注意事項

- **TLS 設定**: クライアント証明書は要求しません。etcd 3.4 では `ETCD_TRUSTED_CA_FILE` を指定するとクライアント証明書が必須になるため、設定していません。
- **証明書の有効期間**: CA は 10 年、サーバー証明書は 825 日です。自動更新は行わないため、期限前に手動で更新してください。
- **認証**: etcd のユーザー/ロール認証は有効にしていません。閉じたネットワークでの利用が前提です。
- **データディスク**: `/dev/vdb` が存在しない、またはブロックデバイスでない場合は playbook が失敗します。フォーマット済みのディスクは再フォーマットされません。
- **再実行**: 設定や証明書が変わると etcd が再起動します。再実行しても、データと証明書は作り直されません。
- **複数ノード構成**: クラスタ構成は [72-etcd-cluster](../72-etcd-cluster/README.md) を参照してください。

## 削除

```console
mactl del -f etcd-single.yaml
```
