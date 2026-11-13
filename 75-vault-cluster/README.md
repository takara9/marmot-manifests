# vault-cluster

HashiCorp Vault を 3 ノードの HA クラスタ(Raft 統合ストレージ)で構築する Marmot 仮想サーバーの定義と Ansible Playbook です。TLS 証明書を使い、起動時に自動で unseal します。

## 仮想サーバー構成

[vault-cluster.yaml](vault-cluster.yaml) で 3 台を定義しています。

| サーバー名 | IPアドレス | 役割 |
|---|---|---|
| `vault-cluster-1` | `192.168.1.194` | 初期化とプライベートCAを担当する先頭ノード |
| `vault-cluster-2` | `192.168.1.195` | フォロワー(Raft に自動参加) |
| `vault-cluster-3` | `192.168.1.196` | フォロワー(Raft に自動参加) |

| 項目 | 値 |
|---|---|
| CPU / メモリ | 2 / 2048 MB |
| イメージ | Ubuntu 24.04 |
| SSH認証 | `ubuntu` ユーザー、GitHub公開鍵(`takara9.keys`) |
| ネットワーク | `host-bridge`、`/24`、デフォルトGW `192.168.1.1`、DNS `192.168.1.9` |

## Ansible構成

```
playbook/
├── hosts.yaml        インベントリ(IPアドレスへ直接SSH)
├── install.yaml      エントリーポイント。linux → vault の順にロールを適用
└── roles/
    ├── linux/        スワップ無効化
    └── vault/
        ├── defaults/main.yaml
        ├── handlers/main.yaml
        ├── tasks/    main, install, tls, configure, init, unseal
        └── templates/  vault.hcl.j2, vault-auto-unseal.sh.j2, vault-unseal.service.j2
```

### roles/linux

スワップを即時無効化し、`/etc/fstab` のswapエントリ削除と `swap.target` の無効化・マスクを行います。

### roles/vault

| タスク | 内容 |
|---|---|
| `install.yaml` | HashiCorp公式APTリポジトリを登録して `vault` をインストールし、ログインユーザー向けに `/etc/profile.d/vault-env.sh`(`VAULT_ADDR`、`VAULT_CACERT`)を配置 |
| `tls.yaml` | 先頭ノードにプライベートCAを作成し、各ノードのサーバー証明書をそこで発行。CA証明書を全ノードに配布 |
| `configure.yaml` | `vault.hcl` を配置して Vault を起動 |
| `init.yaml` | 先頭ノードで `vault operator init` を実行(初回のみ)。Unseal キーを全ノードの `/opt/vault/unseal-keys.json`(root専用、0600)に配布 |
| `unseal.yaml` | 自動unsealスクリプトと `vault-unseal.service` を配置・有効化し、全ノードの unseal を確認。初回のみ Raft ピア一覧を表示 |

### TLS

- 証明書の SAN: `<ホスト名>.labo.local`、`<ホスト名>`、`localhost`、`127.0.0.1`、ノードIP
  - 追加したい場合は `vault_extra_san` に `["DNS:vault.labo.local"]` のように指定します。
- 配置先: `/opt/vault/tls/{ca.crt,vault.crt,vault.key}`
- CA秘密鍵は先頭ノードの `/opt/vault/ca/ca.key`(root専用、0700)にのみ存在します。
- 検証は CA 証明書で行うため、`VAULT_SKIP_VERIFY` は使いません。
- 有効期間は `vault_ca_days`(3650日)と `vault_cert_days`(825日)で変更できます。

### クラスタ構成(Raft)

- `retry_join` に全ノードを列挙しているため、フォロワーは起動順に関係なく自動参加します。
- フォロワーは先頭ノードの unseal 後に参加します。参加後は各ノードの自動unsealが動きます。

### 自動unseal

- `vault-unseal.service`(systemd)が `vault.service` の起動後に `/usr/local/bin/vault-auto-unseal.sh` を実行します。
- スクリプトは API 応答と Raft 参加(initialized)を最大 `vault_unseal_timeout` 秒(300)待ち、sealed なら Unseal キーで unseal します。
- OS再起動や Vault 再起動でも自動で利用可能状態に戻ります。
- 全ノードを同時に再起動しても、各ノードが自分で unseal するため復旧します。

> **セキュリティ上の注意**: Unsealキーは全ノードに平文で保存されます。ラボ用途の簡易構成であり、本番環境では KMS や Transit 等による auto-unseal を推奨します。

## 実行方法

### 1. 仮想サーバーの作成

```bash
cd 75-vault-cluster
mactl create -f vault-cluster.yaml
```

`spec.ansible` はコメントアウトしています。クラスタ構築は全ノードを同時に処理する必要があり、`mactl-ssh` は並列接続で認証エラーになるためです。Playbook は次の手順で手動実行します。

### 2. 事前準備(コントロールノード)

証明書の発行に `community.crypto` コレクションを使います。

```bash
ansible-galaxy collection install community.crypto
```

### 3. Playbook の実行

```bash
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
```

構文のみ確認する場合:

```bash
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml --syntax-check
```

初回実行の最後に **Unseal キーと Root Token が一度だけ表示されます**。必ず安全な場所に控えてください。再実行しても再表示されません。

## 動作確認

```bash
ssh ubuntu@192.168.1.194
vault status
vault login                    # 初期化時に表示された Root Token
vault operator raft list-peers # 3ノードが表示され、1ノードが leader
```

他ノードでも `vault status` を実行すると、`Initialized true` / `Sealed false` / `HA Enabled true` になっています。

### 再起動時の自動unsealの確認

```bash
ssh ubuntu@192.168.1.195 sudo reboot
# 数十秒後
ssh ubuntu@192.168.1.195 vault status   # Sealed false
systemctl status vault-unseal           # 起動時の実行結果
journalctl -u vault-unseal
```

## 主な変数

[roles/vault/defaults/main.yaml](playbook/roles/vault/defaults/main.yaml) で定義しています。

| 変数 | 既定値 | 説明 |
|---|---|---|
| `vault_leader` | グループ先頭のホスト | CA作成と初期化を行うノード |
| `vault_domain` | `labo.local` | 証明書FQDNのドメイン |
| `vault_key_shares` / `vault_key_threshold` | `5` / `3` | Unsealキーの分割数と必要数 |
| `vault_keys_file` | `/opt/vault/unseal-keys.json` | Unsealキーの保存先 |
| `vault_unseal_timeout` | `300` | 自動unsealの待機秒数 |
| `vault_extra_san` | `[]` | 証明書に追加するSAN |

## 注意事項

- Unsealキーと Root Token は初期化時の一度しか表示されません。
- 設定や証明書の変更で Vault が再起動されると、全ノードが同時に再起動して一時的に利用できなくなります。`vault-unseal` が自動復帰させます。無停止で更新したい場合は `install.yaml` に `serial: 1` を指定してください(初回構築では使用不可)。
- ノードを追加する場合は `hosts.yaml` と `vault-cluster.yaml` にノードを足し、Playbook を再実行します。Unsealキーは先頭ノードから自動配布されます。
- `vault-cluster.yaml` の `auth.users` に `vault` が含まれています。パッケージが作るシステムユーザー `vault` と衝突しないか確認してください。
