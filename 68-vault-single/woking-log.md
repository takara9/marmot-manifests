## ACMEの有効化

```bash
ubuntu@vault-single:~$ vault login -method=userpass username=takara
Password (will be hidden): 
Success! You are now authenticated. The token information displayed below
is already stored in the token helper. You do NOT need to run "vault login"
again. Future Vault requests will automatically use this token.

Key                    Value
---                    -----
token                  ****
token_accessor         pjwiCLBA4xH11KjHJwmEqmVq
token_duration         768h
token_renewable        true
token_policies         ["admin" "default"]
identity_policies      []
policies               ["admin" "default"]
token_meta_username    takara


ubuntu@vault-single:~$ vault secrets tune \
  -passthrough-request-headers=If-Modified-Since \
  -allowed-response-headers=Last-Modified \
  -allowed-response-headers=Location \
  -allowed-response-headers=Replay-Nonce \
  -allowed-response-headers=Link \
  pki_int
Success! Tuned the secrets engine at: pki_int/


ubuntu@vault-single:~$ vault write pki_int/config/cluster \
  path=https://vault.labo.local:8200/v1/pki_int \
  aia_path=https://vault.labo.local:8200/v1/pki_int
Key         Value
---         -----
aia_path    https://vault.labo.local:8200/v1/pki_int
path        https://vault.labo.local:8200/v1/pki_int


ubuntu@vault-single:~$ vault write pki_int/config/acme \
  enabled=true \
  default_directory_policy="role:labo-local"
Key                              Value
---                              -----
allow_role_ext_key_usage         false
allowed_issuers                  [*]
allowed_roles                    [*]
challenge_excluded_ip_ranges     <nil>
challenge_permitted_ip_ranges    <nil>
default_directory_policy         role:labo-local
dns_resolver                     n/a
eab_policy                       not-required
enabled                          true
max_ttl                          2160h
```
