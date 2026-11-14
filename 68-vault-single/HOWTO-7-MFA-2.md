# Google の OIDC 認証と 2 段階認証で Vault にログインする

[HOWTO-7-MFA-1.md](HOWTO-7-MFA-1.md) の Community Edition 向けの方法として、Google を IdP にした OIDC 認証を設定します。
MFA は Google 側の 2 段階認証で強制します。Vault 自身は 2 要素目を要求しません。Google のログイン画面で 2 段階認証を通ったユーザーだけが、Vault にログインできます。

## 1. Google 側の設定

1. Google Cloud Console で **API とサービス → OAuth 同意画面** を設定します。
2. **認証情報 → OAuth クライアント ID** を作成します。種類は「ウェブ アプリケーション」です。
3. 「承認済みのリダイレクト URI」に次の 2 つを登録します。
   - `http://localhost:8250/oidc/callback`（CLI ログイン用）
   - `https://<公開ドメイン>:8200/ui/vault/auth/oidc/oidc/callback`（UI ログイン用）
4. 発行された Client ID と Client Secret を控えます。

**注意:** Google はリダイレクト URI に、公開されているトップレベルドメインか `localhost` しか認めません。`vault.labo.local` のような `.local` は登録できません。そのため、CLI は `localhost:8250` を使います。UI は公開ドメイン名で Vault にアクセスするか、SSH ポートフォワードで `localhost` 経由にします。

## 2. MFA の強制

- **Google Workspace の場合:** 管理コンソールの「セキュリティ → 認証 → 2 段階認証プロセス」で、全ユーザーに必須設定します。
- **個人の Gmail の場合:** 各アカウントで 2 段階認証を有効にします。Vault 側で強制はできません。

## 3. Vault 側の設定

```console
vault auth enable oidc

vault write auth/oidc/config \
    oidc_discovery_url="https://accounts.google.com" \
    oidc_client_id="<CLIENT_ID>.apps.googleusercontent.com" \
    oidc_client_secret="<CLIENT_SECRET>" \
    default_role="google"

vault write auth/oidc/role/google \
    role_type="oidc" \
    user_claim="email" \
    oidc_scopes="openid,email,profile" \
    allowed_redirect_uris="http://localhost:8250/oidc/callback" \
    allowed_redirect_uris="https://<公開ドメイン>:8200/ui/vault/auth/oidc/oidc/callback" \
    bound_claims='{"email":["takara@example.com"]}' \
    policies="admin" \
    ttl="1h"
```

`bound_claims` で、ログインを許可するユーザーを必ず絞ってください。絞らないと、Google アカウントを持つ誰でもログインできます。Workspace の場合は、ドメインで絞る方法もあります。

```console
bound_claims='{"hd":"example.com"}'
```

## 4. ログイン

CLI の場合です。ブラウザが開き、Google のログイン、2 段階認証、Vault への許可の順に進みます。

```console
vault login -method=oidc role=google
```

UI の場合は、メソッドに OIDC を選んで Role に `google` を入力します。

Vault を別マシンで動かしている場合は、CLI のコールバックポートを転送します。

```console
ssh -L 8250:localhost:8250 <vaultサーバー>
```

## 補足

- JWT 認証（`vault auth enable jwt`）は、発行済みの JWT をそのまま検証する方式で、対話ログインには向きません。人のログインには OIDC を使います。
- 自動化のログインには、OIDC を使わず AppRole などを別に用意します。
- root トークンと既存の `userpass` は、緊急用として残すか無効化するかを決めます。

## スマホの認証アプリ（Authenticator）の利用

この方法でも、スマホの認証アプリ（Google Authenticator、Microsoft Authenticator、Authy など）を使えます。
Google 側の 2 段階認証の方法として認証アプリを登録すれば、Vault の OIDC ログイン時にそのコードを使えます。Vault 側の追加設定は要りません。

### ログインの流れ

1. `vault login -method=oidc role=google` を実行します。
2. ブラウザで Google のログイン画面が開き、パスワードを入力します。
3. 2 段階認証の画面で、スマホの認証アプリの 6 桁コードを入力します。
4. Vault への許可後、ログインが完了します。

### Google アカウントでの設定

1. [Google アカウントのセキュリティ](https://myaccount.google.com/security) を開きます。
2. 「2 段階認証プロセス」を有効にします。
3. 「認証システム アプリ」を選び、QR コードをスマホの認証アプリで読み取って登録します。

Google Workspace の管理者が許可する 2 段階認証の方法を制限している場合は、認証アプリが使えるか確認してください。

### 注意点

- Google は、パスキーやスマホへの「ログインしますか？」の通知を優先して出すことがあります。通知を使いたくない場合は、「別の方法を試す」から認証システム アプリを選びます。
- Google は同じブラウザセッションでは再認証を省略することがあります。そのため、Vault のログインごとに毎回 2 要素目を求められるとは限りません。毎回確実に求めたい場合は、Enterprise の Login MFA（TOTP）を使います。この方法は [HOWTO-7-MFA-1.md](HOWTO-7-MFA-1.md) に書いてあります。
- Vault が Google の 2 段階認証の有無を検証するわけではありません。Workspace で 2 段階認証を必須にしておくのが確実です。
