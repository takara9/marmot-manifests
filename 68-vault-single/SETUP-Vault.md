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


## 基本設定

```console
$ whoami
ubuntu

$ vault secrets enable -path=secret kv-v2
Success! Enabled the kv-v2 secrets engine at: secret/

$ vault policy write admin - <<'EOF'
path "*" { capabilities = ["create","read","update","delete","list","sudo"] }
EOF

$ vault auth enable userpass

$ vault write auth/userpass/users/takara password='******' policies=admin

$ vault login -method=userpass username=takara
```