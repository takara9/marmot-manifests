# Vault のログインを Keycloak で認証する

Vault の **OIDC 認証メソッド**を使い、Keycloak を IdP（OpenID Provider）として連携します。UI ログインと CLI ログインのどちらも対応します。

## 全体像
1. Keycloak 側に、Vault 用の OIDC クライアントを作成します。
2. Vault 側で `oidc` 認証メソッドを有効化し、Keycloak の Issuer を指定します。
3. Keycloak のグループやロールを Vault のポリシーへマッピングします。

## 1. Keycloak 側の設定
本リポジトリの Keycloak は `https://keycloak.labo.local:8443` です。Realm 名は例として `labo` とします。

1. Realm `labo` を作成し、ユーザーを登録します。
2. **Clients → Create client**
   - Client type: `OpenID Connect`
   - Client ID: `vault`
   - Client authentication: **On**（confidential）
   - Standard flow: 有効
   - Valid redirect URIs:
     - `https://<vault>:8200/ui/vault/auth/oidc/oidc/callback`
     - `https://<vault>:8200/oidc/callback`
     - `http://localhost:8250/oidc/callback`（CLI ログイン用）
3. **Credentials** タブで Client secret を控えます。
4. **グループ連携**
   - Client scopes に `groups` スコープを作成します。
   - 「Group Membership」マッパーを追加し、Token claim name を `groups` にします。
   - Full group path は Off にします。
   - Add to ID token / access token / userinfo を On にします。
   - `vault` クライアントの Client scopes にこのスコープを割り当てます。
5. グループ（例: `vault-admins`、`vault-readers`）を作成し、ユーザーを所属させます。

## 2. Vault 側の設定
```bash
export VAULT_ADDR=https://<vault>:8200

vault auth enable oidc

vault write auth/oidc/config \
  oidc_discovery_url="https://keycloak.labo.local:8443/realms/labo" \
  oidc_client_id="vault" \
  oidc_client_secret="<Keycloakのクライアントシークレット>" \
  default_role="default" \
  oidc_discovery_ca_pem=@keycloak-ca.pem   # 自己署名CAの場合
```
`oidc_discovery_ca_pem` には、`keycloak_tls_mode: selfsigned` で作成したプライベート CA の証明書を指定します。Vault サーバーが Keycloak の証明書を検証できる必要があります。

ロールを作成します。
```bash
vault write auth/oidc/role/default \
  bound_audiences="vault" \
  allowed_redirect_uris="https://<vault>:8200/ui/vault/auth/oidc/oidc/callback" \
  allowed_redirect_uris="https://<vault>:8200/oidc/callback" \
  allowed_redirect_uris="http://localhost:8250/oidc/callback" \
  user_claim="preferred_username" \
  oidc_scopes="openid,profile,email,groups" \
  groups_claim="groups" \
  token_policies="default" \
  token_ttl=1h
```

## 3. グループとポリシーの紐付け（外部グループ）
```bash
# ポリシー作成
vault policy write admin-policy admin.hcl

# 外部グループ作成
vault write identity/group name="vault-admins" type="external" \
  policies="admin-policy"

# OIDC アクセサを取得
ACCESSOR=$(vault auth list -format=json | jq -r '."oidc/".accessor')

# Keycloak のグループ名とエイリアスを対応付け
GROUP_ID=$(vault read -field=id identity/group/name/vault-admins)
vault write identity/group-alias name="vault-admins" \
  mount_accessor="$ACCESSOR" canonical_id="$GROUP_ID"
```
同様に `vault-readers` などを作成します。

## 4. ログイン方法
- **UI**: Method で `OIDC` を選び、Sign in with OIDC Provider をクリックします。Keycloak のログイン画面に遷移します。
- **CLI**: `vault login -method=oidc role=default`

## 注意点
- **TLS**: Vault が Keycloak の証明書を検証できないと失敗します。CA の指定が必要です。
- **Redirect URI**: Keycloak と Vault の登録値が完全一致しないとエラーになります。
- **groups クレーム**: 空のままだとポリシーが付与されません。ID トークンの中身を確認してください。
- **MFA**: Keycloak 側で TOTP/WebAuthn を必須にすれば、Vault ログインにも適用されます。
- **root トークン**: 障害時の備えとして、初期 root やローカルの管理者手段は残しておきます。
- **クラスタ構成 (75-vault-cluster)**: 認証メソッドの設定は Raft で共有されます。リーダーに一度設定すれば足ります。
- **JWT 認証との違い**: ブラウザログインではなく、CI やアプリからのトークン認証を行う場合は `jwt` 認証メソッドを使います。

## 関連
- [68-vault-single](../68-vault-single)
- [75-vault-cluster](../75-vault-cluster)
- [HOWTO-1.md](./HOWTO-1.md): Keycloak で出来ること
