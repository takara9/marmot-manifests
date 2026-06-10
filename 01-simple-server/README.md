# defaultネットワークに繋がるサーバー

外部から接続できないサーバーの起動方法です。
外から接続できませんが、仮想サーバーからは外へアクセスすることができます。



## サーバーの起動方法

マニフェスト `server-01.yaml` を以下のように適用することで、仮想サーバーが起動します。

```console
$ mactl create -f server-01.yaml
リソースの作成要求が受け入れられました。ID: 574b4
```

```console
$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-01        marmot0       PENDING       1    1024     N/A              default        
```

```console
$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-01        marmot0       RUNNING       1    1024     N/A              default  
```

このサーバーが接続する `NETWORK` の `default` は、Marmotホスト内部の専用のネットワークです。独自の内部DHCPでIPアドレスを提供して、外部と通信する時は、NATでソースIPアドレスを、MarmotホストのIPアドレスに付け替えて、外部と繋がります。

この default ネットワークだけに接続するサーバーは、外部すなわち、Marmot以外のPCなどからアクセスするためのIPアドレスを持ちません。しかし、仮想サーバーは、default ネットワークに接続することで、apt-get などでモジュールを導入することができます。

## シリアルコンソールへのアクセス方法

この default ネットワークだけに接続された仮想サーバーにログインする
には、以下のコマンドを利用することで、シリアルコンソールに接続できます。

```console
$ mactl console <server name>
```

コマンド実行後に、何も応答が無い時は、リターンキーを押します。
すると、ログインプロンプトが表示されるので、ユーザー root, パスワード ubunut でログインできます。
（注：将来、ログインユーザーとパスワードは変更になります。）

シリアルコンソールからログアウトするには、`logout` など、Linuxから
ログアウトするコマンドを実行することで、ログアウトできます。

また、シリアルコンソールの制御から脱出した場合は、`Ctrl+]`を押すことで
元のシェルの制御に戻ります。また、コンソールに戻るには、`mactl console <server name>`
を実行することで、コンソール操作を継続できます。


## サーバーの削除
仮想サーバーを削除するには、`mactl del server/srv <server name>`を実行します。

```
$ mactl del srv server-01
server "server-01" deleted successfully
```

削除要求が受け入れられると、削除が完了すると、リストから表示されなくなります。

```
$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-01        marmot0       DELETING      1    1024     N/A              default  
```


## DNSサーバーの設定

DHCPでIPアドレスは、自動で割り当てられるものの、DNSは手作業で設定する必要があります。

```console
root@server-01:~# vi /etc/systemd/resolved.conf 
root@server-01:~# grep ^DNS= resolved.conf 
DNS=192.168.1.9
```

設定を反映して確認、IPアドレスが解決できたらOK

```console
root@server-01:~# systemctl daemon-reload
root@server-01:~# systemctl restart systemd-resolved.service
root@server-01:~# dig +short www.yahoo.co.jp
edge12.g.yimg.jp.
124.83.184.252
```

## パッケージリストの更新

DNSが参照できるようになれば、`apt-get update` が動作するようになります。

```console
root@server-01:~# apt-get update
Hit:1 http://archive.ubuntu.com/ubuntu noble InRelease
＜以下省略＞
```

