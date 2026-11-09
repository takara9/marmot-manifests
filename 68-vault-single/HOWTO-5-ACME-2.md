Vault側の設定が完了していると仮定した状態で、app.labo.local のサーバー証明書を更新する手順について、詳しく説明してください。


# app.labo.local の証明書を ACME で更新する手順

## 前提と重要な点

前回の手順で `vault write pki_int/issue/...` により発行した証明書は ACME アカウントに紐づいていないため、certbot から「更新」はできません。**最初の1回だけ ACME で新規発行し、以後は自動更新に移行する**流れになります。旧証明書は有効期限まで残るので、切り替え時の停止はありません。

| 項目 | 前提 |
|---|---|
| Nginx ホスト | `app.labo.local` で稼働、80/443 を待ち受け |
| Vault → Nginx | Vault が `app.labo.local` を名前解決でき、80番ポートに到達できる（http-01 検証用） |
| ACME ディレクトリ | `https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory` |
| ルートCA | Nginx ホストが `root_ca.crt` を信頼済み |

## 手順

### 1. certbot の準備と Vault の TLS を信頼させる

```bash
sudo apt install certbot python3-certbot-nginx

# ルートCAを OS の信頼ストアへ
sudo cp root_ca.crt /usr/local/share/ca-certificates/labo-root-ca.crt
sudo update-ca-certificates

# certbot（Python requests）に信頼ストアを使わせる
export REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt
```

snap 版の certbot は OS の信頼ストアを参照しないため、上記の環境変数が必須です。cron/systemd で更新する場合は、後述のとおり環境変数も渡します。

### 2. 初回の発行（認証のみ、Nginx 設定は書き換えない）

```bash
sudo -E certbot certonly --nginx \
  --server https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory \
  --cert-name app.labo.local \
  -d app.labo.local \
  --register-unsafely-without-email \
  --key-type ecdsa
```

- `certonly --nginx` は、チャレンジ応答のためだけに Nginx を一時利用します。
- 成功すると、次のファイルができます。

| ファイル | 内容 |
|---|---|
| `/etc/letsencrypt/live/app.labo.local/fullchain.pem` | サーバー証明書 + 中間CA |
| `/etc/letsencrypt/live/app.labo.local/privkey.pem` | 秘密鍵 |

`live/` 配下は最新版へのシンボリックリンクです。更新後も同じパスで参照できます。

### 3. Nginx のパスを差し替える

```nginx
ssl_certificate     /etc/letsencrypt/live/app.labo.local/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/app.labo.local/privkey.pem;
```

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### 4. 更新の自動化を設定する

**(a) 更新タイミングを証明書の寿命に合わせる。** ロールの `ttl` が 30日以下だと、certbot の既定（残り30日未満で更新）では毎回更新されてしまうため、`/etc/letsencrypt/renewal/app.labo.local.conf` の `[renewalparams]` の前（ファイル先頭側）に追記します。

```ini
renew_before_expiry = 10 days
```

（例: `ttl=720h` の証明書は、発行から約20日後に更新されます。）

**(b) 更新後に Nginx を再読み込みする。**

```bash
sudo tee /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh <<'EOF'
#!/bin/sh
systemctl reload nginx
EOF
sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
```

**(c) 定期実行に環境変数を渡す。** apt 版は `certbot.timer` が1日2回実行します。systemd の drop-in で環境変数を追加します。

```bash
sudo systemctl edit certbot.service
```
```ini
[Service]
Environment=REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt
```
```bash
systemctl list-timers | grep certbot   # タイマーの稼働確認
```

### 5. 動作確認

```bash
# 更新を強制して、更新処理と reload までの一連を確認
sudo -E certbot renew --cert-name app.labo.local --force-renewal

# 配信されている証明書の発行元と期限を確認
openssl s_client -connect app.labo.local:443 -servername app.labo.local </dev/null 2>/dev/null \
  | openssl x509 -noout -issuer -serial -dates
```

更新前後で `serial` と `notAfter` が変わっていれば成功です。`--dry-run` はサーバーによって挙動が異なるため、動作確認には上記の強制更新を推奨します。

## よくあるエラーと対処

| 症状 | 原因 | 対処 |
|---|---|---|
| `certificate verify failed` | certbot が Vault の TLS を信頼していない | `REQUESTS_CA_BUNDLE` を設定（timer にも） |
| `Challenge failed` / `Timeout` (http-01) | Vault から `app.labo.local:80` に届かない、または名前解決できない | Vault ホストの DNS、FW、Nginx の 80番を確認 |
| `identifier not allowed` / `rejectedIdentifier` | ロールの `allowed_domains` 外 | ロールに `labo.local` / `allow_subdomains=true` を確認 |
| ディレクトリ取得で 404 / `acme is not enabled` | `config/acme` 未有効、または `config/cluster` の `path` が誤り | `vault read pki_int/config/acme` で確認 |
| `badNonce` や Location ヘッダーが欠落 | `secrets tune` のヘッダー許可漏れ | 前回の `tune` を再確認 |
| 更新後も古い証明書が配信される | Nginx が reload されていない | deploy hook の権限（実行ビット）を確認 |

## 補足

- **80番を開けられない場合は dns-01 を使います。** certbot の DNS プラグイン、または acme.sh/lego で対応できます。
- **旧証明書の失効は必須ではありません。** 短寿命なら期限切れを待つ運用で問題ありません。失効させる場合は `vault write pki_int/revoke serial_number=...` を使います。
- **acme.sh や lego に変える場合も、サーバー URL とルートCAの信頼設定だけが変わります。** 手順の流れは同じです。