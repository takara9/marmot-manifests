# AWS のサービスを使って Vault に MFA でログインする

[HOWTO-7-MFA-1.md](HOWTO-7-MFA-1.md) の Community Edition 向けの方法として、AWS のサービスを使って MFA を実現する方法です。
どちらの方法も、Vault 自身は 2 要素目を要求しません。MFA は AWS 側で強制します。

| 目的 | 方法 |
|---|---|
| 人がブラウザや UI でログインする | 方法 1: Cognito + OIDC |
| すでに AWS の IAM ユーザー / ロール運用がある、CLI 中心 | 方法 2: AWS 認証 + MFA 必須の IAM ロール |

## 方法 1: Amazon Cognito を OIDC の IdP にする（おすすめ）

Cognito ユーザープールは OIDC プロバイダーとして使え、MFA（TOTP / SMS など）をユーザープールの設定で必須にできます。
[HOWTO-7-MFA-2.md](HOWTO-7-MFA-2.md) の Google と同じ `oidc` 認証の構成で、ブラウザログインもできます。

### Cognito 側の設定

1. ユーザープールで MFA を「必須」にし、方式として認証アプリ（TOTP）を有効にします。
2. ホスト型 UI のドメインと、アプリクライアント（クライアントシークレット付き）を作成します。
3. コールバック URL に、`http://localhost:8250/oidc/callback` と Vault UI のコールバック URL を登録します。
4. Vault にログインさせるユーザーを、グループ（例: `vault-admins`）に所属させます。

### Vault 側の設定

```console
vault auth enable oidc

vault write auth/oidc/config \
    oidc_discovery_url="https://cognito-idp.<リージョン>.amazonaws.com/<USER_POOL_ID>" \
    oidc_client_id="<APP_CLIENT_ID>" \
    oidc_client_secret="<APP_CLIENT_SECRET>" \
    default_role="cognito"

vault write auth/oidc/role/cognito \
    role_type="oidc" \
    user_claim="email" \
    groups_claim="cognito:groups" \
    oidc_scopes="openid,email" \
    allowed_redirect_uris="http://localhost:8250/oidc/callback" \
    allowed_redirect_uris="https://vault.labo.local:8200/ui/vault/auth/oidc/oidc/callback" \
    bound_claims='{"cognito:groups":["vault-admins"]}' \
    policies="admin" \
    ttl="1h"
```

### ログイン

```console
vault login -method=oidc role=cognito
```

ブラウザで Cognito のログイン画面が開き、パスワードと TOTP コードを入力すると、ログインが完了します。

## 方法 2: Vault の AWS 認証と、IAM ロールの MFA 条件

Vault の `aws` 認証（iam 方式）で、MFA を要求する IAM ロールだけをログイン対象にします。
MFA は IAM ロールを引き受ける（AssumeRole）時に強制されます。Vault は MFA の有無を直接検証しないため、「MFA 付きでないと引き受けられないロール」だけを許可する形にします。

### IAM ロールの信頼ポリシー

```json
"Condition": { "Bool": { "aws:MultiFactorAuthPresent": "true" } }
```

### Vault 側の設定

```console
vault auth enable aws

vault write auth/aws/role/admin \
    auth_type=iam \
    bound_iam_principal_arn="arn:aws:iam::<アカウントID>:role/vault-admin" \
    policies="admin" \
    ttl="1h"
```

### ログイン

MFA 付きで一時認証情報を取得してから、ログインします。

```console
aws sts assume-role --role-arn arn:aws:iam::<アカウントID>:role/vault-admin \
    --role-session-name vault \
    --serial-number arn:aws:iam::<アカウントID>:mfa/<ユーザー> \
    --token-code <6桁コード>
# 出力された AccessKeyId / SecretAccessKey / SessionToken を
# AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_SESSION_TOKEN に設定

vault login -method=aws role=admin
```

## 注意

- IAM Identity Center（旧 AWS SSO）は、汎用の OIDC プロバイダーとしては使えません。SAML アプリケーションとして使えますが、Vault の SAML 認証は Enterprise 機能です。
- どちらの方法も、`vault login` のたびに 2 要素目を求めるものではありません。毎回求めたい場合は、Enterprise の Login MFA（[HOWTO-7-MFA-1.md](HOWTO-7-MFA-1.md)）を使います。
- 設定内容は、Vault と AWS の仕様を前提に書いており、実機では試していません。
