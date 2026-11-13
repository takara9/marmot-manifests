# Vault のセットアップ

## 自動アンシール（封印解除）

```console
systemctl start vault-unseal
systemctl status vault-unseal
```

```console
vault status
```

## Vaultへのログイン

root の tokenをセットします。
```console
vault login
```


## 監査ログのディレクトリ作成と有効化

```
sudo mkdir -p /var/log/vault && sudo chown vault:vault /var/log/vault
vault audit enable file file_path=/var/log/vault/audit.log
```


## パスワードでのログインを許可して、ユーザーを追加

設定は、Ubuntu Linux のクラウドイメージのビルトイン・アカウントの ubuntu で操作するものとして進めます。
```console
$ whoami
ubuntu
```

シークレットエンジン kv-v2 を有効化します。
シークレットエンジンは、secret/ から始まるパスの下に設定されるので、覚えておいてください。
```console
$ vault secrets enable -path=secret kv-v2
Success! Enabled the kv-v2 secrets engine at: secret/
```

ポリシーと呼ばれる認可を、ポリシー名 admin に設定します。admin は、全ての操作が可能な管理者権限なのでで、path は、全てを意味する "*" として設定します。
```console
$ vault policy write admin - <<'EOF'
path "*" { capabilities = ["create","read","update","delete","list","sudo"] }
EOF
```

パスワードにより認証を有効にします。
```console
$ vault auth enable userpass
```

ユーザー takara を 認可名 admin と対応付けて、パスワードを設定します。
```console
$ vault write auth/userpass/users/takara password='******' policies=admin
```

設定したパスワードでログインできるのか確認します。
```console
$ vault login -method=userpass username=takara
```

---

# root トークンの無効化

Vault の root トークンを無効化（失効）する手順です。この HOWTO の流れ（admin ポリシーの `takara` ユーザーを作成済み）を前提にしています。

## 手順

1. 別の管理者でログインできることを先に確認します。root を失効すると、他に管理手段がない場合に締め出されます。

```console
$ vault login -method=userpass username=takara
```

2. root トークンを失効させます。

```console
# root トークンでログインした状態で、自分自身を失効
$ vault token revoke -self

# 別の管理者でログインしている場合は、root トークンの値を指定
$ vault token revoke <root-token>
```

3. 失効したか確認します。`permission denied` や `bad token` になれば失効済みです。

```console
$ VAULT_TOKEN=<root-token> vault token lookup
```

4. 履歴や環境変数に root トークンが残っていれば消します。

```console
$ unset VAULT_TOKEN
$ rm -f ~/.vault-token
$ history -c
```

`vault operator init` の出力ファイルや自動アンシール用の設定に root トークンを保存している場合は、そこからも削除してください。

## 必要になったときに root トークンを再生成する

root を失効しても、アンシールキー（のしきい値分）があれば再発行できます。

```console
$ vault operator generate-root -init       # OTP と nonce が表示される
$ vault operator generate-root             # アンシールキーを、しきい値の数だけ入力
$ vault operator generate-root -decode=<encoded-token> -otp=<otp>
```

使い終わったら、すぐに `vault token revoke -self` で失効させます。

## 注意点

- 失効は、子トークンも道連れにします。root トークンから作った子トークンは、同時に失効します。
- 監査ログの有効化（`vault audit enable`）や、`admin` ポリシーの作成を済ませてから失効してください。
- `admin` ポリシーは `*` に `sudo` を許可しており、実質 root 相当です。`takara` のパスワードは強固にし、日常運用は権限を絞ったポリシーに分けることを勧めます。
- 自動アンシールの仕組みが root トークンに依存していないか、失効前に確認してください。`vault-unseal` はアンシールキーを使うだけなので、通常は影響しません。

HOWTO-1 に追記が必要なら、反映します。