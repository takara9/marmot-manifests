# aws s3 コマンドの使い方

[README.md](README.md) で構築した Garage（S3互換ストレージ）を、AWS CLI の `aws s3` コマンドで操作する方法をまとめます。Garage 以外の S3 互換ストレージや AWS S3 でも、接続設定を除いて同じ操作です。

## 1. 準備

### AWS CLI のインストール（クライアント側）

```bash
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o awscliv2.zip
unzip awscliv2.zip
sudo ./aws/install
aws --version
```

### 接続設定

Garage のアクセスキーは、サーバー上で確認できます。

```bash
ssh ubuntu@garage.labo.local
sudo garage key info app-key --show-secret
```

クライアントでは環境変数で指定します。

```bash
export AWS_ACCESS_KEY_ID=<Key ID>
export AWS_SECRET_ACCESS_KEY=<Secret key>
export AWS_DEFAULT_REGION=garage
export AWS_ENDPOINT_URL=https://s3.labo.local
export AWS_CA_BUNDLE=/path/to/root_ca.crt   # 自己署名のときはサーバー証明書。OS に登録済みなら不要
```

| 変数 | 説明 |
|---|---|
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | Garage のアクセスキー |
| `AWS_DEFAULT_REGION` | Garage の `s3_region`（既定 `garage`）。AWS の `us-east-1` 等ではなく、この値に合わせる |
| `AWS_ENDPOINT_URL` | S3 API のエンドポイント。AWS CLI v2.13 以降は全サービス共通で使える。省略すると AWS 本体に接続してしまう |
| `AWS_CA_BUNDLE` | 自己署名証明書や社内CAを信頼させる CA 証明書（PEM） |

毎回環境変数を設定しない場合は、プロファイルを作ります。

```bash
aws configure --profile garage
aws configure set endpoint_url https://s3.labo.local --profile garage
aws configure set ca_bundle /path/to/root_ca.crt --profile garage
aws configure set s3.addressing_style path --profile garage

export AWS_PROFILE=garage     # 以降のコマンドで有効（または --profile garage を毎回付ける）
```

> 古い AWS CLI（v2.13 未満）では `endpoint_url` が使えません。その場合は各コマンドに `--endpoint-url https://s3.labo.local` を付けます。

### アドレス方式

S3 のURL形式には2種類あります。

| 方式 | 例 | 備考 |
|---|---|---|
| パス形式 | `https://s3.labo.local/demo/file.txt` | 追加のDNS設定が不要。**迷ったらこちら** |
| 仮想ホスト形式 | `https://demo.s3.labo.local/file.txt` | `*.s3.labo.local` のDNS解決が必要。Garage の `root_domain` と証明書の SAN は対応済み |

上のプロファイル例のように `s3.addressing_style path` を指定すると、パス形式に固定できます。

## 2. コマンドの全体像

`aws s3` は、よく使う操作を簡略化した高レベルコマンドです。

```text
aws s3 <サブコマンド> [対象] [オプション]
```

| サブコマンド | 用途 |
|---|---|
| `ls` | バケット・オブジェクトの一覧 |
| `mb` / `rb` | バケットの作成 / 削除 |
| `cp` | コピー（アップロード・ダウンロード・バケット間） |
| `mv` | 移動（コピー後に元を削除） |
| `rm` | オブジェクトの削除 |
| `sync` | ディレクトリとバケットの同期 |
| `presign` | 期限付きURLの発行 |
| `website` | 静的ウェブサイトの設定 |

S3 のパスは `s3://<バケット名>/<キー>` の形式で指定します。S3 に本当のディレクトリはなく、`/` を含むキー名（プレフィックス）で擬似的に階層を表現しています。

## 3. 一覧表示（ls）

```bash
aws s3 ls                          # バケット一覧
aws s3 ls s3://demo/               # バケット直下の一覧（プレフィックスは PRE と表示）
aws s3 ls s3://demo/logs/          # プレフィックス配下
aws s3 ls s3://demo/ --recursive   # 配下をすべて再帰的に
aws s3 ls s3://demo/ --recursive --human-readable --summarize   # サイズを読みやすく、合計も表示
```

> プレフィックスの末尾の `/` を付け忘れると、`logs` で始まるキーすべてが対象になります。

## 4. バケットの作成と削除（mb / rb）

```bash
aws s3 mb s3://work                # 作成
aws s3 rb s3://work                # 空のバケットを削除
aws s3 rb s3://work --force        # 中のオブジェクトを全削除してからバケットを削除
```

> Garage では、アクセスキーに `create bucket` 権限がないと `mb` が次のエラーで失敗します（Playbook が作るキーには、この権限は付いていません）。
>
> ```text
> make_bucket failed: s3://work An error occurred (AccessDenied) when calling the CreateBucket operation: Forbidden: Access key GKxxxx is not allowed to create buckets
> ```
>
> `mb` を使いたい場合は、サーバー上でキーにバケット作成権限を付与します。
>
> ```bash
> sudo garage key allow --create-bucket app-key
> sudo garage key info app-key        # Permissions の確認
> ```
>
> `mb` で作ったバケットには、実行したキーに read / write / owner がすべて自動で付きます。ただし、バケット名は**そのキー専用のローカルエイリアス**として登録されます（[12. トラブルシューティング](#12-トラブルシューティング)参照）。別のキーにも見せたい場合は、グローバルエイリアスを追加して `garage bucket allow` で許可します。
>
> 権限を付与せずに、サーバー上でバケットを作る方法もあります。
>
> ```bash
> sudo garage bucket create work
> sudo garage bucket allow --read --write --owner work --key app-key
> sudo garage bucket list
> ```

## 5. コピー（cp）

### アップロード

```bash
aws s3 cp ./file.txt s3://demo/               # ファイル名はそのまま
aws s3 cp ./file.txt s3://demo/docs/a.txt     # キー名を変えて保存
aws s3 cp ./dir s3://demo/dir/ --recursive    # ディレクトリごと
```

### ダウンロード

```bash
aws s3 cp s3://demo/docs/a.txt ./a.txt
aws s3 cp s3://demo/dir/ ./dir --recursive
```

### バケット間のコピー

```bash
aws s3 cp s3://demo/docs/a.txt s3://work/docs/a.txt
```

### 標準入出力との連携

`-` を指定するとパイプで接続できます。

```bash
tar czf - /etc/ssl | aws s3 cp - s3://demo/backup/ssl.tar.gz     # 標準入力からアップロード
aws s3 cp s3://demo/backup/ssl.tar.gz - | tar tzf -              # 標準出力へダウンロード
```

### 絞り込み（--exclude / --include）

`--recursive` や `sync` では、フィルタを**指定した順に**評価します。後に書いたものが優先されます。全件が対象になるので、まず除外してから含めます。

```bash
# .log ファイルだけをアップロード
aws s3 cp ./logs s3://demo/logs/ --recursive --exclude "*" --include "*.log"

# .tmp 以外をアップロード
aws s3 cp ./work s3://demo/work/ --recursive --exclude "*.tmp"
```

### よく使うオプション

| オプション | 説明 |
|---|---|
| `--recursive` | ディレクトリ・プレフィックス配下を再帰的に処理 |
| `--exclude` / `--include` | 対象の絞り込み（シェルに展開されないよう引用符で囲む） |
| `--dryrun` | 実行せず、何が行われるかだけ表示 |
| `--content-type` | Content-Type を指定（例: `text/html`） |
| `--metadata key=value` | ユーザー定義メタデータを付与 |
| `--no-progress` | 進捗表示を抑止（スクリプトやログ向け） |
| `--only-show-errors` | エラー以外を表示しない |
| `--expected-size` | 標準入力から 50GB 超を送るときに、サイズ（バイト）を指定 |

> `--acl` や `--storage-class` は AWS 向けの指定です。Garage は ACL とストレージクラスを使わないため、付けても効果はありません。アクセス制御は Garage のキー権限で行います。

## 6. 移動と削除（mv / rm）

```bash
aws s3 mv ./file.txt s3://demo/                    # ローカル → S3（ローカルのファイルは削除される）
aws s3 mv s3://demo/a.txt s3://demo/archive/a.txt  # S3 内での名前変更・移動
aws s3 mv s3://demo/dir/ s3://work/dir/ --recursive

aws s3 rm s3://demo/a.txt                          # 1件削除
aws s3 rm s3://demo/logs/ --recursive              # プレフィックス配下を削除
aws s3 rm s3://demo/ --recursive --exclude "*" --include "*.tmp"   # 条件付きで削除
```

> **`rm` は元に戻せません。** バージョニングを使っていなければ即時に消えます。`--recursive` を使う前に、同じ引数で `--dryrun` を実行して対象を確認してください。

```bash
aws s3 rm s3://demo/logs/ --recursive --dryrun
```

## 7. 同期（sync）

2つの場所の差分だけをコピーします。サイズまたは更新時刻が異なるファイルが対象で、バックアップや静的サイトの公開に向いています。

```bash
aws s3 sync ./site s3://demo/site/          # ローカル → S3
aws s3 sync s3://demo/site/ ./site-copy     # S3 → ローカル
aws s3 sync s3://demo/ s3://work/           # バケット間
```

| オプション | 説明 |
|---|---|
| `--delete` | コピー先にあって、コピー元にないファイルを**削除**し、完全に一致させる |
| `--exclude` / `--include` | 対象の絞り込み（`cp` と同じ規則） |
| `--size-only` | 更新時刻を無視し、サイズだけで差分判定 |
| `--exact-timestamps` | S3 → ローカルで、更新時刻が完全一致しないものを対象にする |
| `--dryrun` | 実行内容の確認のみ |

```bash
# まず確認してから --delete 付きで実行
aws s3 sync ./site s3://demo/site/ --delete --dryrun
aws s3 sync ./site s3://demo/site/ --delete
```

> `sync` は一方向です。双方向の同期はできません。また、空のディレクトリは同期されません。

## 8. 期限付きURL（presign）

アクセスキーを渡さずに、オブジェクトを一時的に共有できます。有効期限の既定は3600秒（1時間）、最大は7日（604800秒）です。

```bash
aws s3 presign s3://demo/docs/a.txt --expires-in 600
```

出力されたURLは、`curl` やブラウザでアクセスできます。

```bash
curl -O "$(aws s3 presign s3://demo/docs/a.txt --expires-in 600)"
```

> 署名にはリージョンが含まれます。`AWS_DEFAULT_REGION=garage` を設定していないとURLが無効になります。パス形式で署名するため、`s3.addressing_style path` を推奨します。

## 9. 大きなファイルの転送

8MB を超えるファイルは自動的にマルチパートアップロードになります。調整は `aws configure set` で行います。

```bash
aws configure set s3.multipart_threshold 64MB --profile garage
aws configure set s3.multipart_chunksize 16MB --profile garage
aws configure set s3.max_concurrent_requests 20 --profile garage
```

| 設定 | 既定値 | 説明 |
|---|---|---|
| `max_concurrent_requests` | `10` | 同時リクエスト数 |
| `multipart_threshold` | `8MB` | マルチパートに切り替えるサイズ |
| `multipart_chunksize` | `8MB` | 1パートあたりのサイズ |
| `max_bandwidth` | 無制限 | 帯域の上限（例: `50MB/s`） |

途中で中断したアップロードは未完了のパートとして残り、容量を消費することがあります。その場合は低レベルコマンドの `aws s3api` で確認して削除します。

```bash
aws s3api list-multipart-uploads --bucket demo
aws s3api abort-multipart-upload --bucket demo --key <Key> --upload-id <UploadId>
```

## 10. 高レベルコマンドで足りないとき（aws s3api）

`aws s3` にない操作は `aws s3api` を使います。S3 API をそのまま呼び出す低レベルコマンドです。

```bash
aws s3api list-buckets
aws s3api head-object --bucket demo --key docs/a.txt           # サイズ・ETag・更新時刻・メタデータ
aws s3api list-objects-v2 --bucket demo --prefix docs/ --max-items 10
aws s3api get-bucket-location --bucket demo
```

`--query`（JMESPath）と `--output` で出力を加工できます。

```bash
aws s3api list-objects-v2 --bucket demo --query 'Contents[].[Key,Size]' --output text
```

> Garage が対応している S3 API は一部です（バケットポリシーやACL、バージョニング等は未対応または一部のみ）。対応状況は [Garage のドキュメント](https://garagehq.deuxfleurs.fr/documentation/reference-manual/s3-compatibility/) を確認してください。

## 11. 動作確認の一連の流れ

```bash
echo "hello garage" > hello.txt

aws s3 ls                                        # 接続とバケットの確認
aws s3 cp hello.txt s3://demo/test/hello.txt     # アップロード
aws s3 ls s3://demo/test/                        # 一覧に表示される
aws s3 cp s3://demo/test/hello.txt - | cat       # 内容を確認
aws s3 rm s3://demo/test/hello.txt               # 後始末
```

## 12. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `SSL validation failed` / `certificate verify failed` | 自己署名証明書や社内CAを信頼していない。`AWS_CA_BUNDLE` を設定する。検証目的に限り `--no-verify-ssl` も使える（通常は使わない） |
| `Could not connect to the endpoint URL` | エンドポイントやDNSが誤っている。`AWS_ENDPOINT_URL` と `s3.labo.local` の名前解決を確認する |
| `InvalidAccessKeyId` / `SignatureDoesNotMatch` | キーが誤っている。`garage key info app-key --show-secret` で再確認する。AWS 本体に接続している場合は、エンドポイントが未指定 |
| `AccessDenied` | キーにバケットの権限がない。`garage bucket allow --read --write --owner <バケット> --key <キー名>` で付与する |
| `NoSuchBucket` | バケット名が誤っている、またはリージョンが `garage` になっていない。`aws s3 ls` で確認する |
| `mb` が `not allowed to create buckets` で失敗する | キーに `create bucket` 権限がない。`sudo garage key allow --create-bucket <キー名>` で付与するか、サーバー上の `garage bucket create` で作成する |
| 仮想ホスト形式でつながらない | `<バケット>.s3.labo.local` がDNSで解決できない。パス形式（`s3.addressing_style path`）を使う |

詳細なやり取りは `--debug` を付けると確認できます。

```bash
aws s3 ls s3://demo/ --debug 2>&1 | less
```

### 事例: `aws s3 mb` が AccessDenied になる

```console
$ aws s3 mb s3://work
make_bucket failed: s3://work An error occurred (AccessDenied) when calling the CreateBucket operation: Forbidden: Access key GKe9be7eee24e474c8bfd19971 is not allowed to create buckets
```

**原因**: 接続・認証は成功していますが、使用したアクセスキー（ここでは `app-key`）にバケット作成権限（create bucket）がありません。Playbook が作るキーには、バケット単位の権限（read / write / owner）だけが付与され、バケット作成権限は付きません。

**対処1: キーにバケット作成権限を付与する**（クライアントから `mb` を使いたい場合）

```bash
ssh ubuntu@garage.labo.local
sudo garage key allow --create-bucket app-key
sudo garage key info app-key      # Permissions に create bucket が付いたことを確認
```

付与後、クライアントで `aws s3 mb s3://work` を再実行します。

**対処2: サーバー上でバケットを作成する**（キーの権限を増やしたくない場合）

```bash
sudo garage bucket create work
sudo garage bucket allow --read --write --owner work --key app-key
sudo garage bucket list
```

クライアントから `aws s3 ls` で `work` が見えれば完了です。

### 事例: `aws s3 mb` で作ったバケットが Garage 側の一覧に表示されない

`aws s3 mb s3://work` は成功し、`aws s3 ls` でも `work` が見えるのに、Garage 側（garage-ui や `garage bucket list` の名前欄、別のキーの `aws s3 ls`）で見つからないことがあります。

**原因**: Garage は、S3 API の `CreateBucket` で作ったバケットの名前を、**グローバルエイリアスではなく、作成したキー専用のローカルエイリアス**として登録します。

| 種類 | 名前が見える範囲 | 作成方法 |
|---|---|---|
| グローバルエイリアス | 全キー共通（Garage 全体で一意） | `garage bucket create`、Playbook の `garage_buckets` |
| ローカルエイリアス | 作成したキーだけ（キーごとに名前空間が別） | S3 API の `CreateBucket`（`aws s3 mb`） |

このため、次のような現象になります。

- `mb` を実行したキー（`app-key`）の `aws s3 ls` には表示される。
- 別のキーの `aws s3 ls` には表示されず、バケット名を指定してもアクセスできない（権限も付いていない）。
- グローバルエイリアスだけを表示する画面（garage-ui など）には名前が出ない。
- 仮想ホスト形式（`work.s3.labo.local`）や Web 公開（`s3_web`）は、グローバルエイリアスでないと使えない。

**確認方法**（サーバー上）: ローカルエイリアスは `<キーID>:<バケット名>` の形式で表示されます。

```bash
sudo garage bucket list                 # Local aliases 欄に GKxxxx:work のように出る
sudo garage bucket info <バケットID>    # Key permissions や Aliases の確認
```

**対処1: グローバルエイリアスを追加する**（全キー共通の名前にしたい場合）

```bash
sudo garage bucket list                       # バケットID（先頭の英数字）を確認
sudo garage bucket alias <バケットID> work    # グローバルエイリアスを追加
sudo garage bucket allow --read --write --owner work --key <他のキー名>   # 他のキーにも許可する場合
```

**対処2: 最初からグローバルエイリアスで作る**（運用ではこちらを推奨）

```bash
sudo garage bucket create work
sudo garage bucket allow --read --write --owner work --key app-key
```

> バケットを削除するときは、`aws s3 rb` がローカルエイリアスの削除として動作します。他にエイリアスが残っていないバケットだけが、実際に削除されます。

## 参考

- [AWS CLI Command Reference: s3](https://docs.aws.amazon.com/cli/latest/reference/s3/)
- [AWS CLI Command Reference: s3api](https://docs.aws.amazon.com/cli/latest/reference/s3api/)
- [S3 の高レベルコマンドの設定](https://docs.aws.amazon.com/cli/latest/topic/s3-config.html)
- [Garage S3 Compatibility](https://garagehq.deuxfleurs.fr/documentation/reference-manual/s3-compatibility/)
