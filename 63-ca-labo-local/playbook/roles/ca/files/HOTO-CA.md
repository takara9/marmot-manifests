# CA (プライベート認証局) 使い方

このディレクトリのスクリプトは、Ansible の `ca` ロールによって
サーバー上の CA ホームディレクトリ (`{{ ca_home }}`、デフォルト `/CertificationAuthorities`)
に配置され、そのディレクトリをカレントディレクトリとして実行する前提になっています。

```
cd /CertificationAuthorities
```

## ディレクトリ構成

- `CA/` … ルート(自己署名)CAの秘密鍵・証明書 (`ca.key` / `ca.crt`)。`setup_ca.sh` が作成。
- `CERTS/<FQDN>/` … `issue_certs.sh` で発行した証明書一式。FQDNごとにサブディレクトリが作られる。
  - `<FQDN>.key` … サーバー秘密鍵
  - `<FQDN>.csr` … 証明書署名要求
  - `<FQDN>.crt` … CAが署名した証明書 (PEM)
  - `<FQDN>.p12` / `<FQDN>.pfx` … 秘密鍵+証明書+CA証明書チェーンを含む PKCS#12 バンドル(同一内容)
  - `<FQDN>.serial` … シリアル番号管理用
  - `<FQDN>_csr.txt` / `<FQDN>_cert.txt` … 人間可読なCSR/証明書のダンプ
  - `info.txt` / `v3ext` … 発行時に使ったホスト情報・拡張設定

## 使用手順

1. **CAの開設(初回のみ)**
   ```
   ./setup_ca.sh
   ```
   `CA/ca.key`, `CA/ca.crt` が既に存在する場合は何もしない。

2. **証明書の発行内容を編集**

   `issue_certs.sh` を編集し、発行したいドメイン・SAN構成を指定する。

   ```sh
   export MY_DOMAIN=dify.local

   export MY_HOST=dify
   export FQDN=$MY_DOMAIN
   export CSR_SUBJ="/C=JP/ST=Tokyo/OU=Dify Service/O=System Labo/CN=$MY_DOMAIN"
   export V3EXT=v3ext.txt
   export SAN_HOSTS="console.$MY_DOMAIN app.$MY_DOMAIN api.$MY_DOMAIN upload.$MY_DOMAIN enterprise.$MY_DOMAIN trigger.$MY_DOMAIN"
   ./create_csr_noip.sh
   ./signature_cert.sh
   ```

   - `SAN_HOSTS` に列挙したFQDNは、同一証明書の SubjectAltName (DNS.3以降) として追加される。
     これにより `dify.local` 用に証明書を1枚だけ発行し、複数サブドメインで共用できる。
   - IPアドレスもSANに含めたい場合は `create_csr_noip.sh` の代わりに `create_csr.sh` を使う
     (実行時に `nslookup` でホストのIPを解決してSANに追加する)。
   - CA自身の中間証明書を発行する場合は `V3EXT=v3ext_ca.txt` を指定する。

3. **証明書の発行実行**
   ```
   ./issue_certs.sh
   ```
   既にCSR/証明書が存在するFQDNはスキップされる(再実行しても壊れない)。

4. **CA証明書・鍵をWebサーバーの公開領域へコピー(任意)**
   ```
   ./cp_ca_htdocs.sh
   ```
   `CERTS/ca.pem` / `CERTS/ca.p12` / `CERTS/ca.key` を生成し、クライアント配布用に使う。

## 発行した証明書の利用例

### Docker (`/etc/docker/certs.d`)
```
mkdir -p "/etc/docker/certs.d/<FQDN>"
cp "CERTS/<FQDN>/<FQDN>.crt" "/etc/docker/certs.d/<FQDN>/<FQDN>.cert"
cp "CERTS/<FQDN>/<FQDN>.key" "/etc/docker/certs.d/<FQDN>"
cp "CA/ca.crt" "/etc/docker/certs.d/<FQDN>"
```

### Microsoft IIS
`CERTS/<FQDN>/<FQDN>.pfx` をIISサーバーへ転送し、「個人」証明書ストアへインポートする
(パスワードはスクリプト内で `root` を指定。運用環境では変更を推奨)。
クライアント/サーバーの「信頼されたルート証明機関」ストアに `CA/ca.crt` を別途インポートしないと、
プライベートCA発行の証明書は信頼されない点に注意。

### nginx
`<FQDN>.crt` はサーバー証明書のみのため、クライアントが中間/CA証明書を検証できるよう
CA証明書を連結したファイルを用意してから指定する。

```
cat "CERTS/<FQDN>/<FQDN>.crt" "CA/ca.crt" > "/etc/nginx/certs/<FQDN>-fullchain.crt"
cp "CERTS/<FQDN>/<FQDN>.key" "/etc/nginx/certs/<FQDN>.key"
chmod 600 "/etc/nginx/certs/<FQDN>.key"
```

```nginx
server {
    listen 443 ssl;
    server_name <FQDN>;

    ssl_certificate     /etc/nginx/certs/<FQDN>-fullchain.crt;
    ssl_certificate_key /etc/nginx/certs/<FQDN>.key;
}
```

### Apache HTTP Server
```
cp "CERTS/<FQDN>/<FQDN>.crt" "/etc/apache2/ssl/<FQDN>.crt"
cp "CERTS/<FQDN>/<FQDN>.key" "/etc/apache2/ssl/<FQDN>.key"
cp "CA/ca.crt"               "/etc/apache2/ssl/ca.crt"
chmod 600 "/etc/apache2/ssl/<FQDN>.key"
```

```apache
<VirtualHost *:443>
    ServerName <FQDN>

    SSLEngine on
    SSLCertificateFile      /etc/apache2/ssl/<FQDN>.crt
    SSLCertificateKeyFile   /etc/apache2/ssl/<FQDN>.key
    SSLCertificateChainFile /etc/apache2/ssl/ca.crt
</VirtualHost>
```

いずれも、クライアント側の「信頼されたルート証明機関」に `CA/ca.crt` を事前にインポートしておかないと
プライベートCA発行の証明書は信頼されない点はIISと同様。

## 注意事項

- PKCS#12出力のパスワードは `signature_cert.sh` 内で `pass:root` に固定されている。本番利用時は変更すること。
- 証明書の有効期間は825日(`signature_cert.sh` の `-days 825`)。
- 一度発行したCSR/証明書は削除しない限り再発行されない。SAN構成を変更したい場合は
  対象の `CERTS/<FQDN>/` ディレクトリを削除してから `issue_certs.sh` を再実行する。
