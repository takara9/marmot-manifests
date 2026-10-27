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
$ ansible-playbook -i hosts.yaml install.yaml
```

実施内容は、IISのインストール、`index.html`の配置、自己署名証明書の作成とHTTPS(443)バインド、Windowsファイアウォールの開放、W3SVCの起動です。

## 確認方法

```console
$ curl http://web3.host-bridge/
$ curl -k https://web3.host-bridge/
```

## 削除

```console
$ mactl del -f web3-ip-free.yaml
```
