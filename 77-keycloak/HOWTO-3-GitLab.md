# GitLab の認証を Keycloak に委任する（OpenID Connect）

GitLab CE の OmniAuth / OpenID Connect（OIDC）を使い、ブラウザでのログインを Keycloak に委任します。同じ realm の Grafana などに Keycloak でログイン済みであれば、Keycloak のセッションを利用して GitLab にもログインできます。

この手順は [61-gitlab-ce](../61-gitlab-ce/README.md) の Linux パッケージ（Omnibus）構成を対象にします。コンテナや Helm 用の設定ではありません。

## 全体像

1. Keycloak の `labo` realm に GitLab 用のクライアントを作成します。
2. GitLab が Keycloak の CA 証明書を信頼するように設定します。
3. GitLab に OIDC プロバイダーを登録します。
4. ログイン・アカウントの承認または既存アカウントとの連携を確認します。
5. プロジェクトやグループへのアクセス権を GitLab 側で付与します。

## 前提

| 項目 | この手順の値 |
|---|---|
| Keycloak | `https://keycloak.labo.local:8443` |
| Realm name | `labo` |
| GitLab | `http://gitlab.labo.local`（既存 Playbook の既定値） |
| Client ID | `gitlab` |
| GitLab の provider name | `openid_connect` |
| Callback URL | `http://gitlab.labo.local/users/auth/openid_connect/callback` |

- GitLab がインストール済みで、ローカル管理者としてログインできること。
- ブラウザから GitLab と Keycloak の両方に接続できること。
- **GitLab サーバーからも** `keycloak.labo.local` を名前解決でき、HTTPS の 8443 番ポートに接続できること。端末だけに `/etc/hosts` を登録しても、GitLab サーバーからの接続には反映されません。
- `labo` realm とログイン用ユーザーが作成済みであること。未作成なら [Grafana の手順の「Realm とユーザーの作成手順」](./HOWTO-3-grafana.md#realm-とユーザーの作成手順) を実施してください。Keycloak のインストール用 Playbook は realm / クライアントを作成しません。
- ユーザーの Email、名前、パスワードを設定しておくこと。メールアドレスは実際に利用者が所有するものを使用し、確認済みのものを使用してください。

> HTTP の GitLab は信頼できるラボ内の検証用です。実運用では GitLab 側も HTTPS を使用してください。すでに `https://gitlab.labo.local` で運用している場合は、この手順の GitLab URL と Callback URL をすべて HTTPS に読み替えます。ポートやパスがある場合も、実際の `external_url` に合わせてください。この手順では GitLab の HTTP / HTTPS 切り替えは行いません。

## 1. Keycloak に GitLab クライアントを作成する

1. `https://keycloak.labo.local:8443/admin/` を開き、管理者でログインします。
2. 左上の realm 選択で **`labo`** を選びます。`LABO` という表示名ではなく、realm の実際の名前が `labo` であることを確認してください。
3. **Clients → Create client** を開きます。
4. 以下を入力します。

   | 項目 | 値 |
   |---|---|
   | Client type | OpenID Connect |
   | Client ID | `gitlab` |
   | Name | `GitLab`（表示用、任意） |

5. Capability config で以下を設定します。
   - **Client authentication: On**（confidential）
   - **Standard flow: On**
   - **Direct access grants: Off**
   - この手順では Implicit flow や Service accounts roles は使用しません。
6. Login settings を設定して保存します。

   | 項目 | 値 |
   |---|---|
   | Root URL | `http://gitlab.labo.local/` |
   | Home URL | `http://gitlab.labo.local/` |
   | Valid redirect URIs | `http://gitlab.labo.local/users/auth/openid_connect/callback` |
   | Web origins | `http://gitlab.labo.local` |

   Redirect URI はワイルドカードを使わず、実際の Callback URL を登録します。既存の `grafana` クライアントを流用せず、GitLab 専用のクライアントを作成してください。

7. **Credentials** タブで Client secret を確認します。後で GitLab サーバーに設定します。シークレットをリポジトリ、チャット、スクリーンショットに含めないでください。
8. **Client scopes** で、標準の `profile` と `email` が割り当てられていることを確認します。この手順では `openid email profile` を要求します。

ユーザーは `labo` 内の既存ユーザーを利用できます。認証だけなら、`gitlab-admins` などのグループ、Group Membership マッパー、グループの Role mapping は不要です。

## 2. GitLab に Keycloak の CA 証明書を登録する

Keycloak にプライベート CA の証明書を使っている場合は、GitLab サーバー上で CA を登録します。公的 CA など、GitLab がすでに信頼している CA の場合はコピーを省略できます。

GitLab の Linux パッケージは独自の証明書ストアを使います。Ubuntu の `update-ca-certificates` だけで済ませず、`/etc/gitlab/trusted-certs` にも登録してください。

以下は **GitLab サーバー上**で実行します。

```bash
scp ubuntu@keycloak.labo.local:/etc/keycloak/tls/ca.crt ./keycloak-ca.crt
sudo install -d -m 0755 /etc/gitlab/trusted-certs
sudo install -m 0644 keycloak-ca.crt /etc/gitlab/trusted-certs/keycloak-labo-ca.crt
sudo gitlab-ctl reconfigure
```

取得するのは **CA の公開証明書**です。Keycloak の秘密鍵をコピーしないでください。中間 CA がある場合は、各 CA 証明書を 1 ファイル 1 証明書で登録します。

GitLab に同梱された Ruby で、証明書検証と discovery の接続を確認します。

```bash
sudo /opt/gitlab/embedded/bin/ruby -rnet/http -rjson -e \
  'url = URI("https://keycloak.labo.local:8443/realms/labo/.well-known/openid-configuration"); response = Net::HTTP.get_response(url); response.value; puts JSON.parse(response.body).fetch("issuer")'
```

次の値が表示されることを確認します。

```text
https://keycloak.labo.local:8443/realms/labo
```

TLS 検証の無効化で回避せず、CA、証明書の SAN、名前解決を修正してください。

## 3. GitLab の OIDC 設定

GitLab サーバー上で `/etc/gitlab/gitlab.rb` を編集します。

```bash
sudo editor /etc/gitlab/gitlab.rb
```

以下を追加します。既存の OmniAuth 設定がある場合は統合し、同じキーを重複して定義しないでください。既存の `omniauth_providers` に他のプロバイダーがある場合、その配列に GitLab 用の OIDC 定義を追加します。

```ruby
gitlab_rails['omniauth_enabled'] = true
gitlab_rails['omniauth_allow_single_sign_on'] = ['openid_connect']
gitlab_rails['omniauth_block_auto_created_users'] = true
gitlab_rails['omniauth_auto_link_user'] = false

gitlab_rails['omniauth_providers'] = [
  {
    name: 'openid_connect',
    label: 'Keycloak',
    args: {
      name: 'openid_connect',
      scope: ['openid', 'profile', 'email'],
      response_type: 'code',
      issuer: 'https://keycloak.labo.local:8443/realms/labo',
      discovery: true,
      client_auth_method: 'basic',
      uid_field: 'sub',
      send_scope_to_token_endpoint: false,
      pkce: true,
      client_options: {
        identifier: 'gitlab',
        secret: '<Keycloak の gitlab クライアントの Client secret>',
        redirect_uri: 'http://gitlab.labo.local/users/auth/openid_connect/callback'
      }
    }
  }
]
```

- `secret` の仮値を実際の Client secret に置き換えます。ファイルの読み取り権限は管理者に限定し、内容を公開しないでください。
- `name` は **`openid_connect` のまま**にします。ボタンの表示名だけを `label: 'Keycloak'` で変更します。
- `issuer` は realm の URL です。`/protocol/openid-connect/auth` や `/.well-known/openid-configuration` は付けません。
- `identifier` は Keycloak の Client ID、`redirect_uri` は Keycloak に登録した Valid redirect URIs と完全一致させます。
- `uid_field: 'sub'` で、Keycloak のユーザー ID を識別子に使用します。運用開始後に識別子や realm を変更すると、既存アカウントとの連携に影響します。
- 新規ユーザーは自動作成しますが、管理者の承認までログインを許可しない設定です。既存アカウントとはメールアドレスだけで自動連携せず、本人の操作で連携します。

反映します。

```bash
sudo gitlab-ctl reconfigure
sudo gitlab-ctl status
```

### Ansible で設定を維持する場合

[61-gitlab-ce の変数ファイル](../61-gitlab-ce/playbook/roles/gitlab/vars/main.yaml) の `gitlab_extra_config` が、`gitlab.rb` の `ANSIBLE MANAGED BLOCK` に書き込まれます。

- Ansible 管理にする場合は、上記の Ruby 設定を既存の `gitlab_extra_config` に統合します。`time_zone` や `high_availability['mountpoint']` などの既存設定は維持してください。
- シークレットは Ansible Vault で暗号化した変数として管理し、`secret: '{{ gitlab_oidc_client_secret }}'` のように参照します。Ruby の単一引用符内に挿入する場合は、値に引用符やバックスラッシュが含まれないことを確認するか、適切にエスケープしてください。
- サーバー上で手動設定する場合は、管理対象ブロックの外に記述してください。ブロック内の手動変更は次回の Playbook 実行で上書きされます。手動設定と Ansible 設定を二重に追加しないでください。

Ansible 経由で反映する場合は、[61-gitlab-ce](../61-gitlab-ce/) から実行します。

```bash
ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
```

## 4. ログインとアカウントの確認

### 新規ユーザー

1. `http://gitlab.labo.local/users/sign_in` を開きます。
2. **Keycloak** ボタンをクリックします。
3. `labo` のユーザーで認証します。Keycloak のセッションが有効なら、資格情報の再入力を省略できる場合があります。
4. 新規アカウントは承認待ちになります。別のブラウザまたはセッションで、GitLab のローカル管理者としてログインします。
5. **Admin → Overview → Users**（バージョンによっては **Admin → Users**）で対象ユーザーを確認し、承認します。画面の表記はバージョンにより異なります。
6. 対象ユーザーで再度 Keycloak ログインを行い、GitLab のホーム画面が開くことを確認します。

管理する `labo` ユーザー全員について即時ログインを許可するラボ構成に限り、`omniauth_block_auto_created_users = false` に変更できます。その場合、この realm で認証に成功したユーザーが管理者の承認なしに GitLab アカウントを作成できることを理解した上で設定してください。

### 既存の GitLab ユーザー

既存のプロジェクトや権限を維持するため、新しいアカウントを作成するのではなく連携します。

1. 既存の GitLab ユーザー名・パスワードでログインします。
2. アバター → **Edit profile → Access → Password and authentication → Service sign-in** を開きます。古いバージョンでは **Account / Connected accounts** などの表記の場合があります。
3. **Keycloak** を選び、連携したい Keycloak ユーザーで認証します。
4. GitLab に戻り、連携を確認します。
5. ログアウトし、Keycloak ボタンからログインして、同じ GitLab アカウントとプロジェクトにアクセスできることを確認します。

## 5. 認証とアクセス権は別に管理する

**この OIDC 設定は認証の委任であり、GitLab の管理者権限やプロジェクト権限の自動付与ではありません。**

- Keycloak の `grafana-admins` / `grafana-editors` に所属していても、GitLab の権限は変わりません。
- GitLab のプロジェクトまたはグループのメンバー管理から、必要なユーザーに Developer / Maintainer などを付与します。
- GitLab 全体の管理者権限は、GitLab の管理画面で必要なユーザーにのみ付与します。
- Grafana の `role_attribute_path` に相当する設定を、この GitLab OIDC 定義に追加することはできません。Keycloak の `groups` クレームを送るだけではメンバーシップは同期されません。
- グループ同期などを追加する場合は、GitLab のバージョン、ライセンス、対応する認証方式を確認し、別途設計してください。

## 6. トラブルシューティング

| 症状 | 確認箇所 |
|---|---|
| Keycloak ボタンが表示されない | `omniauth_enabled`、`omniauth_providers`、`gitlab-ctl reconfigure` の成功。Admin の Sign-in restrictions で OIDC が無効化されていないか |
| Keycloak に `Client not found` | 認証 URL の realm が `labo` か、同じ realm に Client ID `gitlab` があるか。`master` に登録しても利用できません |
| `Invalid parameter: redirect_uri` | Keycloak の Valid redirect URIs、GitLab の `redirect_uri`、実際の `external_url` のスキーム・ホスト・ポート・パス |
| `invalid_client` / Client authentication failed | Client authentication が On か、`identifier` / `secret` が正しいか。シークレット再生成後は GitLab 側も更新 |
| certificate verify failed | `/etc/gitlab/trusted-certs` の CA、reconfigure、Keycloak 証明書の SAN と有効期限 |
| 名前解決エラー / 接続タイムアウト | GitLab サーバー側の DNS、経路、Keycloak の 8443 番ポート |
| メール情報不足でアカウント作成できない | Keycloak ユーザーの Email、`email` スコープ、User info の `email` |
| 認証後に承認待ちになる | この手順では意図した動作。GitLab 管理者で新規アカウントを承認 |
| GitLab でログアウトしても、Keycloak ボタンで前のユーザーに自動ログインされる | 異常ではありません。Keycloak のセッションが残っています。「8. ログアウトとユーザーの切り替え」を参照 |
| ログインできるがプロジェクトが見えない | GitLab のプロジェクト / グループのメンバーシップ。既存アカウントと正しく連携したか |

GitLab サーバーでエラーを確認します。

```bash
sudo gitlab-ctl tail gitlab-rails
```

ログやトークンを共有するときは、Client secret、Cookie、認可コード、アクセストークン、ユーザーの個人情報を除去してください。

## 7. 運用上の注意と復旧

- 動作確認が終わるまでは、ローカル管理者のログイン手段を残します。この手順ではパスワードログインを無効化せず、Keycloak への自動リダイレクトも設定しません。
- Keycloak の MFA は Keycloak の認証時に適用されます。GitLab の 2FA を自動的に無効化する設定は追加していません。設定手順は「9. MFA（2 要素認証）を必須にする」を参照してください。
- GitLab からのログアウトと Keycloak のセッション終了は同じではありません。詳細と対処は「8. ログアウトとユーザーの切り替え」を参照してください。
- この設定はブラウザでのログイン用です。Git over SSH の鍵、Git over HTTPS や API のアクセストークンは別途管理します。
- Keycloak 連携を停止する場合は、`omniauth_providers` からこの OIDC 定義を外し、関連設定を整理して `sudo gitlab-ctl reconfigure` を実行します。他の OmniAuth プロバイダーを使っている場合は残してください。

## 8. ログアウトとユーザーの切り替え

### 現象

GitLab で **Sign out** した後にサインイン画面で **Keycloak** ボタンをクリックすると、ログアウト前のユーザーで自動的にログインされることがあります。user1 でログアウトして user2 でログインしたつもりでも、user1 として扱われます。

### 原因

GitLab の Sign out は、GitLab のセッションだけを終了します。Keycloak のセッションは残るため、Keycloak ボタンをクリックすると、Keycloak は認証済みのユーザーを再利用し、ログイン画面を表示せずに GitLab へ戻します。**GitLab のログアウトが失敗しているわけではありません。**

ブラウザのプライベートウィンドウでも、すでに開いているウィンドウと Cookie が共有されている場合は、同じ状態になることがあります。確認するときは、プライベートウィンドウをすべて閉じてから開き直してください。

### ユーザーを切り替える

1. GitLab で Sign out します。
2. `https://keycloak.labo.local:8443/realms/labo/account/` を開き、右上のユーザーメニューから **Sign out** を実行します。
3. `https://gitlab.labo.local/users/sign_in` を開きます。URL は実際の `external_url` に合わせてください。
4. **Keycloak** ボタンをクリックします。Keycloak のログイン画面が表示されるので、別のユーザーで認証します。

### 毎回ログイン画面を表示する（任意）

共有端末などで、Keycloak のセッションがあっても毎回認証を要求したい場合は、[3. GitLab の OIDC 設定](#3-gitlab-の-oidc-設定) の `args` に `prompt: 'login'` を追加します。

```ruby
args: {
  name: 'openid_connect',
  prompt: 'login',
  # 他の設定は変更しない
}
```

設定を反映します。

```bash
sudo gitlab-ctl reconfigure
```

Keycloak に認証済みでも、ログインのたびに資格情報を求められます。SSO の利便性は下がるため、必要な環境だけで使用してください。この設定は、導入している GitLab のバージョンで動作を確認してから運用に適用してください。

### 放置したセッションを自動で終了する

Keycloak の管理コンソールで **Realm settings → Sessions** を開き、**SSO Session Idle** と **SSO Session Max** を運用に合わせて短くします。この設定は `labo` realm のすべてのクライアントに影響するため、Grafana や Vault など他のアプリへの影響を確認してください。

### SAML に切り替えても同時ログアウトできるとは限りません

GitLab と Keycloak の同時ログアウトを目的に、OIDC から SAML へ切り替えることは推奨しません。

- GitLab の SAML 設定手順で、SAML Single Logout（SLO）の設定項目を確認できませんでした。
- GitLab 公式の OmniAuth 共通設定のドキュメントでは、SAML などでの federated sign out（IdP も含むログアウト）は機能強化の途中であるとされています。
- Keycloak が SAML のログアウトに対応していても、GitLab 側が IdP へのログアウトを連動させなければ、GitLab のログアウトで Keycloak のセッションは終了しません。

同時ログアウトが運用上必須の場合は、導入している GitLab のバージョンのリリースノートと公式ドキュメントで対応状況を確認してください。未対応なら、上記の運用対処を使用するか、別途ログアウト連携を設計してください。

## 9. MFA（2 要素認証）を必須にする

### Keycloak 側で設定する（推奨）

GitLab の認証は Keycloak に委任しているため、`labo` realm で MFA を必須にすれば、Keycloak ボタンからのログインに適用されます。Grafana など同じ realm の他のアプリにも同時に適用されます。

設定手順は [Grafana の「Keycloak で MFA（2 要素認証）を必須にする」](./HOWTO-3-grafana.md#7-keycloak-で-mfa2-要素認証を必須にする任意) を参照してください。Keycloak 管理者の OTP 設定と、認証アプリを失ったユーザーの復旧手順も記載しています。

GitLab で確認するときは、まず [8. ログアウトとユーザーの切り替え](#8-ログアウトとユーザーの切り替え) の手順で Keycloak からもログアウトしてください。Keycloak のセッションが残っていると、再認証されず、OTP の入力画面が表示されないことがあります。

### GitLab 自身の 2FA との関係

GitLab には、独自の 2FA 機能があります。Keycloak の MFA とは別の機能です。

- `omniauth_allow_bypass_two_factor` の既定値は `false` です。ユーザーが GitLab の 2FA を有効にしていると、Keycloak で認証した後にも GitLab の 2FA を求められる場合があります。
- 管理画面の **Settings → General → Sign-in restrictions** で、すべてのユーザーに 2FA を必須にする設定は、OmniAuth のユーザーにも GitLab 側の 2FA 登録を求める可能性があります。Keycloak の MFA と二重になるため、どちらかに統一するかを決めてから設定してください。
- 二重の入力を避けたい場合は、Keycloak の MFA を必須にしたうえで、次の設定で OIDC のログインに GitLab の 2FA を省略できます。**Keycloak で MFA を必須にしていない状態では設定しないでください。**

```ruby
gitlab_rails['omniauth_allow_bypass_two_factor'] = ['openid_connect']
```

設定を `/etc/gitlab/gitlab.rb` の既存の OmniAuth 設定に統合し、反映します。

```bash
sudo gitlab-ctl reconfigure
```

GitLab の公式ドキュメントでは、この設定は **すでに 2FA を実施しているプロバイダーだけに設定する**とされています。運用方針を決めたうえで使用してください。

### 注意

- ローカル管理者（`root` など）のパスワードログインは Keycloak を通りません。強いパスワードを設定し、GitLab 側の 2FA も検討してください。
- Git over SSH の鍵、Git over HTTPS や API のアクセストークンには、この MFA は適用されません。

## 関連資料

- [GitLab CE の構築](../61-gitlab-ce/README.md)
- [Keycloak の構築と TLS](./README.md)
- [Grafana の Keycloak 認証](./HOWTO-3-grafana.md)
- [GitLab 公式: OpenID Connect](https://docs.gitlab.com/administration/auth/oidc/)
- [GitLab 公式: OmniAuth 共通設定とアカウント連携](https://docs.gitlab.com/integration/omniauth/)
- [GitLab 公式: 独自 CA 証明書の登録](https://docs.gitlab.com/omnibus/settings/ssl/#install-custom-public-certificates)
