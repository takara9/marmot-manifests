# dify サーバーの起動

difyサーバーを、docker-compose で起動します仮想サーバーです。
この手順が完了すると、以下のスクリーンショットの画面が表示されます。

```console
$ mactl create -f server-dify-2.yaml
$ ansible -i hosts.yaml all -m ping
$ ansible-playbook -i hosts.yaml playbook/setup-dify.yaml
```

ブラウザで、このリンクをクリックする
http://SERVER-IP-ADDRESS/install

DNSがmarmotサーバーを向いていれば、以下のURLでアクセスできます。

- [http://dify-server-2.host-bridge/install](http://dify-server-2.host-bridge/install)



この画面が表示されます。
![dify](screen-shot.png)

