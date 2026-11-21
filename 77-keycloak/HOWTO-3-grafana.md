# Grafana のログインを Keycloak で認証する（シングルサインオン）

Grafana の **Generic OAuth** 認証を使い、Keycloak を IdP（OpenID Provider）として連携します。Keycloak にログイン済みであれば、Grafana では再入力なしでログインできます（SSO）。Keycloak のグループを Grafana のロール（Admin / Editor / Viewer）へマッピングします。

## 全体像
1. Keycloak 側に、Grafana 用の OIDC クライアントを作成します。
2. Keycloak のグループを、ID トークンの `groups` クレームに含めます。
3. Grafana 側の `[auth.generic_oauth]` を設定します。
4. ログインとロールの割り当てを確認します。

## 前提
- Keycloak: `https://keycloak.labo.local:8443`、Realm 名は `labo`（[HOWTO-2.md](./HOWTO-2.md) と同じ例）
- Grafana: `https://grafana.labo.local:3000`（以降 `<grafana>` と表記。実際の URL に読み替えてください）
- Grafana から Keycloak へ名前解決・HTTPS 接続ができること
- Keycloak が自己署名 CA の場合、Grafana 側でその CA を信頼させること（「注意点」参照）

> Grafana が Kubernetes 上にある場合（[52-ansible-kubernetes](../52-ansible-kubernetes) の Helm デプロイ）の設定方法は「6. Helm でデプロイした Grafana の場合」を参照してください。

## 1. Keycloak 側の設定

1. Realm `labo` を作成し、ユーザーを登録します（作成済みなら不要）。
2. **Clients → Create client**
   - Client type: `OpenID Connect`
   - Client ID: `grafana`
   - Client authentication: **On**（confidential）
   - Authentication flow: **Standard flow** のみ有効（Direct access grants は Off）
   - Root URL: `https://<grafana>/`
   - Home URL: `https://<grafana>/`
   - Valid redirect URIs: `https://<grafana>/login/generic_oauth`
   - Valid post logout redirect URIs: `https://<grafana>/login`
   - Web origins: `https://<grafana>`
3. **Credentials** タブで Client secret を控えます。
4. **グループ連携**（[HOWTO-2.md](./HOWTO-2.md) で `groups` スコープを作成済みなら、手順 5 だけ行います）
   - Client scopes に `groups` スコープを作成します。
   - 「Group Membership」マッパーを追加し、Token claim name を `groups` にします。
   - Full group path は Off にします。
   - Add to ID token / access token / userinfo を On にします。
5. `grafana` クライアントの **Client scopes** タブで、`groups` スコープを Default として割り当てます。
6. グループを作成し、ユーザーを所属させます。

   | Keycloak グループ | Grafana ロール |
   |---|---|
   | `grafana-admins` | Admin |
   | `grafana-editors` | Editor |
   | （どちらにも属さない） | Viewer |

## 2. Grafana 側の設定（パッケージ / 手動インストール）

`/etc/grafana/grafana.ini` を編集します。

```ini
[server]
domain = grafana.labo.local
root_url = https://grafana.labo.local:3000/

[auth.generic_oauth]
enabled = true
name = Keycloak
allow_sign_up = true
client_id = grafana
client_secret = <Keycloakのクライアントシークレット>
scopes = openid email profile groups
auth_url = https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/auth
token_url = https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/token
api_url = https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/userinfo
login_attribute_path = preferred_username
name_attribute_path = name
email_attribute_path = email
use_pkce = true

# グループからロールを決定する（上から順に評価）
role_attribute_path = contains(groups[*], 'grafana-admins') && 'Admin' || contains(groups[*], 'grafana-editors') && 'Editor' || 'Viewer'
allow_assign_grafana_admin = true

# Keycloak のセッションを終了するログアウト
signout_redirect_url = https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/logout?post_logout_redirect_uri=https%3A%2F%2Fgrafana.labo.local%3A3000%2Flogin
```

- `root_url` は Keycloak に登録した Redirect URI の生成元になります。実際のアクセス URL と一致させてください。
- `allow_assign_grafana_admin = true` で、`Admin` ロールを持つユーザーに Grafana サーバー管理者権限も付与します。不要なら外します。
- Client secret をファイルに残したくない場合は、環境変数 `GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET` で渡します（systemd の場合は `/etc/default/grafana-server` など）。

設定を反映します。

```bash
sudo systemctl restart grafana-server
sudo journalctl -u grafana-server -f
```

## 3. ログインを確認する

1. ブラウザで `https://<grafana>/login` を開きます。
2. **Sign in with Keycloak** ボタンが表示されることを確認し、クリックします。
3. Keycloak のログイン画面でユーザー認証を行います。
4. Grafana に戻り、ログインできれば成功です。
5. 右上のユーザーメニュー → **Profile** で、ロールが `grafana-admins` / `grafana-editors` の所属どおりであることを確認します。

別の Grafana や Vault など、同じ Realm の他のクライアントを同一ブラウザで開くと、再入力なしでログインできます（SSO）。

## 4. パスワードログインを無効化する（任意）

SSO の動作確認後、運用に合わせて次の設定を追加します。

```ini
[auth]
# Grafana の自前ログインフォームを隠す
disable_login_form = true

[auth.generic_oauth]
# ログイン画面を経由せず Keycloak へ直接遷移する
auto_login = true
```

> `disable_login_form = true` にすると `admin` のパスワードでもログインできなくなります。障害時に備え、`https://<grafana>/login?disableAutoLogin=true` からのログインや、`grafana cli admin reset-admin-password` での復旧手順を確認してから設定してください。

## 5. ロールの割り当てを厳格にする（任意）

既定では、`role_attribute_path` の結果が得られないユーザーは既定ロール（`auto_assign_org_role`、通常は Viewer）になります。ロールが決まらないユーザーのログインを拒否したい場合は、次のように設定します。

```ini
[auth.generic_oauth]
role_attribute_path = contains(groups[*], 'grafana-admins') && 'Admin' || contains(groups[*], 'grafana-editors') && 'Editor' || contains(groups[*], 'grafana-viewers') && 'Viewer'
role_attribute_strict = true
```

この場合、いずれのグループにも属さないユーザーはログインできません。Keycloak 側で `grafana-viewers` グループも作成してください。

## 6. Helm でデプロイした Grafana の場合

[52-ansible-kubernetes](../52-ansible-kubernetes) の `addon_grafana` は `grafana/grafana` チャートを使います。`grafana.ini` の内容を `values.yaml` に記述します。

Client secret は Secret に保存します。

```bash
kubectl create secret generic grafana-oauth -n prometheus \
  --from-literal=GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET='<Keycloakのクライアントシークレット>'
```

`values.yaml` に追記します。

```yaml
envFromSecret: grafana-oauth

grafana.ini:
  server:
    root_url: https://k8s.labs.local:30000/
  auth.generic_oauth:
    enabled: true
    name: Keycloak
    allow_sign_up: true
    client_id: grafana
    scopes: openid email profile groups
    auth_url: https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/auth
    token_url: https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/token
    api_url: https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/userinfo
    login_attribute_path: preferred_username
    name_attribute_path: name
    email_attribute_path: email
    use_pkce: true
    role_attribute_path: "contains(groups[*], 'grafana-admins') && 'Admin' || contains(groups[*], 'grafana-editors') && 'Editor' || 'Viewer'"
    allow_assign_grafana_admin: true
    tls_client_ca: /etc/grafana/keycloak-ca/ca.crt
    signout_redirect_url: https://keycloak.labo.local:8443/realms/labo/protocol/openid-connect/logout?post_logout_redirect_uri=https%3A%2F%2Fk8s.labs.local%3A30000%2Flogin

# 自己署名 CA を Pod にマウントする
extraSecretMounts:
  - name: keycloak-ca
    secretName: keycloak-ca
    defaultMode: 0440
    mountPath: /etc/grafana/keycloak-ca
    readOnly: true
```

CA の Secret を作成し、チャートを更新します。この例のアクセス URL は `https://k8s.labs.local:30000/` です。Keycloak の Redirect URI にも `https://k8s.labs.local:30000/login/generic_oauth` を登録してください。

```bash
kubectl create secret generic keycloak-ca -n prometheus --from-file=ca.crt=keycloak-ca.pem
helm upgrade grafana -n prometheus -f values.yaml grafana/grafana
```

> Pod から `keycloak.labo.local` が名前解決できる必要があります。社内 DNS を参照できない場合は、`hostAliases` などで解決してください。

## 注意点
- **TLS**: Grafana のサーバーが Keycloak の証明書を検証できないと、ログイン後に `x509: certificate signed by unknown authority` のエラーになります。`keycloak_tls_mode: selfsigned` で作成したプライベート CA を、次のどちらかで信頼させます。
  - OS の信頼ストアに追加する: `sudo cp keycloak-ca.pem /usr/local/share/ca-certificates/keycloak-ca.crt && sudo update-ca-certificates` の後、`grafana-server` を再起動
  - `[auth.generic_oauth]` の `tls_client_ca` に CA ファイルのパスを指定する
- **Redirect URI**: Keycloak に登録した値と、Grafana の `root_url` から生成される `…/login/generic_oauth` が完全一致しないと `Invalid parameter: redirect_uri` になります。リバースプロキシ配下では `root_url` を外部 URL にしてください。
- **groups クレーム**: ロールが常に Viewer になる場合は、ID トークンに `groups` が含まれていません。Keycloak の **Client scopes → Evaluate** で生成されるトークンを確認してください。Full group path が On の場合は、値が `/grafana-admins` のようになるため、`role_attribute_path` の比較文字列も合わせます。
- **ロールの同期タイミング**: ロールはログインのたびに更新されます。Keycloak 側でグループを変更した場合は、Grafana から一度ログアウトして再ログインします。
- **既存ユーザーとの突合**: Grafana は E メールまたはログイン名でユーザーを突合します。既存のローカルユーザーと同じメールアドレスの場合、そのユーザーに紐付きます。
- **ログアウト**: `signout_redirect_url` を設定しないと、Grafana からログアウトしても Keycloak のセッションが残り、再度クリックするだけでログインされます。
- **MFA**: Keycloak 側で TOTP/WebAuthn を必須にすれば、Grafana ログインにも適用されます。
- **管理者の備え**: 障害時の備えとして、Grafana のローカル `admin` ユーザーは残しておきます。

## 関連
- [HOWTO-1.md](./HOWTO-1.md): Keycloak で出来ること
- [HOWTO-2.md](./HOWTO-2.md): Vault のログインを Keycloak で認証する
- [52-ansible-kubernetes](../52-ansible-kubernetes): Grafana のデプロイ
- [57-setup-with-node-exporter-alloy](../57-setup-with-node-exporter-alloy)
