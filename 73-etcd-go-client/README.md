# etcd Go クライアント検証 (71 / 72 の切り替え)

marmot の Go 言語による etcd アクセス (`marmot/pkg/db`) を改善するための検証用コードです。
Go と etcdctl をインストールしたクライアント VM から、[71-etcd-single](../71-etcd-single/README.md) (単一ノード) と
[72-etcd-cluster](../72-etcd-cluster/README.md) (3ノード) のどちらにも、同じテストを実行できます。

## 構成

```
73-etcd-go-client/
├── etcd-single.yaml        # クライアント VM (etcd-client) のマニフェスト
├── playbook/
│   ├── hosts.yaml
│   ├── install.yaml        # roles: golang → etcdctl → gotest
│   └── roles/
│       ├── golang/         # Go のインストール (go_version は defaults で指定)
│       ├── etcdctl/        # etcdctl、CA 証明書の取得、接続先の切り替え (etcd-use)
│       └── gotest/         # テストコードの配置、依存モジュールの取得、go vet
└── gotest/                 # Go のテストコード (独立した Go モジュール)
    ├── client.go           # 接続先の切り替え、TLS・複数エンドポイントのクライアント作成
    ├── store.go            # 排他更新 (CAS)、分散ロック、セッション再生成
    ├── watch.go            # 更新監視 (再接続、取りこぼし防止、compaction 対応)
    ├── access_test.go      # 接続、排他更新、分散ロックのテスト
    ├── watch_test.go       # 更新監視、ジョブ処理、リース、リーダー選出のテスト
    └── Makefile
```

## 接続先の切り替え

環境変数 `ETCD_TARGET` で切り替えます。省略時は `single` です。

| ETCD_TARGET | 接続先 | エンドポイント | メンバー数 |
|---|---|---|---|
| single | 71-etcd-single | https://192.168.1.190:2379 | 1 |
| cluster | 72-etcd-cluster | https://192.168.1.191〜193:2379 | 3 |

次の環境変数で上書きもできます。

| 環境変数 | 内容 |
|---|---|
| `ETCD_ENDPOINTS` | エンドポイント (カンマ区切り) |
| `ETCD_CACERT` | CA 証明書のパス。省略時は `./certs/<target>-ca.crt`、`/etc/etcd-client/<target>-ca.crt` の順に探す |
| `ETCD_EXPECT_MEMBERS` | 期待するメンバー数 |

## クライアント VM の構築

前提として、71 と 72 を構築済みであること (CA 証明書をそこから取得します)。未構築の接続先は飛ばされ、警告が出ます。

```console
mactl create -f etcd-single.yaml
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
```

`etcd-single.yaml` の `ansible:` が有効なため、VM の作成時にも playbook が実行されます。この場合、71 と 72 を先に構築しておいてください。
後から構築した場合は、上の `ansible-playbook` を再実行すると CA 証明書が取得されます。

構築後のクライアント VM には、次のものが入ります。

- Go (`/usr/local/go`) と、gcc / make / git
- etcdctl (`etcd-client` パッケージ、3.4)
- CA 証明書 `/etc/etcd-client/single-ca.crt`、`cluster-ca.crt`
- テストコード `~/etcd-go-client`
- 接続先を切り替えるシェル関数 `etcd-use` (`~/.bashrc` で読み込み。ログイン時は single)

## テストの実行

### クライアント VM 上

```console
ubuntu@etcd-client:~$ etcd-use cluster              # etcdctl と Go テストの接続先を切り替え
ubuntu@etcd-client:~$ etcdctl endpoint health --cluster
ubuntu@etcd-client:~$ cd etcd-go-client
ubuntu@etcd-client:~/etcd-go-client$ make test          # ETCD_TARGET は etcd-use で設定済み
ubuntu@etcd-client:~/etcd-go-client$ make test TARGET=single
```

### このリポジトリを置いた別のマシン上

```console
cd gotest
make fetch-ca                 # 71/72 のサーバーから CA 証明書を certs/ に取得 (ssh が必要)
make test TARGET=single
make test TARGET=cluster
make test-all                 # 両方
make test-short TARGET=...    # 時間のかかるテスト (リース失効) を省略
make test-compact TARGET=...  # compaction のテストも実行
```

特定のテストだけ実行するには、次のようにします。

```console
ETCD_TARGET=cluster go test -count=1 -v -run TestConcurrentUpdateWithCAS ./...
```

## テスト内容

| テスト | 確認すること |
|---|---|
| TestClusterMembers | メンバー数 (single は 1、cluster は 3)、全エンドポイントが同じリーダーを返すこと |
| TestReadFromEveryEndpoint | 1 つのエンドポイントに書いた値が、全エンドポイントから読めること |
| TestEndpointFailover | 先頭のエンドポイントが落ちていても処理を継続できること |
| TestTxnMultiKey | 複数キーのトランザクションが原子的であること |
| TestStaleCASRejected | 古い ModRevision での更新が拒否されること (`PutJSONCAS` 相当) |
| TestCreateIfAbsentExclusive | 複数クライアントが同じキーを作成しても、成功するのは 1 つだけであること |
| TestConcurrentUpdateWithCAS | 5 クライアントの同時加算で、更新が失われないこと |
| TestLostUpdateWithoutCAS | 比較用。CAS なしの読み書きでは更新が失われることを観察する (失敗にはしない) |
| TestDistributedMutexExclusion | 分散ロックで、同時にクリティカルセクションに入るのが 1 クライアントだけであること |
| TestLockTimeout | ロック保持中は他のクライアントが待たされ、期限で諦められること |
| TestLockReleasedWhenHolderDies | ロック保持者が異常終了 (リース失効) すると、待機中のクライアントが取得できること |
| TestSessionRecreatedAfterLeaseLoss | リースが失効しても、Session を作り直してロックを取得できること |
| TestWatchPrefixEvents | prefix 配下の PUT / DELETE が、順序どおりに通知されること |
| TestWatchResumeFromRevision | 監視が途切れた間の更新を、リビジョン指定で取りこぼさず再開できること |
| TestWatchJobQueueExclusiveClaim | 3 ワーカーが同じ prefix を監視し、各ジョブを必ず 1 つのワーカーだけが処理すること |
| TestLeaseExpiryAndKeepAlive | KeepAlive 中はリース付きキーが残り、止めると TTL で消えて監視で検知できること |
| TestLeaderElection | リーダーが常に 1 つで、Resign で順に交代すること |
| TestWatchRecoversFromCompaction | compaction で履歴が失われても、一覧を取り直して監視を継続できること (`ETCD_TEST_COMPACT=1` が必要) |

## marmot への反映候補

`gotest/store.go` と `gotest/watch.go` は、marmot の `pkg/db/db-etcd.go` と比べて次の点を改善した試作です。

- **TLS と複数エンドポイント**: `NewDatabase(url)` は単一 URL で TLS がありません。`NewClient` は全エンドポイントを渡し、keepalive で停止ノードを早く検出して切り替えます。
- **排他更新の再試行**: `PutJSONCAS` は競合すると `ErrUpdateConflict` を返すだけです。`UpdateJSON` は、再読込、変更、CAS 書き込みを競合しなくなるまで繰り返します。
- **新規作成の排他**: `CreateJSON` は、キーが存在しない場合のみ作成します。
- **分散ロック**: `Database.LockKey` は、取得した Mutex を共有フィールド `d.Mutex` にも書き込みます。同じ `Database` から並行して呼ぶと、フィールドが互いに上書きされます。`Store.Lock` は呼び出しごとに Mutex を返します。
- **Session の再生成**: 失効した Session は作り直します。`ensureLiveSession` と同じ狙いで、リース失効後のロック取得をテストしています。
- **更新監視**: `WatchPrefix` は、一覧取得、監視開始、切断時の再接続 (最後のリビジョンの次から再開)、compaction 時の再同期までを行います。

## 注意事項

- **テストが使うキー**: `/etcd-go-client-test/<実行ID>/<テスト名>/` 配下だけを使い、終了時に削除します。`/marmot` 配下には触れません。
- **compaction のテスト**: etcd 全体の履歴を compaction するため、`ETCD_TEST_COMPACT=1` を指定したときだけ実行します。marmot が使っている etcd には実行しないでください。
- **etcd のバージョン**: サーバーは 3.4.30 (Ubuntu 24.04 のパッケージ)、Go のクライアントライブラリは marmot と同じ `go.etcd.io/etcd/client/v3` v3.6.12 です。
- **接続方法**: クライアント VM への接続は `mactl-ssh`、CA 証明書の取得元 (71/72 のサーバー) へは IP アドレスへの直接 SSH です。
- **TLS**: サーバー認証のみで、クライアント証明書は使いません (71/72 の設定どおり)。CA 証明書は、取得後に 71/72 の CA を作り直した場合は再取得してください。
- **IP アドレスの固定**: エンドポイントは `gotest/client.go` と `playbook/roles/etcdctl/defaults/main.yaml` に書かれています。アドレスを変える場合は、環境変数 `ETCD_ENDPOINTS` を使うか、両方を修正してください。
- **所要時間**: 全テストで 10〜20 秒です。リース失効のテストが約 8 秒かかります。
