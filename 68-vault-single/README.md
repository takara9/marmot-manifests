# vault-single

HashiCorp Vault をシングルノード構成で構築する Marmot 仮想サーバーの定義です。

## 仮想サーバー構成

[../vault-single.yaml](../vault-single.yaml) で定義されています。

| 項目 | 値 |
|---|---|
| サーバー名 | `vault-single` |
| CPU | 2 |
| メモリ | 2048 MB |
| イメージ | Ubuntu 24.04 |
| SSH認証 | `ubuntu` ユーザー、GitHub公開鍵（`takara9.keys`）を使用 |
| IPアドレス | `192.168.1.251`（社内DNS登録済み、`vault.labo.local`） |
| ネットワーク | `host-bridge`、ネットマスク `/24`、デフォルトGW `192.168.1.1` |

## Ansible構成

- [install.yaml](install.yaml): エントリーポイントとなる Playbook。`linux` → `vault` の順でロールを適用。
- [hosts.yaml](hosts.yaml): インベントリ。`vault-single` ホストを対象に、`mactl-ssh` 経由でSSH接続。

### roles/linux

OSレベルの共通設定を行うロール。

- スワップを即時無効化し、`/etc/fstab` のswapエントリを削除、`swap.target` を無効化・マスク
- `/etc/profile.d/vault-env.sh` を配置し、ログインユーザーに対し以下の環境変数を自動設定
  - `VAULT_ADDR="https://127.0.0.1:8200"`
  - `VAULT_SKIP_VERIFY=true`

### roles/vault

Vault本体のインストール・初期化・unseal自動化を行うロール。

1. 前提パッケージ（`gnupg`, `software-properties-common`, `curl`, `jq`）をインストール
2. HashiCorp公式APTリポジトリを登録し、`vault` パッケージをインストール
3. [templates/vault.hcl](roles/vault/templates/vault.hcl) を `/etc/vault.d/vault.hcl` へ配置（`node_id` はホスト名を自動設定）
4. Vaultサービスを起動し、HTTPS/HTTPでの疎通を確認したうえで初期化状態を判定
5. 未初期化の場合のみ `vault operator init` を実行し、Unsealキー・Root Tokenを画面出力（**この時にしか表示されないため必ず控えること**）
6. 生成されたUnsealキーで初回unsealを実施
7. 再起動時にも自動でunsealできるよう、Unsealキーを `/opt/vault/unseal-keys.json`（root専用、mode 0600）に保存し、起動時に実行される `vault-unseal.service`（systemd）と [vault-auto-unseal.sh.j2](roles/vault/templates/vault-auto-unseal.sh.j2) を配置

> **セキュリティ上の注意**: `/opt/vault/unseal-keys.json` にはUnsealキーが平文で保存されます。ラボ用途の簡易構成であり、本番環境ではクラウドKMS等によるAuto-unsealの利用を推奨します。

## 実行方法

Marmot の `mactl` コマンドでマニフェストを適用すると、仮想サーバーの作成から Vault のセットアップまで一括で行われます。

```bash
cd 68-vault-single
mactl create -f vault-single.yaml
```

[vault-single.yaml](../vault-single.yaml) の `spec.ansible` に `playbook: playbook/install.yaml` ・ `inventory: playbook/hosts.yaml` が指定されており、`mactl create` が仮想サーバー起動後に自動でこの Playbook を実行するため、`linux` → `vault` ロールが適用されVaultサーバーが構成されます。

Playbookのみを対象環境に対して手動で再実行したい場合（デバッグ時など）は、以下のように直接 `ansible-playbook` を実行することも可能です。

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml
```

構文のみ確認する場合:

```bash
ansible-playbook playbook/install.yaml -i playbook/hosts.yaml --syntax-check
```

## Vault CLI の利用方法

`ubuntu` ユーザーでログインすると環境変数が自動設定されるため、そのまま `vault` コマンドが利用できます。

```bash
ssh ubuntu@vault.labo.local
vault status
vault login   # 初期化時に表示された Root Token を入力
```

## 注意事項

- Unsealキー・Root Tokenは初期化時の一度しか表示されません。画面出力された内容は安全な場所に保管してください。
- TLS証明書は `/opt/vault/tls/tls.crt` ・ `/opt/vault/tls/tls.key` を参照する設定になっているため、別途配置が必要です。
