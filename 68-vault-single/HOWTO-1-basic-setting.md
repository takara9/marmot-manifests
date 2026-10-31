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

