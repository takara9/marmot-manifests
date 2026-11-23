# 仮想マシンをWeb(nginx)サーバーにする

仮想サーバーの起動後に、AnsibleでNginxを構成して、Webサーバーにすることができます。
Ansibleプレイブックを編集することで、Webページを作り変えるなど、発展させることできます。

## 起動方法

```console
$ mactl create -f server-90.yaml
```

## 起動の確認

上から、リスト、サーバー名を指定して個別表示、ラベルを指定して選別表示ができます。

```console
$ mactl get server 
$ mactl get server server-90
$ mactl get server -l app=webserver
```


## 確認方法

`curl`コマンドを実行することで、作成したWebページを見ることができます。

```
$ curl http://SERVER-IP-ADDRESS/
```

DNSが、marmotサーバーに向いていれば、以下のURLアドレスでアクセスできます。

- [http://server-90.host-bridge/](http://server-90.host-bridge/)


## TLS 証明書

nginx は HTTPS（443）で待ち受け、HTTP（80）は HTTPS へリダイレクトします。証明書は、指定の有無で次のように切り替わります。

| 指定 | 使用する証明書 |
|---|---|
| なし（既定） | サーバー上で**自己署名証明書**を作成して使用する |
| `nginx_tls_cert_src` と `nginx_tls_key_src` の両方 | **CA が発行した証明書**を使用する |

### 自己署名証明書（既定）

何も指定せずに Playbook を実行します。サーバー上で RSA 2048 の鍵と自己署名証明書（有効期間 825 日）を作成します。SAN は `nginx_server_name`（既定 `web1.labo.local`）とサーバーの既定 IPv4 アドレスです。追加するには `nginx_tls_extra_san` に `["DNS:www.labo.local"]` のように指定します。

自己署名証明書は、ブラウザや `curl` が信頼しません。動作確認では次のようにします。

```console
$ curl -k https://web1.labo.local/
```

証明書を信頼させる場合は、サーバーの `/etc/nginx/ssl/web1.labo.local.crt` を取得して、クライアントの信頼する証明書として登録してください。

### CA が発行した証明書

CA で発行した証明書（中間 CA がある場合は連結したフルチェーン）と、パスフレーズなしの秘密鍵を、Ansible を実行するマシンに用意します。[playbook/install.yaml](playbook/install.yaml) の `vars` に、両方のパスを指定します。

```yaml
    nginx_tls_cert_src: web1.labo.local.crt
    nginx_tls_key_src: web1.labo.local.key
```

相対パスは `playbook/roles/web-nginx/files/` から探します。この場所の `*.crt` と `*.key` は Git に含まれません。絶対パスも指定できます。実行時に指定する場合は、次のようにします。

```console
$ ansible-playbook -i playbook/hosts.yaml playbook/install.yaml \
    -e nginx_tls_cert_src=/path/to/web1.labo.local.crt \
    -e nginx_tls_key_src=/path/to/web1.labo.local.key
```

- 証明書と鍵は、サーバーの `/etc/nginx/ssl/` に配置されます。秘密鍵は root だけが読めます（0600）。
- 片方だけを指定すると、Playbook は証明書を変更せずに停止します。
- 証明書と鍵が一致しない場合は、`nginx -t` で停止します。起動中の nginx には反映せず、それまでの証明書で応答を続けます。
- 指定を外して再実行すると、自己署名証明書に切り替わります。

### 確認

Playbook は最後に、HTTP の 301 リダイレクトと HTTPS の応答を確認し、**nginx が実際に配信する証明書と設定した証明書が同じこと**を確認します。手動で確認する場合は、次のようにします。

```console
$ openssl s_client -connect web1.labo.local:443 -servername web1.labo.local </dev/null 2>/dev/null \
    | openssl x509 -noout -subject -issuer -dates
```

CA が発行した証明書をクライアントが検証するには、その CA の証明書を信頼させます。

```console
$ curl --cacert ca.crt https://web1.labo.local/
```

## 削除

構成に使用したファイルを指定して削除、または、サーバー名を指定して削除ができます。

```console
$ mactl del -f server-90.yaml
$ mactl del server server-90
```

## エラーになった時の原因調査

テキスト形式で見やすい形式で、詳細表示します。

```console
$ mactl describe server server-90
```

JSON形式で出力して、`jq`コマンドで、エラーメッセージだけを表示できます。

```console
$ mactl get server server-90 -o json |jq .[].status
```


