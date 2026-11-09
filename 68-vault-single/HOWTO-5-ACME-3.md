# certbot を使わずに ACME で Nginx の証明書を更新する方法

主な選択肢は次の3つです。

| 方法 | 特徴 | 向いているケース |
|---|---|---|
| **A. Nginx 公式 ACME モジュール** | Nginx 自身が発行から更新まで行う。外部ツール、cron、reload が不要 | Nginx 単体で完結させたい場合 |
| **B. acme.sh** | シェルスクリプトだけで動く軽量クライアント。dns-01 などプラグインも豊富 | 既存の Nginx をそのまま使いたい場合 |
| **C. lego** | Go 製の単一バイナリ | コンテナや単一バイナリで配布したい場合 |

ここでは A と B を詳しく説明します。前提は前回と同じで、Vault 側は設定済みで、ディレクトリ URL は `https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory` とします。

---

## A. Nginx 公式 ACME モジュール（ngx_http_acme_module）

Nginx に組み込む ACME モジュールで、証明書の取得・更新を Nginx の設定だけで行います。証明書の要求、インストール、更新を、Nginx の設定内のディレクティブから直接行えます。

### 1. モジュールの導入

このモジュールは、nginx 1.29.0 以降、nginx-module-acme という名前でビルド済みパッケージとしても提供されています。お使いの Nginx のバージョンとパッケージ元（nginx.org の公式リポジトリか、ディストリビューション版か）を先に確認してください。ディストリビューション版では提供されていない場合があります。

```bash
nginx -v
sudo apt install nginx-module-acme   # nginx.org の公式リポジトリを追加済みの場合
```

`nginx.conf` のメインコンテキスト（先頭）でロードします。

```nginx
load_module modules/ngx_http_acme_module.so;
```

### 2. Nginx の設定

```nginx
http {
    resolver 192.168.1.1;          # ★必須。ラボの DNS を指定（vault.labo.local を引けること）

    acme_issuer vault {
        uri  https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory;
        ssl_trusted_certificate /etc/nginx/ssl/root_ca.crt;   # Vault の TLS を検証するルートCA
        state_path /var/cache/nginx/acme-vault;               # アカウント鍵・証明書の保存先
        accept_terms_of_service;
        # contact admin@labo.local;
    }

    acme_shared_zone zone=acme_shared:1M;

    # http-01 のチャレンジ用に 80番の待ち受けが必要
    server {
        listen 80;
        server_name app.labo.local;
        location / { return 404; }
    }

    server {
        listen 443 ssl;
        server_name app.labo.local;

        acme_certificate vault;
        ssl_certificate       $acme_certificate;
        ssl_certificate_key   $acme_certificate_key;
        ssl_certificate_cache max=2;

        location / { ... }
    }
}
```

設定のポイントは次のとおりです。

- このモジュールは `http` ブロックに `resolver` の設定が必要です。
- HTTP-01 チャレンジを処理するため、80番ポートのリスナーが必要です。
- `ssl_trusted_certificate` は、ACME サーバー（Vault）の証明書を検証する CA ファイルです。省略した場合はシステムの CA バンドルが使われます。ルートCAを OS に登録済みなら省略できます。
- 証明書は `$acme_certificate` と `$acme_certificate_key` の変数経由で提供され、`ssl_certificate` と `ssl_certificate_key` に割り当てます。ファイルパスを指定する必要はありません。
- `state_path` のディレクトリは、Nginx のワーカーユーザーが書き込める権限にしてください。

### 3. 反映と確認

```bash
sudo mkdir -p /var/cache/nginx/acme-vault && sudo chown nginx:nginx /var/cache/nginx/acme-vault   # ユーザー名は環境による
sudo nginx -t && sudo systemctl reload nginx

# 発行状況はエラーログに出力される
sudo tail -f /var/log/nginx/error.log

# 配信中の証明書の確認
openssl s_client -connect app.labo.local:443 -servername app.labo.local </dev/null 2>/dev/null \
  | openssl x509 -noout -issuer -serial -dates
```

更新はモジュールが自動で行い、reload や cron は不要です。更新の詳しい条件（残り期間の閾値など）は、公式ドキュメント（[ngx_http_acme_module](https://nginx.org/en/docs/http/ngx_http_acme_module.html)）で確認してください。

---

## B. acme.sh を使う方法

Nginx の設定は従来どおり（ファイルパス指定）のままにできます。

### 1. インストール

```bash
curl https://get.acme.sh | sh -s email=admin@labo.local
source ~/.bashrc
```

インストールすると cron ジョブが登録され、毎日更新チェックが走ります。

### 2. 初回発行

```bash
acme.sh --issue \
  --server https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory \
  --ca-bundle /etc/nginx/ssl/root_ca.crt \
  -d app.labo.local \
  --webroot /var/www/html \
  --keylength ec-256 \
  --days 10
```

| オプション | 意味 |
|---|---|
| `--server` | Vault の ACME ディレクトリ URL |
| `--ca-bundle` | Vault の TLS を検証するルートCA |
| `--webroot` | http-01 のチャレンジファイルを置く Nginx のドキュメントルート |
| `--days` | 残り何日で更新するか（既定は60日。短寿命の証明書では必ず調整） |

Nginx を止められるなら `--standalone`、80番が使えないなら dns-01（`--dns dns_xxx`）も使えます。

### 3. Nginx へのインストール

```bash
acme.sh --install-cert -d app.labo.local \
  --key-file       /etc/nginx/ssl/app.labo.local.key \
  --fullchain-file /etc/nginx/ssl/app.labo.local.crt \
  --reloadcmd      "systemctl reload nginx"
```

この設定は保存され、**以後の自動更新のたびに同じ場所へ配置して reload まで実行されます**。Nginx の `ssl_certificate` は、既存の `/etc/nginx/ssl/app.labo.local.{crt,key}` をそのまま使えます。

### 4. 動作確認

```bash
acme.sh --list                                        # 登録済み証明書と次回更新予定
acme.sh --renew -d app.labo.local --force --ecc       # 強制更新（ECC 鍵のため --ecc が必要）
crontab -l | grep acme                                # 自動更新の cron が入っていること
```

---

## 使い分けの目安

| 観点 | A. Nginx モジュール | B. acme.sh |
|---|---|---|
| 運用の手間 | 最小（Nginx のみ） | cron と reload が必要 |
| 導入の前提 | モジュール対応の Nginx（1.29 系のパッケージ等） | 既存の Nginx で可 |
| チャレンジ | 主に http-01 | http-01 / dns-01 / standalone など |
| 証明書のファイル | ファイルとして取り出しにくい（Nginx が管理） | 通常のファイルとして他サービスにも流用可 |

通常のファイルとして他のサービスにも使い回す必要があるなら B、Nginx だけで完結させたいなら A が適しています。

## 共通の注意点

- **Vault の TLS 信頼が必須です。** `ssl_trusted_certificate`（A）または `--ca-bundle`（B）で、ルートCAを必ず指定してください。
- **Vault から `app.labo.local:80` に到達できる必要があります。** DNS と FW の確認項目は前回と同じです。
- **旧証明書は ACME アカウントと無関係です。** 初回発行後に切り替えれば、旧証明書は期限切れを待つだけで構いません。
- **Vault の ACME 設定（`config/acme`、`secrets tune` のヘッダー許可）が不足していると、どのクライアントでも失敗します。**

実環境の Nginx のバージョンや導入元を教えていただければ、モジュール版を使えるかどうかを含めて具体的な手順に絞り込めます。