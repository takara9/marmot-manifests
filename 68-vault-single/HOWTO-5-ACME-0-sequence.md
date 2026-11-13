# ACME プロトコル シーケンス図

ACME（RFC 8555）で証明書を発行する流れです。クライアントは Ansible の `community.crypto.acme_certificate`、ACME サーバーは Vault の PKI（`pki_int/roles/labo-local/acme/directory`）を想定しています。

## 全体シーケンス

```mermaid
sequenceDiagram
    autonumber
    participant C as ACME クライアント<br/>(Ansible / 対象ホスト)
    participant S as ACME サーバー<br/>(Vault PKI)
    participant V as 検証対象<br/>(HTTP サーバー or DNS)

    Note over C,S: 1. ディレクトリ取得・Nonce 取得
    C->>S: GET /directory
    S-->>C: 各エンドポイント URL (newNonce, newAccount, newOrder ...)
    C->>S: HEAD /new-nonce
    S-->>C: Replay-Nonce

    Note over C,S: 2. アカウント登録（初回のみ）
    C->>S: POST /new-acct (JWS: アカウント公開鍵, 利用規約同意)
    S-->>C: 201 Created + アカウント URL (kid)

    Note over C,S: 3. オーダー作成
    C->>S: POST /new-order (identifiers: app.labo.local)
    S-->>C: 201 Created + order (status: pending, authorizations, finalize URL)

    Note over C,S: 4. 認可（Authorization）とチャレンジ
    C->>S: POST-as-GET /authz/{id}
    S-->>C: authorization (challenges: http-01 / dns-01, token)
    C->>C: keyAuthorization = token + "." + アカウント鍵の thumbprint

    alt http-01
        C->>V: /.well-known/acme-challenge/{token} に keyAuthorization を配置
    else dns-01
        C->>V: _acme-challenge.app.labo.local に TXT = base64url(SHA-256(keyAuthorization)) を追加
    end

    C->>S: POST /chall/{id} ({}) 検証開始を通知
    S-->>C: 200 OK (challenge: processing)

    S->>V: 検証（HTTP GET もしくは DNS TXT 問い合わせ）
    V-->>S: keyAuthorization / TXT レコード
    S->>S: 値を照合

    loop status が valid になるまでポーリング
        C->>S: POST-as-GET /authz/{id}（または order）
        S-->>C: status: pending → valid
    end

    Note over C,S: 5. Finalize（証明書発行）
    C->>C: 秘密鍵と CSR を作成（事前に作成済みでも可）
    C->>S: POST /finalize/{id} (CSR)
    S-->>C: order (status: processing → valid, certificate URL)

    Note over C,S: 6. 証明書取得
    C->>S: POST-as-GET /cert/{id}
    S-->>C: 証明書チェーン (PEM: fullchain)

    Note over C,V: 7. 後処理
    C->>V: チャレンジ用ファイル / TXT レコードを削除
    C->>C: Nginx へ配置し reload
```

## 補足

- 初回以外のリクエストは、すべて JWS（JSON Web Signature）で署名され、`nonce` と `url` を含みます。アカウント登録後は `jwk` の代わりに `kid`（アカウント URL）で識別されます。
- 手順 5 の CSR のドメインは、手順 3 の `identifiers` と一致している必要があります。
- Ansible の `acme_certificate` を使う場合、手順 4 の「チャレンジ取得」と「検証開始以降」の 2 段階に分かれます。間でチャレンジの応答（ファイル配置または TXT 追加）を行います。詳細は [HOWTO-6-ACME-5.md](./HOWTO-6-ACME-5.md) を参照してください。
- 更新（renew）時は、手順 2 のアカウント登録を省略し、手順 3 以降を再実行します。

## 状態遷移

```mermaid
stateDiagram-v2
    direction LR
    state "Order" as O {
        [*] --> pending
        pending --> ready: 全 Authorization が valid
        ready --> processing: finalize (CSR 送信)
        processing --> valid: 証明書発行
        pending --> invalid: 検証失敗
        ready --> invalid
        processing --> invalid
    }
    state "Authorization / Challenge" as A {
        [*] --> a_pending
        a_pending --> a_processing: 検証開始通知
        a_processing --> a_valid: 検証成功
        a_processing --> a_invalid: 検証失敗
    }
```
