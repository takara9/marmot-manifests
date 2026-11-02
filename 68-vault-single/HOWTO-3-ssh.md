## SSH-KEYペアを vault に保存して、必要な時に秘密鍵を取り出して接続

鍵ペアの生成
```bash
ssh-keygen -t ed25519 -f ./id_ed25519 -N "" -C "app-deploy"
```

鍵のペアを vault に保存します。この時のキーは、`ssh/app-deploy`になる。
```bash
vault kv put secret/ssh/app-deploy \
  private_key=@id_ed25519 \
  public_key=@id_ed25519.pub
```

元ファイルを安全に削除します。
```bash
shred -u id_ed25519 id_ed25519.pub    
```

秘密鍵を、vault からキーを指定して、ファイルに書き出して使用。
使用後に消去します。
```bash
vault kv get -field=private_key secret/ssh/app-deploy > /tmp/key && chmod 600 /tmp/key
```


## SSH エンジン（CA 署名方式）
サーバーは Vault の CA 公開鍵だけを信頼し、サーバーごとの authorized_keys 管理が不要になります。
証明書は短寿命（例: 30 分〜数時間）なので、失効管理もほぼ不要です。

Vaultの SSH エンジンを有効化して、CA局を作成して、ロール admin を設定します。
```bash
$ vault secrets enable -path=ssh-client-signer ssh
Success! Enabled the ssh secrets engine at: ssh-client-signer/
```

このコマンドで、CA局の鍵ペアを作成して、公開鍵が取得できます。
```bash
$ vault write ssh-client-signer/config/ca generate_signing_key=true
Key           Value
---           -----
public_key    ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCeXgYrHPzu8ctdZBz++YUD5yPL4IPrlDlTNQG4al7mTSefkY2XHYHo/ECrfEiJvjTFx/quAn8PPW0miOoRUDITsruds/MQnWzPTMK9dcBqQ0BYYV9eri5B2PKzqx9jaFFeYCpFNfS1/Ur17/F/RpunS5uFdNjoXol4QLd92a2Msaj6uXY1pRnCc1KJDhp4vF4F2nn2E6Tlj6xQ0h4N75z44qX3OfVR9xRAcHz9JO/lov9CPHHye7lZJQQAyiJ+cwiCaYMRtep2WQHdX1po255mezGr/a4wNClLKVfi2jH2dQKZYsSDuCjNJ+EH/4vwu0NBFALwAJ4rHvwkQ3WBBA7vnglGQhRi/dfeqSHKfuic2JgQM8WP1eitciARvFDe9Y10ZmJa35OIbVQWF43GjMzj0/tBsbZCQIMEwKtROv3rV5BOhiUEbp7Q0yKbyBg5rM3ljJx1gtWKG06d7sMfdV/vMiimH7MeupN9INpxxvzACoxAql+Y1PvfKsTx61OUyFcy3TW49DilRINvNrXRZwVfd0669tQZz+pHeoJErP6plftBP2xUUXmHoOMN9GvMzNnIk2fa5pvSsKlKSfB9McNU1vxu77BWmuPLLkH2iMAGz0I5w7aoOwOKttoPhgnIMpc01VlovwHxv7bXuBAfGu94V4LJHm8Ckz2OSGULVIvmmQ==
```

ロールの `allowed_users` には、**接続先サーバーに実在するログインユーザー名**を列挙します。
ここに無いユーザー宛の証明書は発行できず、接続は `Permission denied` になります。
```bash
$ vault write ssh-client-signer/roles/admin -<<'EOF'
{
  "key_type": "ca",
  "allow_user_certificates": true,
  "allowed_users": "ubuntu,takara,almalinux,debian",
  "default_user": "almalinux",
  "default_extensions": { "permit-pty": "" },
  "ttl": "30m0s"
}
EOF
Success! Data written to: ssh-client-signer/roles/admin
```




## 対象サーバー側
CA 公開鍵を配置し sshd に登録します。
自己署名TLS証明書のため `curl -k` が必要です。
```bash
sudo curl -k -o /etc/ssh/trusted-user-ca-keys.pem \
  https://vault.labo.local:8200/v1/ssh-client-signer/public_key
echo 'TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem' | sudo tee /etc/ssh/sshd_config.d/50-vault-ca.conf
sudo sshd -t && sudo systemctl restart sshd
```

> `TrustedUserCAKeys` の登録を忘れると、正しい証明書でも `Permission denied (publickey,...)` になります。
> 登録済みかは `sudo grep -r TrustedUserCAKeys /etc/ssh/sshd_config /etc/ssh/sshd_config.d/` で確認できます。


## クライアント側

まず、クライアント上で自分専用の鍵ペアを生成します。
CA署名方式では、秘密鍵は Vault に保存せず、クライアントのローカルで生成してそのまま手元に置きます。
Vault に渡すのは公開鍵だけで、Vault はそれに署名した証明書を返します。
```bash
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N ""
```

> すでに `~/.ssh/id_ed25519` が存在する場合は、生成をスキップしてその鍵をそのまま使えます。

公開鍵に署名して接続します。
`valid_principals` に **接続先のログインユーザー名** を指定します(省略時はロールの `default_user` になります)。
```bash
vault write -field=signed_key ssh-client-signer/sign/admin \
  public_key=@$HOME/.ssh/id_ed25519.pub \
  valid_principals=almalinux > ~/.ssh/id_ed25519-cert.pub
ssh -i ~/.ssh/id_ed25519 -i ~/.ssh/id_ed25519-cert.pub almalinux@ssh-server
```

接続できない場合は、証明書の内容(特に `Principals` と `Valid` の有効期限)を確認します。
`Principals` がログインユーザー名と一致していないと `Permission denied` になります。
```bash
ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub
```