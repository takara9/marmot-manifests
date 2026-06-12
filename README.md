# marmot のマニフェスト集

サンプル構成やテストのための Marmot用マニフェスト集です。

## サーバーとネットワークの基本

- [defaultネットワークに繋がるサーバー](01-simple-server/README.md)
- [ホストネットワーク上にIPを持ちアクセス可能なサーバー](02-accessible-server/README.md)
- [DHCPクライアントとしての仮想サーバー](03-server-as-dhcp-client/README.md)

## プライベートネットワークと接続方式

- [PCにインストールした VPN クライアント経由で、marmot内部のプライベートネットワークへ接続する](04-connect-via-openvpn/README.md)
- [クラスタ横断のプライベートネットワークに繋がったサーバー](05-private-cluster-network-server/README.md)
- [プライベートネットワークに繋がったサーバー](30-server-on-private-net/README.md)
- [プライベートネットワークに繋がった エアギャップのサーバー](31-server-in-air-gapped/README.md)
- [プライベートネットワークに繋がったサーバーにパブリックIPを設定する](32-private-network-server/README.md)

## ロードバランサー

- [アプリケーションロードバランサーでWebサーバーを負荷分散する](06-application-loadbalancer/README.md)
- [ネットワークロードバランサーでサービスを負荷分散する](07-network-loadbalancer/README.md)

## 認証とアクセス

- [rootパスワードの変更](10-change-user-password/README.md)
- [ssh-key を作成して、サーバーに公開鍵をセットしてログイン](11-ssh-login-with-key/README.md)
- [GitHub公開鍵を使ったログイン設定](12-ssh-login-with-github-key/README.md)

## イメージ管理

- [稼働中の仮想サーバーのOSイメージを保存する](20-save-machine-image/README.md)
- [マシンイメージのエクスポートとインポート](21-image-export-import/README.md)

## ボリューム管理

- [ブートボリュームの拡大](40-expand-boot-volume/README.md)
- [永続ボリューム](41-volume-attach/README.md)

## 自動構築 (Ansible)

- [仮想マシン起動後に自動的に Ansible プレイブックを適用する](50-setup-ansible/README.md)
- [仮想マシンにdockerをインストール](51-ansible-docker/README.md)

