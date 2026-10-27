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

`iis_cert_password`は`web3.labo.local.pfx`のパスワードです。平文ではなく`ansible-vault`で暗号化した`secrets.yaml`に保存し、`-e @secrets.yaml`で`ansible-playbook`に渡します。

### パスワードの保存(初回のみ)

```console
$ cd playbook
$ ansible-vault create secrets.yaml
```

Vaultのパスワードを設定後、エディタが開くので次の内容を保存します。

```yaml
iis_cert_password: <pfxのパスワード>
```

内容の確認や変更は`ansible-vault view secrets.yaml`、`ansible-vault edit secrets.yaml`で行います。
Vaultのパスワードを毎回入力したくない場合は、ファイルに保存して`--vault-password-file <ファイル>`を指定します(このファイルはリポジトリに含めないでください)。

実施内容は、IISのインストール、`index.html`の配置、PKCS#12証明書(`roles/iis/files/web3.labo.local.pfx`)のインポートとHTTPS(443)バインド、Windowsファイアウォールの開放、W3SVCの起動です。

## 確認方法

```console
$ curl http://web3.host-bridge/
$ curl -k https://web3.host-bridge/
```

## 削除

```console
$ mactl del -f web3-ip-free.yaml
```
