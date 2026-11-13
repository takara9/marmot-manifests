# Vault Go クライアント検証 (75-vault-cluster)

marmot から Go 言語で HashiCorp Vault を使うための検証用コードです。
[75-vault-cluster](../75-vault-cluster/README.md) (3ノード、Raft、TLS、自動 unseal) に対して、
**KV として機密情報を保存・取得する**サンプルプログラムと、marmot で役に立ちそうな Vault の機能を評価するテストを含みます。

## 構成

```
76-vault-go-client/
├── vault-client.yaml          # クライアント VM (vault-client) のマニフェスト
├── playbook/
│   ├── hosts.yaml
│   ├── install.yaml           # roles: golang → vaultcli → gotest
│   └── roles/
│       ├── golang/            # Go のインストール (go_version は defaults で指定)
│       ├── vaultcli/          # vault CLI、CA 証明書の取得、接続先の環境変数
│       └── gotest/            # コードの配置、依存モジュールの取得、go vet、kvdemo のビルド
└── gotest/                    # Go のコード (独立した Go モジュール)
    ├── client.go              # 接続設定、TLS、複数ノードからの接続先選択 (Connect)
    ├── store.go               # KV v2: 保存・取得、排他更新 (CAS)、バージョン、削除、一覧
    ├── transit.go             # Transit: 暗号化、復号、鍵のローテーション、Rewrap
    ├── auth.go                # AppRole ログイン、トークンの自動更新 (WatchToken)
    ├── cmd/kvdemo/main.go     # サンプルプログラム (put / get / list / history / delete / undelete)
    ├── access_test.go         # クラスタの状態、接続先の切り替え、全ノードからの読み書き
    ├── kv_test.go             # KV v2 のテスト
    ├── auth_test.go           # 最小権限、AppRole、トークン更新、レスポンスラッピング
    ├── transit_test.go        # 暗号化サービス
    ├── pki_test.go            # 証明書の発行
    ├── ssh_test.go            # SSH 証明書の署名
    ├── cluster_test.go        # リーダー交代
    └── Makefile
```

## 接続先の設定

環境変数で指定します。名前は Vault CLI と共通です。

| 環境変数 | 内容 | 省略時 |
|---|---|---|
| `VAULT_ADDRS` | ノードのアドレス (カンマ区切り)。先頭から順に試し、最初に unseal 済みで応答したノードに接続する | 75 の 3 ノード (`https://192.168.1.194〜196:8200`) |
| `VAULT_ADDR` | 単一アドレス (`VAULT_ADDRS` が未設定のとき) | |
| `VAULT_CACERT` | CA 証明書のパス | `./certs/ca.crt`、`/etc/vault-client/ca.crt` の順に探す |
| `VAULT_TOKEN` | 認証トークン | `~/.vault-token` (`vault login` が書き込む) |
| `VAULT_EXPECT_NODES` | 期待するノード数 | アドレスの数 |

## クライアント VM の構築

前提として、75-vault-cluster を構築済みであること (CA 証明書をそこから取得します)。

```console
mactl create -f vault-client.yaml
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
```

`vault-client.yaml` の `ansible:` はコメントアウトされています。VM の作成時に playbook も実行したい場合は、コメントを外してください。
75 を後から構築した場合は、上の `ansible-playbook` を再実行すると CA 証明書が取得されます。

構築後のクライアント VM には、次のものが入ります。

- Go (`/usr/local/go`) と、gcc / make / git
- vault CLI (サーバーのサービスは無効化)
- CA 証明書 `/etc/vault-client/ca.crt`
- 環境変数 `VAULT_ADDR`、`VAULT_ADDRS`、`VAULT_CACERT` (`/etc/vault-client/vault-client.sh`、`~/.bashrc` で読み込み)
- コード `~/vault-go-client` と、ビルド済みの `~/vault-go-client/bin/kvdemo`

## 事前準備 (Vault 側)

KV v2 を `secret/` に有効化します。Vault は初期状態では何も有効化されていません。68-vault-single と同じ手順です。

```console
ubuntu@vault-client:~$ vault login                              # 75 の初期化時に表示された Root Token
ubuntu@vault-client:~$ vault secrets enable -path=secret kv-v2
```

`vault login` が `~/.vault-token` に保存したトークンを、Go のコードも読みます。

## サンプルプログラム kvdemo

機密情報を KV に保存し、取り出します。

```console
ubuntu@vault-client:~/vault-go-client$ K=bin/kvdemo
ubuntu@vault-client:~/vault-go-client$ $K put marmot/db-password username=admin password='s3cret'
saved secret/marmot/db-password (version 1)
ubuntu@vault-client:~/vault-go-client$ $K get marmot/db-password
# version 1, created 2026-11-13T06:10:18Z
password=s3cret
username=admin
ubuntu@vault-client:~/vault-go-client$ $K get marmot/db-password password      # 1 フィールドだけ (スクリプト向け)
s3cret
ubuntu@vault-client:~/vault-go-client$ $K put marmot/db-password username=admin password='rotated'
ubuntu@vault-client:~/vault-go-client$ $K history marmot/db-password            # バージョン一覧
ubuntu@vault-client:~/vault-go-client$ $K list marmot
ubuntu@vault-client:~/vault-go-client$ $K delete marmot/db-password             # 論理削除
ubuntu@vault-client:~/vault-go-client$ $K undelete marmot/db-password           # 復元
```

`-mount` で KV のマウントパスを変えられます (既定は `secret`)。ライブラリとして使う場合は、次のようにします。

```go
cfg, _ := vaultclient.LoadConfig()
c, _ := vaultclient.Connect(ctx, cfg)          // 動いているノードに接続
store := vaultclient.NewStore(c, "secret")
store.Put(ctx, "marmot/db-password", map[string]any{"password": "s3cret"})
sec, _ := store.Get(ctx, "marmot/db-password") // sec.Data["password"]
```

## テストの実行

テストは、**一時的なマウントや認証方式を作成して終了時に削除**します (パスは `vct-<実行ID>-<連番>`)。
既存の `secret/` などには触れません。ただし、マウントの作成にはルート権限相当のトークンが必要です。

### クライアント VM 上

```console
ubuntu@vault-client:~$ vault login                     # Root Token
ubuntu@vault-client:~$ cd vault-go-client
ubuntu@vault-client:~/vault-go-client$ make test
ubuntu@vault-client:~/vault-go-client$ make test-short       # トークン更新 (約 15 秒) を省略
ubuntu@vault-client:~/vault-go-client$ make test-stepdown    # リーダー交代のテストも実行
```

### このリポジトリを置いた別のマシン上

```console
cd gotest
make fetch-ca                          # 75 のサーバーから CA 証明書を certs/ca.crt に取得 (ssh が必要)
export VAULT_TOKEN=hvs.xxxxxxxx
make test
```

特定のテストだけ実行するには、次のようにします。

```console
go test -count=1 -v -run TestKVConcurrentUpdateWithCAS ./...
```

## テスト内容

### クラスタ

| テスト | 確認すること |
|---|---|
| TestClusterHealth | 全ノードが初期化済み・unseal 済み、アクティブが 1 つ、同じクラスタ、Raft の全メンバーが投票権付きで参加していること |
| TestEndpointFailover | 先頭のノードが落ちていても、次のノードに切り替えて接続できること。全ノードが落ちていれば `ErrNoHealthyNode` |
| TestReadWriteViaEveryNode | 1 つのノードに書いた値が全ノードから読め、スタンバイ経由の書き込みもアクティブに届くこと |
| TestLeaderStepDown | アクティブノードがステップダウンしても、別のノードが引き継ぎ、読み書きを継続できること (`VAULT_TEST_STEPDOWN=1` が必要) |

### KV v2 (機密情報の保存)

| テスト | 確認すること |
|---|---|
| TestKVPutGet | 複数行の秘密鍵 (PEM)、日本語、ネストした値、100KB の値を保存して取り出せること |
| TestKVVersioning | 書き込みごとにバージョンが増え、古いバージョンを読めること。保持数を超えた古いバージョンは破棄されること |
| TestKVStaleCASRejected | 古いバージョン番号での更新 (CAS) が拒否されること |
| TestKVCreateExclusive | 8 クライアントが同じキーを作成しても、成功するのは 1 つだけであること |
| TestKVConcurrentUpdateWithCAS | 5 クライアントが同時に加算しても、更新が失われないこと (`Store.Update`) |
| TestKVLostUpdateWithoutCAS | 比較用。CAS なしの読み書きでは更新が失われることを観察する (失敗にはしない) |
| TestKVDeleteUndeleteDestroy | 論理削除、復元、完全破棄、全削除。削除後でも `Update` でバージョンが続くこと |
| TestKVList | 階層のあるキーを一覧できること (下位階層は末尾に `/`) |

### 認証・権限

| テスト | 確認すること |
|---|---|
| TestPolicyLeastPrivilege | サービスごとに、必要なパスだけを読めるトークンを発行できること。他のパスの読み取りと書き込みは 403 |
| TestAppRoleLogin | 固定のトークンを持たず、`role_id` と使い捨ての `secret_id` で認証できること。2 回目の使用と誤った `role_id` は拒否される |
| TestTokenRenewWithWatcher | `WatchToken` がトークンを更新し続け、最大 TTL に達したら `ErrTokenExpired` で再ログインを促すこと |
| TestResponseWrapping | ラップされた機密情報は、トークンを持たない受け取り側が 1 回だけ取り出せること。VM の起動時に機密情報を渡す用途 |

### 暗号化・証明書

| テスト | 確認すること |
|---|---|
| TestTransitEncryptDecrypt | 暗号化・復号、鍵のローテーション、平文を取り出さない Rewrap、古い鍵バージョンの無効化 |
| TestTransitEncryptOnlyToken | 暗号化だけを許可したトークンでは、復号できないこと |
| TestPKIIssueCertificate | CA として短命のサーバー証明書を発行できること。CA で検証でき、SAN と有効期限が指定どおりで、許可していないドメインは発行できない。失効もできる |
| TestSSHSignClientKey | SSH 公開鍵に CA で署名した短命のユーザー証明書を得られること。sshd と同じ検証が通り、許可していないユーザー名は署名できない |

## marmot での使いどころ

| 機能 | 用途の例 |
|---|---|
| KV v2 + `Store.Update` | VM のパスワードや API キーの保存。CAS で同時更新が競合しても失われない。バージョン管理で誤更新から戻せる |
| `Connect` | 3 ノードのうち動いているノードへ接続する。75 には仮想 IP やロードバランサーがないため、クライアント側で切り替える |
| AppRole + `WatchToken` | marmot のサービスが、固定のトークンを持たずに認証し、トークンを自動更新する |
| ポリシー | サービスごとに読めるパスを限定する (最小権限) |
| Transit | etcd に保存する値を暗号化する。鍵は Vault の外に出ない。暗号化だけのサービスも作れる |
| PKI | 内部サービスの TLS 証明書を短命で発行する (59-ca-and-dns、62-ca-acme の代替) |
| SSH 署名 | サーバーごとの `authorized_keys` を管理せず、短命の証明書で VM にログインする |
| レスポンスラッピング | VM の起動時に、使い捨てのトークンだけを渡して、機密情報を安全に届ける |

## 注意事項

- **トークン**: テストには、ルート権限相当のトークンが必要です。`VAULT_TOKEN` をシェルの履歴に残さないよう、`vault login` で `~/.vault-token` に保存する方法を推奨します。
- **リーダー交代**: `TestLeaderStepDown` は、アクティブノードをステップダウンさせるため、数秒間リクエストが失敗します。75 を他の用途で使っているときは実行しないでください。
- **削除済みシークレット**: `vault/api` の `KVv2.Get` は、論理削除したシークレットでもエラーにせず、`Data` が nil のシークレットを返します。`Store.Get` は、これを `ErrNotFound` にそろえています。
- **Create の制限**: `Store.Create` は、論理削除したキーに対しても `ErrAlreadyExists` を返します。バージョン番号が残っているためです。再作成するときは `Update` を使ってください。
- **自動 unseal の検証**: ノードの再起動で unseal されることは、このテストの範囲外です。75 の README の手順で、ノードを再起動して確認してください。
- **接続先**: アドレスは `gotest/client.go` の `DefaultAddrs` と `playbook/roles/vaultcli/defaults/main.yaml` に書かれています。変更する場合は、環境変数 `VAULT_ADDRS` を使うか、両方を修正してください。
- **TLS**: サーバー認証のみで、クライアント証明書は使いません。75 の CA を作り直した場合は、`make fetch-ca` で証明書を再取得してください。
- **接続方法**: クライアント VM への接続は `mactl-ssh`、CA 証明書の取得元 (75 の先頭ノード) へは IP アドレスへの直接 SSH です。
- **所要時間**: 全テストで 30〜40 秒です。トークン更新のテストが約 13 秒、SSH 署名のテストが数秒かかります。
- **ライブラリ**: Go のクライアントライブラリは `github.com/hashicorp/vault/api` v1.23.0、サーバーは Vault 2.2.2 です。
