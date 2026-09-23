# DHCPクライアントとしての仮想サーバー

MarmotのIPアドレス管理機能を使わず、LANのDHCPサーバーからIPアドレスを割当ることができます。

## ２種のDHCPクライアントのマニフェスト

- srv-dhcp4.yaml : DHCP4 の IPアドレスをキャッチ
- srv-dhcp6.yaml : DHCP6 の IPアドレスをキャッチ

## ssh でのログイン方法

`mactl get server` コマンドでは、IPアドレスが表示されないので、
`mactl console srv-dhcp6` でログインして、`ip a`コマンドで、
割当られたIPアドレスを調べます。

ログアウトして、取得したIPアドレスで、ssh するとログインすることができます。
