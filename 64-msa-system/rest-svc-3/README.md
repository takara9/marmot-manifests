# 5桁乱数文字列 REST API サービス

このシナリオは 5 桁の数字文字列を乱数で返す REST API サービスです。

環境変数 `REST_API_RESPONSE_DELAY_SECONDS` で、リクエスト受信からレスポンス返却までの待ち時間を指定できます。
指定可能な範囲は 0 から 10 秒です。

```console
export REST_API_RESPONSE_DELAY_SECONDS='2.5'
mactl create -f server-63.yaml
```

起動後は app-63 上で以下を実行します。

```console
curl http://127.0.0.1:5000/health
curl http://127.0.0.1:5000/random-5digits
```

`/random-5digits` は以下を返します。

1. `random_digits`: 5 桁の数字文字列（例: `03927`）
2. `response_delay_seconds`: 実際に適用した遅延秒数

`REST_API_RESPONSE_DELAY_SECONDS` が数値でない、または 0 未満 / 10 超の場合は `400` を返します。
