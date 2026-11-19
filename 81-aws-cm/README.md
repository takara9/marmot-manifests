# 81-aws-cm

手元（Ansible実行ホスト）にあるTLS証明書を AWS Certificate Manager (ACM) へインポートする Ansible playbook です。
社内CA（[62_ca-acme](../62_ca-acme/README.md)、[63-ca-labo-local](../63-ca-labo-local/ca-labo-local.yaml)）で発行した証明書を、ALB/NLB/CloudFront 等で使う用途を想定しています。

## ファイル

- [playbook/import-cert.yaml](playbook/import-cert.yaml): `community.aws.acm_certificate` でインポート
- [playbook/vars.yaml](playbook/vars.yaml): リージョン、証明書名、PEMファイルのパス

## 前提

```console
pip install boto3 botocore
ansible-galaxy collection install amazon.aws community.aws
```

AWS認証情報は環境変数（`AWS_ACCESS_KEY_ID` など）、または `acm_aws_profile` でプロファイル指定します。
IAMには `acm:ImportCertificate`、`acm:ListCertificates`、`acm:DescribeCertificate`、`acm:GetCertificate`、`acm:ListTagsForCertificate`、`acm:AddTagsToCertificate`、`acm:RemoveTagsFromCertificate` が必要です。

## 使い方

PEMファイルを `playbook/certs/` に置き、`vars.yaml` を編集して実行します。

```console
cd playbook
ansible-playbook import-cert.yaml
# 変数を上書きする場合
ansible-playbook import-cert.yaml -e acm_region=us-east-1 -e acm_cert_name=api.labo.example.com
```

| 変数 | 説明 |
|---|---|
| `acm_region` | 登録先リージョン（CloudFront用は `us-east-1`） |
| `acm_aws_profile` | AWSプロファイル名（空なら環境変数） |
| `acm_cert_name` | Name タグ。同名があれば更新（ARNは維持） |
| `acm_cert_arn` | 更新対象のARNを明示する場合のみ |
| `acm_cert_file` | サーバー証明書（PEM、leafのみ） |
| `acm_key_file` | 秘密鍵（PEM、パスフレーズなし） |
| `acm_chain_file` | 中間CA証明書（PEM）。不要なら空文字 |

## 注意

- 同じ `acm_cert_name` で再実行すると、証明書を新規作成せず同じARNに上書きインポートします（証明書の更新時もALBリスナー等の設定変更は不要）。
- 差分判定はモジュール側で行われます（内容が同一なら `changed` にならない設計）。モックAWSでは常に `changed` となったため、実AWSでの挙動は初回実行時に確認してください。
- ログに秘密鍵が出ないよう、インポートタスクは `no_log: true` にしています。失敗原因の調査時のみ一時的に外してください（`-v` 系オプションは併用しないこと）。
- ACMは暗号化された秘密鍵を受け付けません。必要なら `openssl rsa -in enc.key -out server.key` で解除してください。
- fullchain の場合は、先頭をサーバー証明書、残りを中間CA証明書に分割してください。
- 秘密鍵はリポジトリにコミットしないでください。
