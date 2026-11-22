# Vault のログインを Keycloak で認証する（シングルサインオン）

Vault の OIDC 認証を使い、Keycloak にユーザー認証を委任します。同じブラウザで同じ Realm の Grafana などにログイン済みなら、Keycloak のセッションが有効な間はパスワードを再入力せずに Vault にログインできます。

この例では Keycloak の `vault-readers` グループに所属するユーザーだけをログインさせ、Vault の `secret/` KV v2 を読み取り専用にします。グループと権限は運用に合わせて変更してください。

## 前提

- Keycloak: `https://keycloak.labo.local:8443`、Realm: `labo`
- Vault: `https://vault.labo.local:8200`、認証マウントパス: `oidc/`（[68-vault-single](../68-vault-single) の例）
- Vault の初期化とアンシールが済み、認証メソッド・ポリシー・Identity を設定できる管理者でログインできること。既存の管理者ログインは動作確認まで残してください。
- Vault サーバーから Keycloak の名前解決と HTTPS 接続ができること。ブラウザからは両方に接続でき、両方の TLS 証明書を信頼できること。
- コマンドを実行する端末に Vault CLI と `jq` があること。`secret/` に KV v2 が有効になっていること（[KV ストアの手順](../68-vault-single/HOWTO-2-kv-store.md)）。
- URL、ポート、Realm、マウントパスを変更する場合は、以下の Keycloak と Vault の設定を同じ値に読み替えてください。

## 1. Keycloak の設定

1. 管理コンソール `https://keycloak.labo.local:8443/admin/` で `labo` Realm を選択します。Realm とユーザーの作成は [Grafana の手順](./HOWTO-3-grafana.md#realm-とユーザーの作成手順)を参照してください。
2. **Clients → Create client** で以下を設定します。
   - Client type: `OpenID Connect`
   - Client ID: `vault`
   - Client authentication: **On**
   - Authentication flow: **Standard flow** を有効、**Direct access grants** を無効
   - Valid redirect URIs:
     - `https://vault.labo.local:8200/ui/vault/auth/oidc/oidc/callback`（UI）
     - `http://localhost:8250/oidc/callback`（CLI）
3. 作成後、**Credentials** タブで Client secret を確認します。秘密情報なのでリポジトリやシェル履歴に記録しないでください。
4. **Client scopes** で `groups` スコープを作成します。既にあれば再利用します。
   - **Mappers → Configure a new mapper → Group Membership**
   - Token claim name: `groups`
   - Full group path: **Off**
   - Add to ID token: **On**
5. `vault` クライアントの **Client scopes** で `groups` を **Default** として追加します。
6. **Groups** でトップレベルの `vault-readers` を作成し、ログインさせるユーザーを所属させます。グループ名は大文字・小文字を含めて Vault の設定と一致させてください。

## 2. 証明書と接続の確認

Vault CLI を実行する端末で、Vault の証明書を検証できるようにします。プライベート CA の場合は、その CA 証明書を安全な方法で入手し、次のように指定します。

```bash
export VAULT_ADDR="https://vault.labo.local:8200"
export VAULT_CACERT="/path/to/vault-ca.crt"
vault status
```

公開 CA を使用する場合など、システムの信頼ストアで検証できる場合は `VAULT_CACERT` は不要です。

Keycloak がこのリポジトリの `selfsigned` 構成の場合、コマンドを実行する端末に CA 証明書を取得します。取得先と証明書の正当性を確認してください。

```bash
scp ubuntu@keycloak.labo.local:/etc/keycloak/tls/ca.crt ./keycloak-ca.crt
curl --fail --cacert ./keycloak-ca.crt \
  https://keycloak.labo.local:8443/realms/labo/.well-known/openid-configuration | jq -r .issuer
```

Issuer が `https://keycloak.labo.local:8443/realms/labo` であることと、Vault サーバーからも同じ URL に接続できることを確認します。`provided` 構成では Keycloak の証明書を発行した CA を用意してください。

## 3. Vault の OIDC 設定

以下は Bash を使用し、管理者として `vault login` した端末で実行します。Client secret の入力と設定送信は、別々に実行してください。

### 3.1. OIDC 認証メソッドを確認・有効化する

先に `vault auth list` を確認し、`oidc/` が既に存在する場合は次の有効化コマンドを省略します。以下の config や `default` ロールの書き込みは既存設定を変更するため、`vault read auth/oidc/config`、`vault read auth/oidc/role/default` を確認し、別の用途に使っている場合は別パス・別ロールを選んでください。

```bash
vault auth enable -path=oidc oidc
```

### 3.2. Keycloak の CA 証明書を配置する

以降は CA 証明書を `$HOME/keycloak-ca.crt` に配置して使います。手順 2 で別の場所に取得した場合は、その証明書をここへコピーしてください。既に配置済みならコピーは不要です。

`.ssh` に取得した場合は、次を実行します。

```bash
cp "$HOME/.ssh/keycloak-ca.crt" "$HOME/keycloak-ca.crt"
```

ファイルの所有者と読み取り権限が正しくても、Snap 版 Vault などのアクセス制限により、`.ssh` 配下では `permission denied` になる場合があります。SSH 秘密鍵をコピーしたり、`.ssh` の権限を緩めたりする必要はありません。`VAULT_CACERT` も `.ssh` 配下を参照している場合は、その CA 証明書も同様にコピーして参照先を更新してください。

CA ファイルは Vault CLI が実行される端末上のものです。Vault に CA 証明書の内容が保存され、Vault サーバーが Keycloak を検証するために使います。

### 3.3. Client secret を入力する

**このコードブロックだけを先に実行します。** Keycloak の **labo → Clients → vault → Credentials** に表示される Client secret を貼り付けて Enter を押してください。入力内容は表示されません。次の送信コマンドまでまとめて貼り付けないでください。

```bash
unset KEYCLOAK_CLIENT_SECRET
read -rs -p "Keycloak client secret: " KEYCLOAK_CLIENT_SECRET </dev/tty; echo
```

### 3.4. OIDC 設定を送信する

入力後、同じ端末で次を実行します。空の場合は送信せず、エラーを表示します。CA ファイルを絶対パスで指定するため、実行ディレクトリに依存しません。

```bash
if [ -n "$KEYCLOAK_CLIENT_SECRET" ]; then
  printf '%s' "$KEYCLOAK_CLIENT_SECRET" | vault write auth/oidc/config \
    oidc_discovery_url="https://keycloak.labo.local:8443/realms/labo" \
    oidc_discovery_ca_pem=@"$HOME/keycloak-ca.crt" \
    oidc_client_id="vault" \
    oidc_client_secret=- \
    default_role="default"
else
  printf '%s\n' "Client secret が空です。入力コマンドから再実行してください。" >&2
fi
unset KEYCLOAK_CLIENT_SECRET
```

設定が成功したことを確認してから次へ進みます。失敗時にも `unset` を実行しているため、再試行するときは **3.3 の Client secret 入力から**やり直してください。

`both 'oidc_client_id' and 'oidc_client_secret' must be set for OIDC` は、Client ID をこの例のとおり指定している場合、Client secret が空で送信されたことを示します。入力と送信を分けて再実行してください。Secret を確認のために画面やログへ出力しないでください。

Keycloak が Vault サーバーのシステムで信頼する CA を使用している場合は、3.2 と `oidc_discovery_ca_pem` の行を省略できます。設定を書き込む際には前後の行の `\` に注意してください。

### 3.5. OIDC ロールを作成する

ロールを作成します。`bound_claims` は `vault-readers` に所属しないユーザーのこのロールでのログインを拒否します。他のロールや認証方法は制限しません。`sub` は Keycloak のユーザー ID であり、ユーザー名の変更によって Vault 上の識別子が変わるのを避けます。

```bash
vault write auth/oidc/role/default - <<'EOF'
{
  "role_type": "oidc",
  "user_claim": "sub",
  "groups_claim": "groups",
  "bound_audiences": ["vault"],
  "bound_claims": {"groups": ["vault-readers"]},
  "oidc_scopes": ["profile", "groups"],
  "allowed_redirect_uris": [
    "https://vault.labo.local:8200/ui/vault/auth/oidc/oidc/callback",
    "http://localhost:8250/oidc/callback"
  ],
  "token_ttl": "1h"
}
EOF
```

リダイレクト URI は Keycloak と Vault に完全一致で登録し、UI へのアクセスにも同じホスト名とポートを使ってください。

### 3.6. 保存された設定を確認する

```bash
vault read auth/oidc/config
vault read auth/oidc/role/default
```

Discovery URL、Client ID、既定ロール、許可するリダイレクト URI、グループ制限が意図した値であることを確認し、手順 4 のポリシー設定へ進んでください。

## 4. グループとポリシーを紐付ける

既存の同名ポリシーや Identity グループがないことを確認してから実行してください。この例は KV v2 の値の読み取りとキーの一覧のみを許可します。

```bash
vault policy write vault-sso-reader - <<'EOF'
path "secret/data/*" {
  capabilities = ["read"]
}
path "secret/metadata/*" {
  capabilities = ["list"]
}
EOF

vault write identity/group name="vault-readers" type="external" \
  policies="vault-sso-reader"
ACCESSOR=$(vault auth list -format=json | jq -r '."oidc/".accessor')
GROUP_ID=$(vault read -field=id identity/group/name/vault-readers)
vault write identity/group-alias name="vault-readers" \
  mount_accessor="$ACCESSOR" canonical_id="$GROUP_ID"
```

`vault-readers` は Keycloak の `groups` クレームの値と完全一致させます。別グループを追加する場合は、対応するポリシーと外部グループ・エイリアスを作成し、必要に応じてロールの `bound_claims` の許可リストにも追加します。管理者権限を付与する場合は対象を慎重に限定してください。

## 5. ログインと権限を確認する

### UI

1. `https://vault.labo.local:8200/ui/` を開き、認証方法 **OIDC** を選択します。ロール名は空欄（既定の `default`）または `default` にします。
2. **Sign in with OIDC Provider** をクリックし、`vault-readers` に所属する Keycloak ユーザーで認証します。
3. Vault に戻ることと、`secret/` の許可されたデータを読めることを確認します。
4. Keycloak にログイン済みの同じブラウザで Vault にログインし、再入力なしで認証されることを確認します。Keycloak のセッションが期限切れの場合などは再認証が必要です。

### CLI

ブラウザを使う端末で実行します。その端末で `localhost:8250` を使用でき、ブラウザが Keycloak に接続できる必要があります。`vault login` は通常、その端末のトークンヘルパーに保存済みのトークンを置き換えるため、管理者の復旧手段を確保してから実行してください。

```bash
vault login -method=oidc role=default
vault token lookup
vault kv list secret/
```

`vault token lookup` の `identity_policies` に `vault-sso-reader` が含まれることを確認します。既知のシークレットがあれば `vault kv get secret/<パス>` で読み取りを確認し、書き込みが拒否されることも確認してください。`vault-readers` に属さないユーザーでも試し、ログインが拒否されることを確認します。

SSH 先のサーバーで CLI を実行し、手元のブラウザを使う場合は、別の手元の端末から `ssh -L 8250:localhost:8250 ubuntu@vault.labo.local` を実行し、表示された Keycloak の認証 URL を手元のブラウザで開きます。ポート転送中、手元と SSH 先の双方でポート 8250 が利用可能である必要があります。

## 注意点

- **TLS エラー**: Vault CLI から Vault、Vault サーバーから Keycloak、ブラウザから両サーバーのそれぞれで CA の信頼と名前解決を確認します。証明書検証を無効化しないでください。
- **権限が付かない／ログインできない**: Keycloak の `groups` が ID トークンに含まれること、ユーザーの所属、グループ名、`bound_claims`、エイリアスのマウントアクセサを確認します。トークンや Client secret をログに出力しないでください。
- **権限変更**: Keycloak のグループ変更は Vault のログイン時に反映されます。既存トークンの失効や有効期間も考慮してください。
- **MFA**: 必要なら Keycloak の Realm で MFA を強制します。同じセッションでは毎回 MFA を求められるとは限りません。
- **緊急時のアクセス**: Keycloak の障害時にも管理できるよう、既存の管理者手段を安全に保管してください。
- **クラスタ**: [75-vault-cluster](../75-vault-cluster) では認証設定はクラスタで共有されます。実際にブラウザが利用する Vault URL と証明書に合わせて URI を設定してください。

## 関連

- [Keycloak の構築と TLS](./README.md)
- [Grafana の Keycloak 認証](./HOWTO-3-grafana.md)
- [Vault 公式: JWT/OIDC 認証](https://developer.hashicorp.com/vault/docs/auth/jwt)
- [Vault 公式: Keycloak の設定](https://developer.hashicorp.com/vault/docs/auth/jwt/oidc-providers/keycloak)
