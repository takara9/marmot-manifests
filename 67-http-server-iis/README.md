# 仮想マシンをWeb(IIS)サーバーにする

Windows Server 2025 の仮想サーバーを起動し、AnsibleでIISを構成してWebサーバーにします。
接続はWindows標準のOpenSSHサーバー経由で、PowerShellをシェルとして使用します。

## 起動方法

```console
$ mactl create -f web3-ip-free.yaml
```

## Ansible Pingの実行

```console
$ ansible -i playbook/hosts.yaml all -m ansible.windows.win_ping
```

## Ansibleコレクションの準備

```console
$ cd playbook
$ ansible-galaxy collection install -r requirements.yaml
```

## IISの構成

```console
$ ansible-playbook -i hosts.yaml install.yaml -e @secrets.yaml --ask-vault-pass
```

`webserver_iis_cert_password`は`web3.labo.local.pfx`のパスワードです。平文ではなく`ansible-vault`で暗号化した`secrets.yaml`に保存し、`-e @secrets.yaml`で`ansible-playbook`に渡します。

### パスワードの保存(初回のみ)

```console
$ cd playbook
$ ansible-vault create secrets.yaml
```

Vaultのパスワードを設定後、エディタが開くので次の内容を保存します。

```yaml
webserver_iis_cert_password: <pfxのパスワード>
```

内容の確認や変更は`ansible-vault view secrets.yaml`、`ansible-vault edit secrets.yaml`で行います。
Vaultのパスワードを毎回入力したくない場合は、ファイルに保存して`--vault-password-file <ファイル>`を指定します(このファイルはリポジトリに含めないでください)。

実施内容は、IISのインストール、`index.html`の配置、PKCS#12証明書(`roles/webserver-iis/files/web3.labo.local.pfx`)のインポートとHTTPS(443)バインド、Windowsファイアウォールの開放、W3SVCの起動です。

## TLS証明書の有効期限の確認

```console
$ ansible-playbook -i hosts.yaml check-cert.yaml
```

Windowsの証明書ストア(`LocalMachine\My`)にある秘密鍵付き証明書の、サブジェクト、サムプリント、有効期限、残り日数、IISのHTTPSバインドでの使用有無を収集して表示します。
残り日数が`check_cert_warn_days`(既定30日)以下の証明書は警告します。収集結果はfact `check_cert_certificates`に格納されます。

## TLS証明書の更新

更新用のPKCS#12ファイルを`playbook/roles/renew-cert/files/`に置き、そのパスワードを`secrets.yaml`の`renew_cert_password`に追加します(`ansible-vault edit secrets.yaml`)。

```console
$ cd playbook
$ ansible-playbook -i hosts.yaml renew-cert.yaml -e @secrets.yaml --ask-vault-pass
```

ファイル名が既定の`web3.labo.local.pfx`と異なる場合は`-e renew_cert_file=<ファイル名>`で指定します。
証明書を`LocalMachine\My`にインポートし、IISのHTTPS(443)バインドの証明書を新しいものに置き換えて、HTTPSの応答を確認します。
`-e renew_cert_remove_old=true`を指定すると、置き換え前の証明書をストアから削除します。

## 確認方法

```console
$ curl http://web3.host-bridge/
$ curl -k https://web3.host-bridge/
```

## 削除

```console
$ mactl del -f web3-ip-free.yaml
```
