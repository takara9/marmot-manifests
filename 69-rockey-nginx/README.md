# 69-rockey-nginx

Rocky Linux 上に NGINX をセットアップし、TLS証明書を配置して HTTPS で公開するための marmot マニフェスト一式です。

## 構成

- [web4.yaml](web4.yaml): marmot サーバー定義（`web4`）。プロビジョニング後に `playbook/install.yaml` を実行します。
- [playbook/hosts.yaml](playbook/hosts.yaml): Ansible インベントリ。
- [playbook/install.yaml](playbook/install.yaml): NGINX のインストール・TLS証明書配置・サービス起動を行うプレイブック。
- [playbook/check-cert-expiry.yaml](playbook/check-cert-expiry.yaml): 配置済みTLS証明書の有効期限を確認するプレイブック。
- [playbook/renew-cert.yaml](playbook/renew-cert.yaml): TLS証明書ファイルの正当性検証とWebサービスの正常性確認を行うプレイブック。
- [playbook/vars/cert.yaml](playbook/vars/cert.yaml): 各プレイブック共通のTLS証明書指定（`cert_basename` / `nginx_cert_dir`）。
- [playbook/roles/web-nginx/files](playbook/roles/web-nginx/files): コントロールノードに置くTLS証明書・秘密鍵（`{{ cert_basename }}.crt` / `.key`）。

## デプロイ方法

1. marmot でサーバーを作成します。

    ```bash
    marmot apply -f web4.yaml
    ```

    サーバー作成後、`web4.yaml` の `spec.ansible` 設定に従って `playbook/install.yaml` が自動実行されます。

2. 手動でプレイブックを再実行する場合は、`playbook` ディレクトリで以下を実行します。

    ```bash
    cd playbook
    ansible-playbook -i hosts.yaml install.yaml
    ```

    このプレイブックは以下を行います。

    - TLS証明書・秘密鍵を `playbook/roles/web-nginx/files` から管理対象ノードの `/etc/nginx/ssl` へ配置
    - NGINX のインストールと `conf.d/default.conf` の配置
    - firewalld が有効な場合は `http` / `https` サービスの許可
    - `index.html` の配置とサービス起動確認（HTTPS での疎通確認）

## TLS証明書の有効期限のチェック方法

`playbook/check-cert-expiry.yaml` を実行すると、`/etc/nginx/ssl` 配下の `*.crt` の有効期限（残り日数）が一覧表示されます。

```bash
cd playbook
ansible-playbook -i hosts.yaml check-cert-expiry.yaml
```

- 残り日数が `cert_expiry_warning_days`（デフォルト30日）以下の証明書は警告メッセージとして表示されます。
- 警告の閾値は `-e` オプションで変更できます。

    ```bash
    ansible-playbook -i hosts.yaml check-cert-expiry.yaml -e cert_expiry_warning_days=60
    ```

## TLS証明書の更新手順

1. 新しい証明書・秘密鍵を `playbook/roles/web-nginx/files` に配置します。

    - ファイル名を既存の `cert_basename`（[playbook/vars/cert.yaml](playbook/vars/cert.yaml) で定義）と揃える場合は、そのまま同名で上書きします。
    - ファイル名（ドメイン）を変更する場合は、`playbook/vars/cert.yaml` の `cert_basename` を新しいファイル名（拡張子なし）に書き換えます。

2. `renew-cert.yaml` を実行します。証明書ファイルの正当性（フォーマット・証明書と秘密鍵の対応・有効期限）を確認したうえで、管理対象ノードへ証明書を配置してnginxをリロードし、Webサービスの正常性（nginx起動・証明書配置・HTTPS応答）を確認します。

    ```bash
    cd playbook
    ansible-playbook -i hosts.yaml renew-cert.yaml
    ```

3. 必要に応じて有効期限を再確認します。

    ```bash
    ansible-playbook -i hosts.yaml check-cert-expiry.yaml
    ```
