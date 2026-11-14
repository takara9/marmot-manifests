# GitHub を外部 IdP にして MFA で Vault にログインする

[HOWTO-7-MFA-2.md](HOWTO-7-MFA-2.md) の Google と同様に、Community Edition でも GitHub を使って MFA を実現できます。
MFA は GitHub 側の 2 段階認証で強制します。Vault 自身は 2 要素目を要求しません。

## GitHub を使う場合の前提

GitHub は、人のログイン用の OIDC プロバイダー（Discovery URL）を提供していません。Google のように Vault の `oidc` 認証を直接つなぐことはできません。
次の 2 つの方法があります。

| 方法 | 内容 | ログイン方法 |
|---|---|---|
| **A. Vault の GitHub 認証** | Vault 標準の `github` 認証を使う。GitHub の Organization と Team を Vault のポリシーに対応付ける | Personal Access Token（PAT）を渡す |
| **B. Dex / Keycloak 経由の OIDC** | Dex や Keycloak を OIDC の仲介役として立て、GitHub を上流の IdP にする | ブラウザで GitHub にログイン |

## 1. GitHub 側で 2 段階認証を必須にする（共通）

1. GitHub の Organization を用意します。
2. **Organization の Settings → Authentication security → Require two-factor authentication** を有効にします。
3. 2 段階認証が未設定のメンバーは、Organization から外されます。

利用できる 2 段階認証の方法は、スマホの認証アプリ（TOTP）、セキュリティキー、パスキー、GitHub Mobile です。
Vault 側は「その Organization / Team のメンバーであること」だけを条件にすれば、2 段階認証済みのユーザーに限定できます。

## 2. 方法 A: Vault の GitHub 認証

### Vault 側の設定

```console
vault auth enable github

vault write auth/github/config \
    organization="<Organization 名>" \
    ttl="1h" max_ttl="4h"

# Team ごとにポリシーを割り当てる（Team 名は slug）
vault write auth/github/map/teams/vault-admins value="admin"

# 個人ごとに割り当てる場合
vault write auth/github/map/users/takara9 value="admin"
```

Organization 名の変更に備えて、`organization_id` も設定することを推奨します。ID は `gh api orgs/<Organization 名> --jq .id` で取得できます。

```console
vault write auth/github/config organization="<Organization 名>" organization_id=<ID>
```

### ログイン

GitHub で PAT を発行します。classic PAT で `read:org` スコープを付けます。
Organization が SAML SSO を使っている場合は、PAT に SSO の認可も必要です。

```console
vault login -method=github token="<PAT>"
```

環境変数でも渡せます。

```console
export VAULT_AUTH_GITHUB_TOKEN="<PAT>"
vault login -method=github
```

### 注意点

- 2 段階認証が効くのは、GitHub にログインして PAT を発行する時と、Organization への所属を保つ時です。`vault login` のたびに 2 要素目を求めるものではありません。
- PAT は漏えいするとパスワードと同じ効力を持ちます。有効期限を短くし、安全に保管してください。
- Vault の UI でも、GitHub を選んで PAT を入力すればログインできます。

## 3. 方法 B: Dex 経由の OIDC（ブラウザログイン）

ブラウザで GitHub にログインし、2 段階認証を通す流れにしたい場合の構成です。

```
Vault CLI / UI ──OIDC──> Dex（または Keycloak） ──OAuth──> GitHub（2 段階認証）
```

1. GitHub の **Settings → Developer settings → OAuth Apps** で OAuth App を作成します。
   Callback URL は Dex の `https://<dex ドメイン>/callback` です。
2. Dex の設定に GitHub コネクタを追加します。

```yaml
connectors:
  - type: github
    id: github
    name: GitHub
    config:
      clientID: <GitHub の Client ID>
      clientSecret: <GitHub の Client Secret>
      redirectURI: https://<dex ドメイン>/callback
      orgs:
        - name: <Organization 名>
          teams:
            - vault-admins
```

3. Vault に Dex を OIDC プロバイダーとして登録します。

```console
vault auth enable oidc

vault write auth/oidc/config \
    oidc_discovery_url="https://<dex ドメイン>" \
    oidc_client_id="vault" \
    oidc_client_secret="<Dex に登録した secret>" \
    default_role="github"

vault write auth/oidc/role/github \
    role_type="oidc" \
    user_claim="email" \
    groups_claim="groups" \
    oidc_scopes="openid,email,groups" \
    allowed_redirect_uris="http://localhost:8250/oidc/callback" \
    allowed_redirect_uris="https://vault.labo.local:8200/ui/vault/auth/oidc/oidc/callback" \
    bound_claims='{"groups":["<Organization 名>:vault-admins"]}' \
    policies="admin" \
    ttl="1h"
```

4. ログインします。

```console
vault login -method=oidc role=github
```

Dex や Keycloak は自分の環境内に立てられるため、Google と違い、リダイレクト URI に `vault.labo.local` のようなドメインを使えます。
Dex の `groups` クレームの形式は `<Organization 名>:<Team 名>` です。

## 方法の選び方

| 目的 | 推奨 |
|---|---|
| 手軽に始めたい、CLI 中心の運用 | 方法 A（Vault の GitHub 認証） |
| ブラウザで GitHub にログインさせ、UI でも使いたい | 方法 B（Dex / Keycloak 経由） |
| ログインごとに必ず 2 要素目を求めたい | Enterprise の Login MFA（[HOWTO-7-MFA-1.md](HOWTO-7-MFA-1.md)） |

## 補足

- root トークンと既存の `userpass` は、緊急用として残すか無効化するかを決めてください。
- 自動化のログインには、AppRole などを別に用意してください。
