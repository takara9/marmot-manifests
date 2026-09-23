## ホストネットワークにブリッジ接続する仮想サーバー

ネットワーク host-birdgeを、仮想サーバーのマニフェストに指定すると、
marmotホストのネットワークにブリッジされたネットワークに、仮想サーバーが繋がります。

IPアドレス、デフォルトゲートウェイ、DNSサーバーなどをマニフェストで指定することで、
パソコンが繋がるネットワーク上にサーバーが起動することになります。


## VMにログインするためのキーを生成

仮想サーバーを作成する前に、SSHキーを作成しておくと便利です。

```bash
$ ssh-keygen -t ed25519 -C "For marmot VMs" -N ""
Generating public/private ed25519 key pair.
Enter file in which to save the key (/home/ubuntu/.ssh/id_ed25519): 
/home/ubuntu/.ssh/id_ed25519 already exists.
Overwrite (y/n)? 

$ cat ~/.ssh/id_ed25519.pub 
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIP6CPvNHHeF8rHScC7vrq7HMqTPQJcl08atMSqHk/gSy For marmot VMs
```

あまりお勧めでは無いですが、以下のコマンドで、ホームディレクトリに秘密鍵をコピーして、マニフェストの変更なしでログインできます。

```bash
$ cp id_ed25519 ~/.ssh
$ chmod 0400 ~/.ssh/id_ed25519
```


## ブリッジ接続する仮想サーバー ３種のマニフェスト

- srv-0201.yaml IPアドレス自動割り当て
- srv-0202.yaml IPアドレス手動設定
- srv-0203.yaml IPアドレス自動割り当て、GitHub公開鍵利用


## 仮想サーバーの起動方法

```bash
$ mactl create -f srv-0201.yaml 
リソースの作成要求が受け入れられました。ID: ebe84
```

起動して、ログインできるまでに、１分くらいかかります。


## ログイン方法
`macth ssh <SERVER-NAME>` で名前でログインできます。また、`ssh username@192.168.1.201` のようなIPアドレスとユーザー指定でのログインもできます。

```bash
$ mactl get server srv-0201
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
srv-0201         mh5           RUNNING       1    1024     192.168.1.201    host-bridge      12m

$ mactl ssh srv-0201
Welcome to Ubuntu 24.04.5 LTS (GNU/Linux 6.8.0-139-generic x86_64)
＜以下省略＞
```

