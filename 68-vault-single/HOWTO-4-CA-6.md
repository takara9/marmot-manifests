# インストールした Root CA の削除

[HOWTO-4-CA-3.md](HOWTO-4-CA-3.md) で登録した Root CA（`labo.local Root CA`）を、各 OS の信頼ストアから削除する手順です。
Root CA の更新に伴う旧 Root の削除や、検証環境の撤去時に使います。

> **削除前の確認**
> - Root CA の更新に伴う削除は、旧チェーンで発行した証明書がすべて失効・期限切れになってから行います（[HOWTO-4-CA-5.md](HOWTO-4-CA-5.md) の「旧 Issuer の後始末」を参照）。
> - 同じ名前の証明書が複数ある場合（Root CA の更新後など）は、名前指定だと新しい Root まで消えることがあります。**サムプリント（フィンガープリント）で対象を特定**してください。

## 1. Linux

### 1.1 OS の信頼ストア

```bash
# Ubuntu / Debian
sudo rm /usr/local/share/ca-certificates/labo-local-root-ca.crt
sudo update-ca-certificates --fresh

# RHEL / Rocky Linux / AlmaLinux / CentOS / Fedora
sudo rm /etc/pki/ca-trust/source/anchors/labo-local-root-ca.crt
sudo update-ca-trust extract

# openSUSE / SLES
sudo rm /usr/share/pki/trust/anchors/labo-local-root-ca.crt
sudo update-ca-certificates --fresh

# Arch Linux
sudo trust anchor --remove root_ca.crt
```

RHEL 系と openSUSE では、削除後に `update-ca-trust extract` / `update-ca-certificates` を実行しないと、ストアに反映されません。

### 1.2 OS のストアとは別に登録した場合

```bash
# Chrome / Chromium（NSS DB）
certutil -d sql:$HOME/.pki/nssdb -D -n "labo.local Root CA"

# Java
sudo keytool -delete -alias labo-local-root -cacerts -storepass changeit
```

Firefox は「設定 > プライバシーとセキュリティ > 証明書 > 証明書を表示 > 認証局」から該当の証明書を選んで削除します。
`security.enterprise_roots.enabled` を `true` にしている場合は、OS のストアから削除すれば反映されます。

環境変数（`SSL_CERT_FILE`、`REQUESTS_CA_BUNDLE`、`NODE_EXTRA_CA_CERTS`）や、Docker の `/etc/docker/certs.d/<registry>/ca.crt` に設定している場合は、それらも削除します。

## 2. macOS

```bash
sudo security remove-trusted-cert -d root_ca.crt
sudo security delete-certificate -c "labo.local Root CA" /Library/Keychains/System.keychain
```

ログインキーチェーンに登録した場合は、キーチェーンを指定します。

```bash
security delete-certificate -c "labo.local Root CA" ~/Library/Keychains/login.keychain-db
```

GUI の場合は、キーチェーンアクセスで `labo.local Root CA` を選んで削除します。

## 3. Windows

管理者権限の PowerShell で、サムプリントを確認してから削除します。

```powershell
Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -like "*labo.local Root CA*"
Remove-Item Cert:\LocalMachine\Root\<Thumbprint>
```

```cmd
certutil -delstore "Root" "labo.local Root CA"
```

現在のユーザーに登録した場合は、`Cert:\CurrentUser\Root` を指定するか、`certutil -user -delstore` を使います。
GUI の場合は、`certmgr.msc`（ユーザー）または `certlm.msc`（コンピューター）を開き、「信頼されたルート証明機関 > 証明書」から削除します。

## 4. 別のストアを持つ環境

次の環境は、Windows / macOS のストアとは独立しています。それぞれの環境で個別に削除してください。

| 環境 | 対応 |
|---|---|
| WSL | WSL 内で「1. Linux」の手順を実施 |
| Kubernetes ノード | 「1. Linux」の手順で削除し、containerd / CRI-O を再起動 |
| Kubernetes の Pod | ConfigMap / trust-manager の `Bundle`、イメージ内の CA を削除（[HOWTO-4-CA-4.md](HOWTO-4-CA-4.md) 参照） |
| 構成管理・MDM・GPO | 配布元から削除しないと、再び配布される |

## 5. 削除の確認

削除後は、Root CA を信頼しない状態になっているはずです。

```bash
curl -v https://app.labo.local/
# 例: curl: (60) SSL certificate problem: unable to get local issuer certificate
```

```powershell
# Windows: 該当の証明書が表示されないこと
Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -like "*labo.local Root CA*"
```

```bash
# macOS: 該当の証明書が見つからないこと
security find-certificate -c "labo.local Root CA" -a /Library/Keychains/System.keychain
```
