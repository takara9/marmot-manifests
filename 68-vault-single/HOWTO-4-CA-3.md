# Vault の Root CA を Windows / Linux / macOS にインストールして信頼させる

[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) で Vault の PKI エンジンに作成した Root CA（`labo.local Root CA`）を、クライアント OS の信頼ストアへ登録する手順です。
登録すると、Vault の中間CAが発行したサーバー証明書（例: `app.labo.local`）を、ブラウザや curl などが警告なしで受け入れます。

> **注意**
> - 登録するのは **Root CA 証明書（公開情報）のみ**です。秘密鍵やサーバー証明書の `private_key` は絶対に配布しないでください。
> - Root CA を信頼すると、その CA が発行した全ての証明書を信頼することになります。ラボ環境など、管理下の端末にのみ登録してください。
> - 中間CAは、サーバー側が `certificate` + `ca_chain` を連結して提示するため、クライアントには Root CA だけ登録すれば足ります。

## 1. Root CA 証明書の取得

HOWTO-4-CA-2.md の手順で作成した `root_ca.crt` をそのまま使えます。
別の端末で取得する場合は、Vault の PKI API から認証なしで取得できます（`pki/ca/pem` は公開エンドポイントです）。

```bash
export VAULT_ADDR=https://vault.labo.local:8200

# PEM 形式で取得（Vault の証明書を未信頼の場合は初回のみ -k が必要）
curl -sk $VAULT_ADDR/v1/pki/ca/pem -o root_ca.crt

# DER 形式が必要な場合（Windows の GUI など）
curl -sk $VAULT_ADDR/v1/pki/ca -o root_ca.der
```

内容とフィンガープリントを確認します。**フィンガープリントは Vault 管理者から別経路で受け取った値と照合**してください（初回取得に `-k` を使うため、改ざん確認が必要です）。

```bash
openssl x509 -in root_ca.crt -noout -subject -issuer -dates -fingerprint -sha256
```

```text
subject=CN = labo.local Root CA
issuer=CN = labo.local Root CA
notBefore=...
notAfter=...
sha256 Fingerprint=XX:XX:...
```

`subject` と `issuer` が同じ（自己署名）であること、`CN = labo.local Root CA` であることを確認します。

Vault サーバー上であれば、次でも取得できます。

```bash
vault read -field=certificate pki/cert/ca > root_ca.crt
```

## 2. Linux

ファイルの拡張子は **`.crt`** にします（Debian 系は `.crt` 以外を無視します）。

### 2.1 Ubuntu / Debian

```bash
sudo apt-get install -y ca-certificates
sudo cp root_ca.crt /usr/local/share/ca-certificates/labo-local-root-ca.crt
sudo update-ca-certificates
```

```text
Updating certificates in /etc/ssl/certs...
1 added, 0 removed; done.
```

### 2.2 RHEL / Rocky Linux / AlmaLinux / CentOS / Fedora

```bash
sudo cp root_ca.crt /etc/pki/ca-trust/source/anchors/labo-local-root-ca.crt
sudo update-ca-trust extract
```

### 2.3 openSUSE / SLES

```bash
sudo cp root_ca.crt /usr/share/pki/trust/anchors/labo-local-root-ca.crt
sudo update-ca-certificates
```

### 2.4 Arch Linux

```bash
sudo trust anchor --store root_ca.crt
```

### 2.5 動作確認

```bash
# システムの信頼ストアを使って検証（--cacert は不要）
curl -v https://app.labo.local/

openssl s_client -connect app.labo.local:443 -servername app.labo.local </dev/null 2>/dev/null \
  | grep -E "Verify return code|Verification"
# Verify return code: 0 (ok)
```

### 2.6 ストアを参照しないアプリケーションの注意点

| アプリケーション | 対応 |
|---|---|
| Firefox | 独自の信頼ストアを使用。`設定 > プライバシーとセキュリティ > 証明書 > 証明書を表示 > 認証局 > インポート` で `root_ca.crt` を登録し「このCAによるウェブサイトの識別を信頼する」を有効化。または `about:config` で `security.enterprise_roots.enabled` を `true`（OS のストアを参照） |
| Chrome / Chromium（Linux） | NSS DB を使用。`certutil -d sql:$HOME/.pki/nssdb -A -t "C,," -n "labo.local Root CA" -i root_ca.crt`（`libnss3-tools` が必要） |
| Java | `sudo keytool -importcert -trustcacerts -alias labo-local-root -file root_ca.crt -cacerts -storepass changeit`（Debian 系では `update-ca-certificates` が `/etc/ssl/certs/java/cacerts` も更新） |
| Python (requests) | `export REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt`（RHEL 系は `/etc/pki/tls/certs/ca-bundle.crt`） |
| Node.js | `export NODE_EXTRA_CA_CERTS=/path/to/root_ca.crt` |
| Docker | `/etc/docker/certs.d/<registry>/ca.crt` に配置 |

### 2.7 削除する場合

```bash
# Ubuntu / Debian
sudo rm /usr/local/share/ca-certificates/labo-local-root-ca.crt
sudo update-ca-certificates --fresh

# RHEL 系
sudo rm /etc/pki/ca-trust/source/anchors/labo-local-root-ca.crt
sudo update-ca-trust extract
```

## 3. macOS

### 3.1 コマンドラインで登録（推奨）

システム全体（全ユーザー）の System キーチェーンに、信頼済みルートとして登録します。管理者権限が必要です。

```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain root_ca.crt
```

自分のユーザーのみに登録する場合は、ログインキーチェーンへ登録します（`sudo` 不要、確認ダイアログが表示されます）。

```bash
security add-trusted-cert -r trustRoot \
  -k ~/Library/Keychains/login.keychain-db root_ca.crt
```

### 3.2 GUI で登録

1. `root_ca.crt` をダブルクリックし、キーチェーンアクセスで「システム」（またはログイン）に追加します。
2. キーチェーンアクセスで `labo.local Root CA` をダブルクリックします。
3. 「信頼」を展開し、「この証明書を使用するとき」を **「常に信頼」** に変更します。
4. 管理者パスワードまたは Touch ID で確定します。

### 3.3 構成プロファイルによる配布（複数台）

MDM（Jamf、Intune など）では、`com.apple.security.root` ペイロードに Root CA を含めた構成プロファイルを配布します。

### 3.4 動作確認

```bash
# 証明書が登録され、信頼設定されていることを確認
security find-certificate -c "labo.local Root CA" -a /Library/Keychains/System.keychain

# 検証（Verify に成功すれば "root_ca.crt: OK"）
security verify-cert -c root_ca.crt

curl -v https://app.labo.local/
```

Safari / Chrome / Edge は macOS のキーチェーンを使用します。Firefox は上記 Linux の注意点と同様、`security.enterprise_roots.enabled` を `true` にするか、Firefox 側にインポートします。

### 3.5 削除する場合

```bash
sudo security remove-trusted-cert -d root_ca.crt
sudo security delete-certificate -c "labo.local Root CA" /Library/Keychains/System.keychain
```

## 4. Windows

Windows では「信頼されたルート証明機関」ストアへ登録します。`.crt`（PEM または DER）をそのまま使えます。

### 4.1 PowerShell で登録（推奨）

**管理者として実行した PowerShell** で、コンピューター全体（ローカルコンピューター）に登録します。

```powershell
Import-Certificate -FilePath .\root_ca.crt `
  -CertStoreLocation Cert:\LocalMachine\Root
```

現在のユーザーのみに登録する場合（管理者権限不要。確認ダイアログが表示されます）:

```powershell
Import-Certificate -FilePath .\root_ca.crt `
  -CertStoreLocation Cert:\CurrentUser\Root
```

### 4.2 certutil で登録

```cmd
:: コンピューター全体（管理者のコマンドプロンプト）
certutil -addstore -f "Root" root_ca.crt

:: 現在のユーザーのみ
certutil -user -addstore -f "Root" root_ca.crt
```

### 4.3 GUI で登録

1. `root_ca.crt` をダブルクリックし「証明書のインストール」をクリックします。
2. 保存場所で「ローカル コンピューター」（全ユーザー）または「現在のユーザー」を選びます。
3. 「証明書をすべて次のストアに配置する」を選択し、「参照」から **「信頼されたルート証明機関」** を選びます。
4. ウィザードを完了します。現在のユーザーへの登録時はセキュリティ警告が出るので、サムプリントを確認して「はい」を選びます。

### 4.4 Active Directory のグループポリシーで配布（ドメイン環境）

`コンピューターの構成 > ポリシー > Windows の設定 > セキュリティの設定 > 公開キーのポリシー > 信頼されたルート証明機関` に `root_ca.crt` をインポートすると、ドメイン参加の全端末へ配布されます。

### 4.5 動作確認

```powershell
# 登録の確認
Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -like "*labo.local Root CA*"

# 検証
certutil -verify root_ca.crt

# HTTPS アクセス（--cacert なしで成功すること）
curl.exe -v https://app.labo.local/
Invoke-WebRequest https://app.labo.local/ -UseBasicParsing
```

Edge / Chrome は Windows の証明書ストアを使用します。Firefox は `about:config` の `security.enterprise_roots.enabled` を `true` にすると Windows のストアを参照します。

### 4.6 WSL 上の Linux

WSL は Windows のストアを引き継がないため、WSL 内で「2. Linux」の手順を別途実施してください。

### 4.7 削除する場合

```powershell
# サムプリントを確認して削除
Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -like "*labo.local Root CA*"
Remove-Item Cert:\LocalMachine\Root\<Thumbprint>
```

```cmd
certutil -delstore "Root" "labo.local Root CA"
```

## 5. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `unable to get local issuer certificate` | Root CA が未登録、またはサーバーが中間CAを提示していない。サーバー証明書には `certificate` と `ca_chain` を連結して設定する（HOWTO-4-CA-2.md 参照） |
| ブラウザでのみ警告が出る | ブラウザが独自ストアを使用している（Firefox、Linux の Chrome）。2.6 / 3.4 / 4.5 を参照 |
| `certificate has expired` | サーバー証明書の TTL 切れ。`vault write pki_int/issue/labo-local ...` で再発行する |
| `certificate is not valid for app.labo.local` | アクセスしたホスト名が CN/SAN と不一致。ロール `labo-local` の `allowed_domains` と発行時の `common_name` / `alt_names` を確認する |
| `update-ca-certificates` で `0 added` | 拡張子が `.crt` ではない、または `/usr/local/share/ca-certificates/` 以外に置いている |
| Root CA 更新後に検証が失敗する | Root CA を再作成すると別の証明書になる。全クライアントで古い証明書を削除し、新しい `root_ca.crt` を登録する |

## 6. 運用上の推奨事項

- Root CA の配布前に、**SHA-256 フィンガープリントを別経路で照合**する。
- Root CA の有効期限（本手順では 10 年）が近づく前に、新しい Root CA（`issuer_name` を変更）を作成し、クライアントへ先行配布する。
- 配布する場合は、構成管理ツール（Ansible、MDM、グループポリシー）を使用して手作業を減らす。
- 不要になった端末・環境では、上記の削除手順で Root CA を必ず削除する。
