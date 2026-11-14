# Vault に多要素認証（MFA）でログインする

## 概要

Vault の Login MFA（ログイン時に追加の認証要素を要求する機能）は Enterprise / HCP Vault の機能です。
Community Edition では使えません。Community Edition で MFA を実現するには、外部 IdP に認証を任せます。

| 方法 | 内容 | 版 |
|---|---|---|
| **Login MFA**（TOTP / Duo / Okta / PingID） | userpass、LDAP、OIDC などのログイン時に、2 要素目の入力を要求する | Enterprise / HCP |
| **OIDC / JWT 認証** | 外部 IdP（Keycloak、Entra ID、Google、GitHub など）に認証を任せ、IdP 側で MFA を強制する | Community でも可 |
| **LDAP 認証** | LDAP の前段で MFA を使う構成にできる | Community でも可 |
| **TLS 証明書認証** | クライアント証明書で認証する。パスワードと組み合わせる構成は Vault 単体では強制できない | Community でも可 |

このリポジトリの Vault は Community Edition で、[HOWTO-1-basic-setting.md](HOWTO-1-basic-setting.md) の `userpass` 認証を使っています。
Enterprise ライセンスの設定は [vault.hcl](playbook/roles/vault/templates/vault.hcl) ではコメントアウトされています。

## Enterprise での TOTP MFA の例

```console
# TOTP 方式を作成
vault write identity/mfa/method/totp/my_totp \
    issuer=Vault period=30 key_size=30 algorithm=SHA256 digits=6

# ユーザーの entity を作成し、TOTP を紐付ける
vault write identity/entity name=takara

# entity_id を使って QR コード用の URL を生成
vault write -f identity/mfa/method/totp/my_totp/admin-generate entity_id=<ENTITY_ID>
# 出力された url を認証アプリ（Google Authenticator など）に登録

# userpass の mount accessor に MFA を強制
vault write identity/mfa/login-enforcement/enf1 \
    mfa_method_ids=<METHOD_ID> \
    auth_method_accessors=<userpass_accessor>
```

ログイン時は、TOTP コードを併せて渡します。

```console
vault login -method=userpass username=takara mfa="my_totp:123456"
```

## Community Edition で MFA を実現する場合

Keycloak などの IdP を立てて OIDC 認証を有効にし、IdP 側で OTP を必須にします。

```console
vault auth enable oidc
vault write auth/oidc/config oidc_discovery_url=... oidc_client_id=... oidc_client_secret=...
vault write auth/oidc/role/default ...
vault login -method=oidc
```
