# 82-mail-server

`labo.local` ドメインのメールサーバー（Postfix + Dovecot）を、Marmot 仮想サーバー上に Ansible で構築します。

- サーバー間のメール交換: 宛先ドメインの **DNS の MX レコードを参照して配送**します。
- メールクライアント: **IMAP4**（STARTTLS / IMAPS）でメールを受信し、**SMTP submission**（587 / 465、認証必須）で送信します。IMAP にはメールを送る機能がないため、クライアントには受信（IMAP）と送信（SMTP）の両方を設定します。

## 仮想サーバー構成

[mail-server.yaml](mail-server.yaml) で定義されています。

| 項目 | 値 |
|---|---|
| サーバー名 | `mail` |
| CPU / メモリ | 1 / 1024 MB |
| イメージ | Ubuntu 24.04 |
| IPアドレス | `192.168.1.12`（`mail.labo.local`） |
| DNS | `192.168.1.9`、検索ドメイン `labo.local`（MX レコードの参照に必要） |

## 事前準備

DNS に次のレコードが必要です。[79-BindDNS](../79-BindDNS/playbook/vars/zones.yaml) には、すでに定義されています。

| レコード | 値 |
|---|---|
| A | `mail.labo.local` → `192.168.1.12` |
| MX | `labo.local` → `10 mail.labo.local.` |

```bash
dig +short MX labo.local            # 10 mail.labo.local.
dig +short A mail.labo.local        # 192.168.1.12
```

Playbook を実行するマシンには、Ansible、`community.crypto`・`community.general` コレクション、`passlib` が必要です。パスワードのハッシュ化に `passlib` を使います。

```bash
ansible-galaxy collection install community.crypto community.general
sudo apt install python3-passlib       # または pip install passlib
```

## 構築

```bash
cd 82-mail-server
mactl create -f mail-server.yaml
```

[mail-server.yaml](mail-server.yaml) の `spec.ansible` により、仮想サーバーの起動後に Playbook が自動で実行されます。Playbook のみを手動で実行する場合は、次のコマンドを使います。

```bash
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
```

Playbook は、実行の最後に **DNS の MX レコードがこのサーバー（`mail_hostname`）を指していること**を確認します。指していない場合は失敗するため、DNS を修正して再実行してください。

## 構成

| ファイル | 内容 |
|---|---|
| [playbook/install.yaml](playbook/install.yaml) | エントリーポイント。`mail_servers` に `mail_server`、`mail_clients` に `mail_client` を適用 |
| [playbook/hosts.yaml](playbook/hosts.yaml) | インベントリ。`mactl-ssh` 経由で SSH 接続 |
| [playbook/group_vars/all.yaml](playbook/group_vars/all.yaml) | ドメイン、ユーザー、別名 |
| [playbook/roles/mail_server](playbook/roles/mail_server) | Postfix + Dovecot の構築（変数は `defaults/main.yaml`） |
| [playbook/roles/mail_client](playbook/roles/mail_client) | 送信専用の Postfix（他のサーバー用、任意） |

### メールサーバー（mail_server）

1. Postfix、Dovecot（IMAP / LMTP）、動作確認用ツール（`swaks`、`dig`）をインストール
2. メールボックス用のユーザー `vmail`（uid 5000）を作成。メールは `/var/mail/vhosts/<ドメイン>/<ユーザー>/Maildir` に保存
3. TLS 証明書を用意（`mail_tls_mode`。後述）
4. Dovecot を設定: IMAP（143 は STARTTLS 必須、993 は TLS）、パスワードファイル認証、Postfix への LMTP 配送、Postfix への SMTP 認証の提供
5. Postfix を設定: 受信、MX 参照による送信、`submission`（587）と `smtps`（465）の追加
6. ポートの待ち受けと、MX レコードを確認

メールボックスは Linux のアカウントを作らない仮想ユーザーです。ログインやシェルは使えません。

| ポート | 用途 | 認証 |
|---|---|---|
| 25 | サーバー間のメール配送（受信） | 不要。**`labo.local` 宛のみ受け付け、他ドメインへの中継は拒否** |
| 587 | メールクライアントの送信（STARTTLS） | 必須 |
| 465 | メールクライアントの送信（TLS）。`mail_enable_smtps: false` で無効化 | 必須 |
| 143 | IMAP（STARTTLS 必須。平文ログインは拒否） | 必須 |
| 993 | IMAPS（TLS） | 必須 |

### メールの流れ

| 経路 | 動作 |
|---|---|
| 他のサーバー → `user@labo.local` | 送信側が MX レコードで `mail.labo.local` を引き、25 番ポートへ配送。Postfix が Dovecot（LMTP）経由でメールボックスに保存 |
| クライアント → 他ドメイン | 587 / 465 で認証して送信。Postfix が宛先ドメインの MX レコードを引いて配送（`relayhost` は使用しない） |
| クライアント ← メールボックス | IMAP で受信 |

外部のインターネットへ配送するには、仮想サーバーから外部への SMTP（25 番ポート）が通る必要があります。

### 送信側のサーバー（mail_client、任意）

`labo.local` の他のサーバーから、cron や監視のメールをこのサーバーへ送るための、送信専用の Postfix です。受信はせず（`loopback-only`）、宛先のないユーザー名（`root` など）は `<ユーザー>@labo.local` として MX 宛に配送します。

[playbook/hosts.yaml](playbook/hosts.yaml) の `mail_clients` にホストを追加して実行します。

```yaml
    mail_clients:
      hosts:
        web01:
        app01:
```

## 設定する変数

[playbook/group_vars/all.yaml](playbook/group_vars/all.yaml) の主な変数です。

| 変数 | 既定値 | 説明 |
|---|---|---|
| `mail_domain` | `labo.local` | メールドメイン |
| `mail_hostname` | `mail.labo.local` | メールサーバーの FQDN。DNS の MX の宛先と一致させる |
| `mail_users` | `admin`、`user1` | メールボックスを持つユーザー。**パスワードは仮値** |
| `mail_aliases` | `postmaster`、`root`、`abuse`、`hostmaster` → `admin` | 別名 |
| `mail_alias_domains` | `[]` | 別名ドメイン。例: `lab.internal`（`user@lab.internal` を `user@labo.local` へ配送） |

[roles/mail_server/defaults/main.yaml](playbook/roles/mail_server/defaults/main.yaml) の主な変数です。

| 変数 | 既定値 | 説明 |
|---|---|---|
| `mail_enable_smtps` | `true` | 465 番ポートを待ち受ける |
| `mail_mynetworks` | ループバックのみ | 認証なしで中継を許可するネットワーク。通常は変更しない |
| `mail_message_size_limit` | 26214400 | 1通の最大サイズ（バイト） |
| `mail_check_mx` | `true` | 実行の最後に MX レコードを確認する |
| `mail_configure_ufw` | `false` | UFW でメール関連のポートを開放する |
| `mail_tls_mode` | `selfsigned` | TLS 証明書の入手方法（後述） |

### ユーザーの追加・パスワード変更

`mail_users` を編集して、Playbook を再実行します。**パスワードは必ず変更してください。**

```bash
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml \
  -e '{"mail_users":[{"name":"admin","password":"強いパスワード"},{"name":"user1","password":"強いパスワード"}]}'
```

`ansible-vault` で暗号化した変数ファイルを使うのが安全です。作成済みのハッシュを使う場合は、`password` の代わりに `password_hash: "{SHA512-CRYPT}$6$..."` を指定します（`doveadm pw -s SHA512-CRYPT` で生成）。ユーザーを削除してもメールボックスのデータは削除されません。

## TLS 証明書

`mail_tls_mode` で選択します。証明書は `/etc/mail-server/tls/server.crt` と `server.key` に配置されます。

| `mail_tls_mode` | 内容 |
|---|---|
| `selfsigned`（既定） | サーバー上にプライベート CA を作成し、サーバー証明書を自動発行する |
| `provided` | 別途用意した証明書と秘密鍵（Ansible 実行マシン上のファイル）を配置する |

`provided` の例:

```yaml
mail_tls_mode: provided
mail_tls_cert_src: /path/to/mail.labo.local.fullchain.crt   # 必須
mail_tls_key_src: /path/to/mail.labo.local.key              # 必須
mail_tls_ca_src: /path/to/labo-ca.crt                       # 任意
```

`selfsigned` の場合は、メールクライアントに CA 証明書を信頼させてください。

```bash
scp ubuntu@mail.labo.local:/etc/mail-server/tls/ca.crt ./mail-labo-ca.crt
```

サーバー間の配送（25 番ポート）は、証明書を検証しない opportunistic TLS（`may`）です。

## メールクライアントの設定

| 項目 | 値 |
|---|---|
| ユーザー名 | `admin@labo.local`（`admin` でも可） |
| 受信 (IMAP) | `mail.labo.local`、ポート 143（STARTTLS）または 993（SSL/TLS） |
| 送信 (SMTP) | `mail.labo.local`、ポート 587（STARTTLS）または 465（SSL/TLS） |
| 認証 | 通常のパスワード認証（IMAP / SMTP とも同じ） |

接続先は IP アドレスではなく、証明書の名前（`mail.labo.local`）にしてください。

## 動作確認

DNS と MX レコード:

```bash
dig +short MX labo.local
```

別のサーバー（`mail_client` を適用済み）から送信し、`admin` に届くことを確認します。

```bash
echo "test" | mail -s "test from $(hostname)" admin@labo.local
```

メールサーバー上で、配送ログとメールボックスを確認します。

```bash
sudo journalctl -t postfix/smtpd -t postfix/lmtp --no-pager | tail
sudo find /var/mail/vhosts -path '*/new/*' -type f
```

メールクライアントと同じ手順（認証付き送信）で、SMTP を確認します。

```bash
swaks --server mail.labo.local --port 587 --tls --tls-ca-path ./mail-labo-ca.crt \
  --auth LOGIN --auth-user user1 --auth-password '<パスワード>' \
  --from user1@labo.local --to admin@labo.local
```

IMAP を確認します。

```bash
openssl s_client -connect mail.labo.local:993 -CAfile ./mail-labo-ca.crt -quiet
# a login admin <パスワード>
# b list "" "*"
```

他ドメイン宛が MX を参照することは、キューとログで確認します。

```bash
sudo postqueue -p
sudo journalctl -t postfix/smtp --no-pager | tail
```

## 注意事項

- `mail_users` のパスワードは**ラボ用の仮値**です。実際に使う前に変更するか、`ansible-vault` で上書きしてください。
- 受信用の 25 番ポートは認証なしで開いていますが、受け付けるのは `labo.local`（と別名ドメイン）宛のみで、他ドメインへの中継（オープンリレー）はしません。
- 外部のインターネットへの送信は、仮想サーバーの送信元 IP に対する逆引き（PTR）、SPF、DKIM などがないと、受信側に拒否または迷惑メール扱いされる場合があります。この構成は、`labo.local` 内のメール交換を目的としています。
- `.local` は mDNS の予約名です。systemd-resolved や avahi を使うクライアントでは、DNS ではなく mDNS で解決されることがあります。MX レコードを参照できない場合は、[79-BindDNS](../79-BindDNS/README.md) の「ドメイン名」の注意を確認してください。
- DNS のレコードを変更したら、[79-BindDNS](../79-BindDNS/README.md) の手順に従って DNS を更新し、メールサーバーの Playbook を再実行して MX レコードを確認してください。
- メールボックスは `/var/mail/vhosts` に保存されます。サーバーのルートディスク上にあるため、大量のメールを保存する場合は、別ディスクをマウントしてください。バックアップも別途検討してください。
- Dovecot と Postfix の設定は、Playbook が管理しています。サーバー上で直接変更した内容は、次回の実行で上書きされます。
