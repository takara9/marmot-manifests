# 稼働中の仮想サーバーのOSイメージを保存する

稼働中の仮想マシンが、起動イメージを作ることができます。
これにより、色々なモジュールやプログラムをインストールした仮想サーバーから、ゴールデンイメージを作ることができます。
ゴールデンイメージを利用することで、サーバーのセットアップ時間を短縮できます。

## 手順
- ID が必要なため、`mactl server list` で IDを取得します。
- `mactl server createimage 1d91a srv110` で イメージを登録します。
- `mactl get img` で作成できたいことを確認します。


## 実行例
```console
$ mactl server list
  NO   SERVER-ID   SERVER-NAME           STATUS        CPU  RAM(MB)   NODE          IP-ADDRESS       NETWORK        
   1   1d91a       server-110            RUNNING       2    2028      marmot3       192.168.1.110    host-bridge    

$ mactl server createimage 1d91a srv110
サーバーのイメージ作成が受け付けられました。ID: 1d91a

$ mactl get img
NAME            STATUS     SYNCED    LV     QCOW2  AGE
----            ------     ------    ------  -----  ---
websrv          AVAILABLE  COMPLETE  no      yes    4d
docker          AVAILABLE  COMPLETE  no      yes    3d
ubuntu22.04     AVAILABLE  COMPLETE  mixed   yes    2h
ubuntu24.04     AVAILABLE  COMPLETE  mixed   yes    2h
srv110          MIXED      DEGRADE   no      yes    7s

# しばらく待ちます
$ mactl get img
NAME            STATUS     SYNCED    LV     QCOW2  AGE
----            ------     ------    ------  -----  ---
websrv          AVAILABLE  COMPLETE  no      yes    4d
docker          AVAILABLE  COMPLETE  no      yes    3d
ubuntu22.04     AVAILABLE  COMPLETE  mixed   yes    2h
ubuntu24.04     AVAILABLE  COMPLETE  mixed   yes    2h
srv110          AVAILABLE  COMPLETE  no      yes    52s
```

