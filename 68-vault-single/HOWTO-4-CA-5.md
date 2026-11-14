# Vault PKI の Root CA / 中間CA 証明書のリニューアル

[HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) で作成した Root CA（`pki/`、TTL 10 年）と中間CA（`pki_int/`、TTL 5 年）には有効期限があります。
CA 証明書が期限切れになると、そのCAが発行した全てのサーバー証明書の検証が失敗します。本書では、期限前にCA証明書を更新する手順を説明します。

> 本書の手順は Vault の複数 Issuer 機能（Vault 1.11 以降）を前提にしています。
> 手順に使うコマンドは、本番に適用する前に必ず検証環境で確認してください。

## 1. 基本方針

### 1.1 「リニューアル」には 2 種類ある

| 方式 | 内容 | 特徴 |
|---|---|---|
| 同一鍵で再発行 | 既存の秘密鍵を使い、新しい有効期限の証明書を作る | 公開鍵が変わらないため、既存の下位証明書の検証が引き続き成立する。鍵漏えい対策にはならない |
| 鍵も新規作成（ローテーション） | 新しい鍵ペアと証明書を作る | 鍵を刷新できる。クライアントに新しい証明書の配布が必要 |

### 1.2 更新の順序と影響範囲

- **中間CAの更新:** Root CA は変わらないため、**クライアントへの再配布は不要**です。更新作業は Vault 内で完結します。
- **Root CA の更新:** 全クライアントの信頼ストアの更新が必要です。**最も影響が大きい**ため、期限の十分前（目安: 1 年以上前）から始めます。
- 中間CAの有効期限は、署名した Root CA の有効期限を超えないようにします。Root CA の残存期間が中間CAの TTL より短い場合は、先に Root CA を更新します。

### 1.3 有効期限が近いサーバー証明書と混同しない

サーバー証明書（例: `app.labo.local`）は短 TTL で自動更新する運用です。ここで扱うのは、その発行元である CA 自体の証明書です。

## 2. 事前準備

### 2.1 現在の状態と有効期限を確認

```bash
# Issuer の一覧
vault list pki/issuers
vault list pki_int/issuers

# Issuer の詳細（issuer_name、key_id、ca_chain など）
vault read pki/issuer/root-2026
vault read pki_int/issuer/int-2026

# 有効期限
vault read -field=certificate pki/issuer/root-2026     | openssl x509 -noout -subject -enddate
vault read -field=certificate pki_int/issuer/int-2026  | openssl x509 -noout -subject -enddate
```

期限を定期的に監視するスクリプト例です。

```bash
#!/bin/bash
# 期限まで 90 日を切ったら警告する
for m in pki pki_int; do
  for id in $(vault list -format=json $m/issuers | jq -r '.[]'); do
    cert=$(vault read -field=certificate $m/issuer/$id)
    if ! echo "$cert" | openssl x509 -noout -checkend $((90*24*3600)) >/dev/null; then
      echo "WARN: $m issuer $id expires within 90 days:"
      echo "$cert" | openssl x509 -noout -subject -enddate
    fi
  done
done
```

### 2.2 バックアップ

作業前に Raft スナップショットを取得します。

```bash
vault operator raft snapshot save vault-before-ca-renewal.snap
```

スナップショットには CA の秘密鍵が含まれます。安全な場所に保管し、不要になれば削除してください。

### 2.3 作業用の変数

```bash
export VAULT_ADDR=https://vault.labo.local:8200
```

## 3. 中間CAのリニューアル

Root CA は変更せず、中間CAだけを更新します。クライアントへの配布は不要です。

### 3.1 方式A: 新しい鍵で更新（推奨）

#### (1) 新しい CSR を生成

```bash
vault write -format=json pki_int/intermediate/generate/internal \
  common_name="labo.local Intermediate CA" key_name="int-key-2031" \
  | jq -r '.data.csr' > pki_int_2031.csr
```

#### (2) Root CA で署名

```bash
vault write -format=json pki/root/sign-intermediate \
  csr=@pki_int_2031.csr format=pem_bundle ttl=43800h \
  | jq -r '.data.certificate' > int_2031.cert.pem
```

Root CA に複数の Issuer がある場合は、署名に使う Issuer を指定します。

```bash
vault write -format=json pki/issuer/root-2026/sign-intermediate \
  csr=@pki_int_2031.csr format=pem_bundle ttl=43800h \
  | jq -r '.data.certificate' > int_2031.cert.pem
```

#### (3) 署名済み証明書を登録

```bash
vault write pki_int/intermediate/set-signed certificate=@int_2031.cert.pem
```

`imported_issuers` に新しい Issuer の ID が表示されます。

#### (4) Issuer に名前を付け、既定の Issuer に切り替える

```bash
vault list pki_int/issuers

# 新しい Issuer の ID を指定して名前を付ける
vault write pki_int/issuer/<新しいIssuerのID> issuer_name="int-2031"

# 以降の発行は新しい Issuer で署名される
vault write pki_int/config/issuers default=int-2031
```

ロール `labo-local` は `issuer_ref=default` のため、切り替え後に発行される証明書は新しい中間CAで署名されます。

#### (5) 動作確認

```bash
vault write -format=json pki_int/issue/labo-local common_name="test.labo.local" ttl=1h \
  | jq -r '.data.issuing_ca' | openssl x509 -noout -subject -issuer -dates -fingerprint -sha256
```

`issuing_ca` が新しい中間CA（有効期限が延びたもの）であることを確認します。

### 3.2 方式B: 同一の鍵で再発行

鍵を変えずに有効期限だけを延ばす場合です。既存の鍵 ID を指定して CSR を作ります。

```bash
# 既存の鍵 ID を確認
vault read -field=key_id pki_int/issuer/int-2026

vault write -format=json pki_int/intermediate/generate/existing \
  common_name="labo.local Intermediate CA" key_ref=<既存のkey_id> \
  | jq -r '.data.csr' > pki_int_renew.csr
```

以降は方式A の (2)〜(5) と同じです。

### 3.3 旧中間CAで発行済みの証明書

- 発行済みのサーバー証明書は、期限まで有効です。旧中間CAが期限切れになるまでは検証が成立します。
- 旧中間CAの Issuer は、発行済み証明書がすべて失効・期限切れになるまで削除しないでください（CRL の発行にも必要です）。
- サーバー側は、更新後に再発行した証明書へ入れ替えます。`certificate` + `ca_chain` を連結して設定します（HOWTO-4-CA-2.md 参照）。

## 4. Root CA のリニューアル

### 4.1 方式A: 同一の鍵で新しい証明書を作る（クライアント影響が小さい）

公開鍵と Subject が同じ新しい Root CA 証明書を、新しい有効期限で作ります。既存の中間CAは再署名しなくても、新しい Root 証明書で検証できます。

```bash
# 既存の Root の鍵 ID を確認
vault read -field=key_id pki/issuer/root-2026

vault write -field=certificate pki/root/generate/existing \
  common_name="labo.local Root CA" key_ref=<既存のkey_id> \
  issuer_name="root-2036" ttl=87600h > root_ca_2036.crt

# 新しい Root を既定にする
vault write pki/config/issuers default=root-2036
```

- 新旧の Root 証明書は、同じ公開鍵と同じ Subject を持ちます。クライアントは新しい証明書を信頼ストアに登録すれば、新旧どちらの中間CAも検証できます。
- 鍵そのものは更新されないため、**鍵の漏えいや危殆化への対策としては使えません。** その場合は 4.2 を実施します。

### 4.2 方式B: 新しい鍵で Root CA をローテーション

#### (1) 新しい Root CA を作成

```bash
vault write -field=certificate pki/root/rotate/internal \
  common_name="labo.local Root CA G2" issuer_name="root-2036" \
  key_name="root-key-2036" ttl=87600h > root_ca_2036.crt
```

`rotate` は新しい Root を作成し、既定の Issuer に設定します。旧 Root（`root-2026`）は Issuer として残り、旧中間CAの CRL 発行などに使われ続けます。

> `common_name` を変えると、旧 Root と区別でき、信頼ストア上でも見分けやすくなります。

#### (2) 新しい Root で中間CAを作り直す

```bash
vault write -format=json pki_int/intermediate/generate/internal \
  common_name="labo.local Intermediate CA" key_name="int-key-2036" \
  | jq -r '.data.csr' > pki_int_2036.csr

vault write -format=json pki/issuer/root-2036/sign-intermediate \
  csr=@pki_int_2036.csr format=pem_bundle ttl=43800h \
  | jq -r '.data.certificate' > int_2036.cert.pem

vault write pki_int/intermediate/set-signed certificate=@int_2036.cert.pem
vault write pki_int/issuer/<新しいIssuerのID> issuer_name="int-2036"
```

この時点では、まだ既定の Issuer を切り替えません（クライアントへの配布が終わっていないため）。

#### (3) 新旧の Root をクライアントに配布する（並行運用）

ここが最も重要です。**新しい Root CA が全クライアントに行き渡るまで、新しい中間CAからの発行に切り替えないでください。**

1. 新しい `root_ca_2036.crt` を、全クライアントの信頼ストアに**追加**します。旧 Root は削除しません。
2. 配布方法は [HOWTO-4-CA-3.md](HOWTO-4-CA-3.md)（Windows / Linux / macOS）、[HOWTO-4-CA-4.md](HOWTO-4-CA-4.md)（Kubernetes のノード・Pod）を参照します。
3. 配布の完了を確認します。例:

```bash
# 新しい Root だけを信頼ストアとして使い、検証できるか確認
openssl verify -CAfile root_ca_2036.crt -untrusted int_2036.cert.pem issued_new.crt
```

#### (4) 発行を新しい中間CAへ切り替える

```bash
vault write pki_int/config/issuers default=int-2036
```

切り替え後に発行・更新される証明書は、新しい Root → 新しい中間CAのチェーンになります。

#### (5) 旧チェーンの証明書の置き換え

- 旧中間CAで発行した証明書は、期限までは旧 Root で検証できます。
- 短 TTL の証明書は、自動更新により、自然に新しいチェーンへ移行します。
- 長期の証明書は、手動で再発行して入れ替えます。

### 4.3 クロス署名（旧クライアントを残したまま移行する場合）

旧 Root しか持たないクライアントと、新 Root しか持たないクライアントが混在する間、同じ中間CAの鍵を両方の Root で署名します。

```bash
# 既存の中間CAの鍵から、新しい Root 宛の CSR を作る
vault write -format=json pki_int/intermediate/cross-sign \
  common_name="labo.local Intermediate CA" key_ref=<既存の中間CAのkey_id> \
  | jq -r '.data.csr' > pki_int_cross.csr

# 新しい Root で署名
vault write -format=json pki/issuer/root-2036/sign-intermediate \
  csr=@pki_int_cross.csr format=pem_bundle ttl=43800h \
  | jq -r '.data.certificate' > int_cross.cert.pem

# 登録
vault write pki_int/intermediate/set-signed certificate=@int_cross.cert.pem
```

サーバーが提示する `ca_chain` は、クライアントが信頼している側のチェーンで検証が成立します。通常のラボ環境では 4.2 の並行運用で十分です。

## 5. 旧 Issuer の後始末

旧Root / 旧中間CAで発行した証明書が、すべて期限切れ・失効になるまで待ってから実施します。

```bash
# 不要な失効済み・期限切れの証明書を整理
vault write pki_int/tidy tidy_cert_store=true tidy_revoked_certs=true safety_buffer=72h

# 旧 Issuer の削除（鍵も不要なら削除）
vault delete pki_int/issuer/int-2026
vault delete pki_int/key/<旧key_id>
```

クライアントの信頼ストアからも、旧 Root CA を削除します（HOWTO-4-CA-3.md の削除手順）。**削除は、旧チェーンの証明書がすべてなくなったことを確認してから**行います。

## 6. 検証

```bash
# 現在の Issuer と既定の Issuer
vault list pki_int/issuers
vault read pki_int/config/issuers

# 発行した証明書のチェーンを検証（クライアント視点）
vault write -format=json pki_int/issue/labo-local common_name="verify.labo.local" ttl=1h > issued.json
jq -r '.data.certificate' issued.json > leaf.crt
jq -r '.data.ca_chain[]' issued.json > chain.pem
openssl verify -CAfile root_ca_2036.crt -untrusted chain.pem leaf.crt
rm issued.json

# サーバーへの接続確認
curl -v https://app.labo.local/
openssl s_client -connect app.labo.local:443 -servername app.labo.local </dev/null 2>/dev/null \
  | grep -E "Verify return code"
```

`Verify return code: 0 (ok)` と `leaf.crt: OK` を確認します。

## 7. ロールバック

- 既定の Issuer を戻すだけで、発行元を元に戻せます。

```bash
vault write pki_int/config/issuers default=int-2026
```

- 旧 Issuer を削除する前であれば、切り戻しが可能です。**旧 Issuer の削除は、新しいチェーンでの運用が安定してから**行います。
- 重大な問題が起きた場合は、2.2 で取得したスナップショットから復元できます（ただし、スナップショット取得後の変更は失われます）。

## 8. チェックリスト

| # | 項目 |
|---|---|
| 1 | Root / 中間CAの有効期限を確認した（期限の 1 年以上前に開始） |
| 2 | Raft スナップショットを取得した |
| 3 | 中間CAの新しい有効期限が、Root CA の有効期限を超えていない |
| 4 | Root CA 更新時は、新 Root を全クライアント（ノード OS・Pod 含む）へ配布してから、発行を切り替えた |
| 5 | `ca_chain` を含めてサーバー証明書を再設定した |
| 6 | `openssl verify` と実サーバーへの接続で検証した |
| 7 | 旧 Issuer は、旧チェーンの証明書がなくなるまで残した |
| 8 | 次回の更新時期を、カレンダーや監視に登録した |
