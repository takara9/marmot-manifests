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

1. Realm `labo` を作成し、ユーザーを登録します（作成済みなら不要。詳細は下記「Realm とユーザーの作成手順」を参照）。
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

### Realm とユーザーの作成手順

Realm は、ユーザーやアプリの認証設定をまとめる領域です。以下は、上記の手順 1 の詳細です。

#### `labo` realm を作成する

1. 管理コンソール `https://keycloak.labo.local:8443/admin/` を開き、Keycloak の管理者アカウントでログインします。
2. 左上の realm 選択メニュー（最初は通常 `master`）を開きます。
3. **Create realm** をクリックします。
4. 以下を設定し、**Create** をクリックします。

   | 項目 | 設定値 |
   |---|---|
   | Realm name | `labo` |
   | Enabled | On |

すでに `labo` が一覧にあれば、作成せず選択してください。

表示名を `LABO` にしたい場合は、作成後の **Realm settings → General → Display name** で設定します。認証 URL に使われるのは表示名ではなく、Realm name の `labo` です。

#### ログイン用ユーザーを登録する

1. 選択中の realm が `labo` であることを確認します。
2. 左メニューの **Users → Add user** を開きます。
3. 以下を入力します。

   | 項目 | 入力例 |
   |---|---|
   | Username | `grafana-user` |
   | Email | 利用者のメールアドレス |
   | First name / Last name | 利用者の名前 |

4. **Create** をクリックします。
5. 作成したユーザーの **Credentials** タブを開きます。
6. **Set password** でパスワードと確認用パスワードを入力します。
   - **Temporary: On**: 初回ログイン時にパスワード変更を要求します。
   - **Temporary: Off**: 設定したパスワードでそのまま利用できます。
7. **Save** をクリックし、確認画面でも保存します。

管理者アカウントとは別に、`labo` 内に作成したこのユーザーで Grafana にログインします。ユーザーが作成済みなら、再作成は不要です。

#### ユーザーのログインを確認する

次の URL を開き、作成したユーザーでログインします。

```text
https://keycloak.labo.local:8443/realms/labo/account/
```

ログインできれば、realm とユーザーの準備は完了です。

**Realm とユーザーを作るだけでは「Client not found」は解消しません。** 続けて、同じ `labo` realm で上記の手順 2（**Clients → Create client**）から Client ID `grafana` を登録し、クライアント設定とグループ連携を行ってください。

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

## 7. Keycloak で MFA（2 要素認証）を必須にする（任意）

Grafana と GitLab は Keycloak で認証するため、`labo` realm で MFA を有効にすれば、どちらのログインにも適用されます。アプリごとの 2FA 設定は不要です。この手順は、Keycloak の認証アプリ（TOTP）を前提にします。画面名は Keycloak のバージョンにより異なることがあります。

### 前提

- ユーザーが、Google Authenticator や Microsoft Authenticator などの TOTP 対応アプリを利用できること。
- 管理者が、MFA を設定しても **ローカル管理者（Grafana の `admin`、GitLab の `root`）のパスワードログインは MFA の対象外**であることを理解していること。

### 手順 1: 認証ポリシーを確認する

1. 管理コンソールで realm `labo` を選びます。
2. **Authentication → Policies → OTP Policy** を開きます。
3. 標準の `Time based`、6 桁、30 秒などの値を確認します。変更する場合は、すでに登録済みの認証アプリに影響しないか確認してください。

### 手順 2: Configure OTP を必須にする

Keycloak の Browser フローは、既定では OTP を登録済みのユーザーだけに入力を求めます。全員に求めるには、OTP の登録を必須アクションにします。

**新規ユーザーにも自動で適用する**

1. **Authentication → Required actions** を開きます。
2. **Configure OTP** が **Enabled** であることを確認します。
3. **Set as default action** を On にします。

この設定は、設定後に作成するユーザーに適用されます。既存ユーザーには適用されません。

**既存ユーザーに適用する**

1. **Users → 対象ユーザー → Details** を開きます。
2. **Required user actions** で **Configure OTP** を選びます。
3. **Save** をクリックします。

ユーザーが OTP を登録すると、この必須アクションは完了します。以降のログインでは、パスワードの後に OTP の入力が求められます。

### 手順 3: ログインを確認する

1. テスト用ユーザーの Keycloak セッションをログアウトします。ログアウトについては、[GitLab の「ログアウトとユーザーの切り替え」](./HOWTO-3-GitLab.md#8-ログアウトとユーザーの切り替え) を参照してください。
2. `https://<grafana>/login` を開き、**Sign in with Keycloak** をクリックします。
3. ユーザー名とパスワードを入力します。
4. **Configure OTP** の画面で QR コードを認証アプリで読み取り、表示されたコードを入力します。
5. Grafana に戻り、ログインできることを確認します。
6. 再度ログアウトして Keycloak にログインし、今度は OTP の入力画面が表示されることを確認します。

### 管理者と復旧

- Keycloak の管理者は `master` realm のユーザーです。`labo` の設定だけでは保護されないため、管理者アカウントにも OTP を設定してください。
- ユーザーが認証アプリを失った場合は、**Users → 対象ユーザー → Credentials** で OTP を削除し、**Required user actions** に **Configure OTP** を再設定します。本人確認のうえで実施してください。
- 認証アプリの交換に備え、管理者の復旧手順を事前に決めておきます。
- セキュリティキーなどを使う場合は、Keycloak の **Required actions** と **Authentication flows** で WebAuthn / Passkeys を設定します。運用方法は、使用している Keycloak のバージョンのドキュメントで確認してください。

### 注意

- MFA は、Keycloak を経由するログインだけに適用されます。Grafana のローカル `admin` ユーザーのログインフォームは別です。必要に応じて、[4. パスワードログインを無効化する](#4-パスワードログインを無効化する任意) を検討してください。ただし、障害時の復旧手順を確認してから設定してください。
- Grafana OSS には、ここで使えるアプリ側の 2FA 設定は確認していません。Keycloak 側の MFA を使用してください。

## 注意点
- **Client not found**: 「Sign in with Keycloak」が表示されても、Keycloak 側のクライアント登録が完了しているとは限りません。ボタンを押した後のブラウザの URL で `/realms/<realm名>/` と `client_id` を確認し、管理コンソールでその realm の **Clients** に同じ Client ID があることを確認してください。この例では realm は `labo`、Client ID は `grafana` です。画面の `LABO` は表示名の場合があるため、URL の realm 名で判断します。`master` や別の realm に作成したクライアントは利用できません。未登録なら「1. Keycloak 側の設定」の手順で作成します（Keycloak のインストール用 Playbook は realm / クライアントを作成しません）。登録済みなら Grafana の `client_id` と `auth_url` を修正し、`token_url` / `api_url` も同じ realm に揃えて Grafana を再起動します。環境変数 `GF_AUTH_GENERIC_OAUTH_CLIENT_ID` / `GF_AUTH_GENERIC_OAUTH_AUTH_URL` による上書きも確認してください。Client secret や Redirect URI の不一致は、このエラーとは別の段階で確認します。
- **TLS**: Grafana のサーバーが Keycloak の証明書を検証できないと、ログイン後に `x509: certificate signed by unknown authority` のエラーになります。`keycloak_tls_mode: selfsigned` で作成したプライベート CA を、次のどちらかで信頼させます。
  - OS の信頼ストアに追加する: `sudo cp keycloak-ca.pem /usr/local/share/ca-certificates/keycloak-ca.crt && sudo update-ca-certificates` の後、`grafana-server` を再起動
  - `[auth.generic_oauth]` の `tls_client_ca` に CA ファイルのパスを指定する
- **Redirect URI**: Keycloak に登録した値と、Grafana の `root_url` から生成される `…/login/generic_oauth` が完全一致しないと `Invalid parameter: redirect_uri` になります。リバースプロキシ配下では `root_url` を外部 URL にしてください。
- **groups クレーム**: ロールが常に Viewer になる場合は、ID トークンに `groups` が含まれていません。Keycloak の **Client scopes → Evaluate** で生成されるトークンを確認してください。Full group path が On の場合は、値が `/grafana-admins` のようになるため、`role_attribute_path` の比較文字列も合わせます。
- **ロールの同期タイミング**: ロールはログインのたびに更新されます。Keycloak 側でグループを変更した場合は、Grafana から一度ログアウトして再ログインします。
- **既存ユーザーとの突合**: Grafana は E メールまたはログイン名でユーザーを突合します。既存のローカルユーザーと同じメールアドレスの場合、そのユーザーに紐付きます。
- **ログアウト**: `signout_redirect_url` を設定しないと、Grafana からログアウトしても Keycloak のセッションが残り、再度クリックするだけでログインされます。
- **MFA**: Keycloak 側で TOTP/WebAuthn を必須にすれば、Grafana ログインにも適用されます。手順は「7. Keycloak で MFA（2 要素認証）を必須にする」を参照してください。
- **管理者の備え**: 障害時の備えとして、Grafana のローカル `admin` ユーザーは残しておきます。

## 関連
- [HOWTO-1.md](./HOWTO-1.md): Keycloak で出来ること
- [HOWTO-2.md](./HOWTO-2.md): Vault のログインを Keycloak で認証する
- [52-ansible-kubernetes](../52-ansible-kubernetes): Grafana のデプロイ
- [57-setup-with-node-exporter-alloy](../57-setup-with-node-exporter-alloy)
