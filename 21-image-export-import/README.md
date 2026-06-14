# マシンイメージのエクスポートとインポート
仮想マシンの起動イメージを tgz ファイルとして外部へ取り出すことができます。
また、tgzファイルをインポートして、仮想マシンの起動イメージとして利用できます。


対象のイメージを以下として、エクスポートとインポートを実行して確認します。
```console
$ mactl image list
NAME            STATUS     SYNCED    LV     QCOW2  AGE
----            ------     ------    ------  -----  ---
srv110          AVAILABLE  COMPLETE  no      yes    52s
```

## エクスポート

```console
$ mactl image export srv110
marmot-machine-image-srv110.tgz
```

## インポート

インポートを確認するために、イメージを消します。

```console
$ mactl image list
  No   IMAGE-ID  IMAGE-NAME        STATUS        NODE-NAME     ROLE     LV    QCOW2  CREATED-AT               
   5   07781     srv110            AVAILABLE     marmot3       master   no    yes    2026-06-14T16:53:51+09:00

$ mactl image delete 07781
イメージが削除されました。ID: 07781

$ mactl image list
  No   IMAGE-ID  IMAGE-NAME        STATUS        NODE-NAME     ROLE     LV    QCOW2  CREATED-AT               
```

先にエクスポートしたイメージをインポートします。イメージをリストして、ロードできたことを確認します。

```console
$ mactl image import marmot-machine-image-srv110.tgz 
{"id":"6dd82","message":"Image imported successfully"}

$ mactl image list
  No   IMAGE-ID  IMAGE-NAME        STATUS        NODE-NAME     ROLE     LV    QCOW2  CREATED-AT               
   5   6dd82     srv110            AVAILABLE     marmot1       master   no    yes    2026-06-14T17:05:01+09:00
```