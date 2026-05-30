# defaultネットワークに繋がるサーバー

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
server-01        marmot-fre    PENDING       1    1024     N/A              default        
```

```console
$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK        
----             ----          ------        ---  -------  ----------       -------        
server-01        marmot-fre    RUNNING       1    1024     N/A              default  
```

このサーバーが接続する `NETWORK` の `default` は、Marmotホスト内部の専用のネットワークです。独自の内部DHCPでIPアドレスを提供して、外部と通信する時は、NATでソースIPアドレスを、MarmotホストのIPアドレスに付け替えて、外部と繋がります。

この default ネットワークだけに接続するサーバーは、外部すなわち、Marmot以外のPCなどからアクセスするためのIPアドレスを持ちません。しかし、仮想サーバーは、default ネットワークに接続することで、apt-get などでモジュールを導入することができます。

## シリアルコンソールへのアクセス方法

この default ネットワークだけに接続された仮想サーバーにログインする
には、以下のコマンドを利用することで、シリアルコンソールに接続できます。

```console
$ mactl console <server name>
```

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
server-01        marmot-fre    DELETING      1    1024     N/A              default  
```