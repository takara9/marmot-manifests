# Helm 3 インストール用プレイブック

## 作業内容

helm の実行形式を https://github.com/helm/helm/releases からダウンロードして、
/usr/local/bin へインストールする。

helm チャートのリポジトリをhttps://kubernetes-charts.storage.googleapis.com/
からインストールする。


## 結果

以下のコマンドが動作するようになる。

* helm version
* helm search repo stable
* helm inspect values stable/rocketchat > config.yml


## デフォルト変数

helm_version       : v3.3.1


