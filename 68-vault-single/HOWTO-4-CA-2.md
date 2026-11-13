```bash
$ vault status
Key                     Value
---                     -----
Seal Type               shamir
Initialized             true
Sealed                  false
Total Shares            5
Threshold               3
Version                 2.2.1
Build Date              2026-10-22T16:27:49Z
Storage Type            raft
Cluster Name            vault-cluster-7aa94cfa
Cluster ID              25d3cb54-5783-e1fc-3f81-d5052987b8f6
Removed From Cluster    false
HA Enabled              true
HA Cluster              https://vault.labo.local:8201
HA Mode                 active
Active Since            2026-11-09T01:42:49.848633349Z
Raft Committed Index    6520
Raft Applied Index      6520
```

## 1. PKI エンジンを有効化し、ルートCAを作成

```bash
$ vault secrets enable pki
Success! Enabled the pki secrets engine at: pki/

$ vault secrets tune -max-lease-ttl=87600h pki
Success! Tuned the secrets engine at: pki/

$ vault write -field=certificate pki/root/generate/internal \
  common_name="labo.local Root CA" issuer_name="root-2026" \
  ttl=87600h > root_ca.crt

$ echo $VAULT_ADDR
https://127.0.0.1:8200
```

```bash
$ vault write pki/config/urls \
  issuing_certificates="$VAULT_ADDR/v1/pki/ca" \
  crl_distribution_points="$VAULT_ADDR/v1/pki/crl"
Key                              Value
---                              -----
crl_distribution_points          [https://127.0.0.1:8200/v1/pki/crl]
delta_crl_distribution_points    []
enable_templating                false
issuing_certificates             [https://127.0.0.1:8200/v1/pki/ca]
ocsp_servers                     []
```

## 2. 中間CAを作成し、ルートで署名

```bash
$ vault secrets enable -path=pki_int pki
Success! Enabled the pki secrets engine at: pki_int/

$ vault secrets tune -max-lease-ttl=43800h pki_int
Success! Tuned the secrets engine at: pki_int/
```

### CSR 生成

```bash
$ vault write -format=json pki_int/intermediate/generate/internal \
  common_name="labo.local Intermediate CA" issuer_name="int-2026" \
  | jq -r '.data.csr' > pki_int.csr
```

### ルートで署名

```bash
$ vault write -format=json pki/root/sign-intermediate \
  csr=@pki_int.csr format=pem_bundle ttl=43800h \
  | jq -r '.data.certificate' > int.cert.pem
```

### 署名済み証明書を中間CAへ登録


```bash
$ vault write pki_int/intermediate/set-signed certificate=@int.cert.pem
WARNING! The following warnings were returned from Vault:

  * This mount hasn't configured any authority information access (AIA)
  fields; this may make it harder for systems to find missing certificates
  in the chain or to validate revocation status of certificates. Consider
  updating /config/urls or the newly generated issuer with this information.

Key                 Value
---                 -----
existing_issuers    <nil>
existing_keys       <nil>
imported_issuers    [895876c1-2054-6bd8-7743-139851efd384 7be7a412-2494-a440-5c6e-1121de34c361]
imported_keys       <nil>
mapping             map[7be7a412-2494-a440-5c6e-1121de34c361: 895876c1-2054-6bd8-7743-139851efd384:3d19ef38-0374-e587-b28f-2774ae9cebea]
```

警告！ Vault から以下の警告が返されました。

このマウントでは、Authority Information Access（AIA）フィールドが設定されていません。そのため、証明書チェーン内の不足している証明書をシステムが見つけたり、証明書の失効状態を検証したりすることが難しくなる可能性があります。

この情報を設定するために、/config/urls または新しく生成された Issuer（発行者）の設定を更新することを検討してください。

補足： AIA（Authority Information Access）は、証明書の発行元（CA）の情報や、中間証明書の取得先などを示すための情報です。

つまり、Vaultの警告は「証明書の検証に必要な情報が不足している可能性があるため、AIAの設定を追加することを推奨する」という意味です。


## 3. ロール（発行ポリシー）を定義

```bash
$ vault write pki_int/roles/labo-local \
  allowed_domains="labo.local" allow_subdomains=true \
  max_ttl=720h key_type=ec key_bits=256
Key                                   Value
---                                   -----
allow_any_name                        false
allow_bare_domains                    false
allow_glob_domains                    false
allow_ip_sans                         true
allow_localhost                       true
allow_subdomains                      true
allow_token_displayname               false
allow_wildcard_certificates           true
allowed_domains                       [labo.local]
allowed_domains_template              false
allowed_other_sans                    []
allowed_serial_numbers                []
allowed_uri_sans                      []
allowed_uri_sans_template             false
allowed_user_ids                      []
basic_constraints_valid_for_non_ca    false
client_flag                           true
cn_validations                        [email hostname]
code_signing_flag                     false
country                               []
csr_extra_names_oids                  []
email_protection_flag                 false
enforce_hostnames                     true
ext_key_usage                         []
ext_key_usage_oids                    []
generate_lease                        false
issuer_ref                            default
key_bits                              256
key_type                              ec
key_usage                             [DigitalSignature KeyAgreement KeyEncipherment]
locality                              []
max_ttl                               720h
no_store                              false
not_after                             n/a
not_before_duration                   30s
organization                          []
ou                                    []
parameter_set                         n/a
policy_identifiers                    []
postal_code                           []
province                              []
require_cn                            true
serial_number_source                  json-csr
server_flag                           true
signature_bits                        256
street_address                        []
ttl                                   0s
use_csr_common_name                   true
use_csr_sans                          true
use_pss                               false
```

## 4. 証明書を発行


返却される `certificate` / `private_key` / `issuing_ca` / `ca_chain` を利用します。

```bash
$ vault write pki_int/issue/labo-local \
  common_name="app.labo.local" ttl=24h
Key                 Value
---                 -----
authority_key_id    8f:64:33:3b:b9:0a:3b:02:71:19:1e:21:b3:12:e5:be:0a:87:07:a6
ca_chain            [-----BEGIN CERTIFICATE-----
MIIDpzCCAo+gAwIBAgIUL5YyawbMYPK1FwYKt2JwHA6flbYwDQYJKoZIhvcNAQEL
BQAwHTEbMBkGA1UEAxMSbGFiby5sb2NhbCBSb290IENBMB4XDTI2MTEwOTA1NTcz
M1oXDTMxMTEwODA1NTgwM1owJTEjMCEGA1UEAxMabGFiby5sb2NhbCBJbnRlcm1l
ZGlhdGUgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC3pT2VSSs8
3uuPyK2TG7u3fWoXG1birKmZfD8tgzHoS9MIANnTz5SdhNTXmxuwvWjiSGZHeNga
XXgz8DlLXrj47pywFgQUIpvkrgFxYQ1stNCj7uuASfU/J4NuGlJSUcDCJIzh1Hwf
2jOviufMvXWRcMR017ybXZXzK9QMxXulwnZBnylDwWARUWERMFTfh8zx25S/+cdA
GEzIEme0cVfz8oWOl5dq3FVHAC4XNhYQI6J4Uj8/d0w+2JnAnGKawuYsuTR6dKVw
X5e7N80TXWxcKDnC46tG9kK+kYn8ZfWQ8idZbEnP32aV2ItXMulmbXEeFSNlEjye
OrUHQOILFPdJAgMBAAGjgdYwgdMwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQF
MAMBAf8wHQYDVR0OBBYEFI9kMzu5CjsCcRkeIbMS5b4KhwemMB8GA1UdIwQYMBaA
FOaPcugHmEXYHYl/FX4Sv+bektLNMDwGCCsGAQUFBwEBBDAwLjAsBggrBgEFBQcw
AoYgaHR0cHM6Ly8xMjcuMC4wLjE6ODIwMC92MS9wa2kvY2EwMgYDVR0fBCswKTAn
oCWgI4YhaHR0cHM6Ly8xMjcuMC4wLjE6ODIwMC92MS9wa2kvY3JsMA0GCSqGSIb3
DQEBCwUAA4IBAQBCPSVbH+pZmq7tHMAYgrdEK1MJTBYcFIyv4lQEE9rQoCHbEUle
hOEoF6CeyLgoeQ033QW9dwXolrGkAyEW6Q9FinDW3egqVWox0ZFuOoMmPuciA/iN
FhENh1VWeXzkevMYnjCdw1USIHiHdSqFuEjsuALKQuRQ2YMSQ3fv30ZsqrGU5GDl
JRrBRap3dZcknIbceSpS/Opdlk/w5r9HCYcxhSx7WJkL6XX8VhpZ9TpPZRllJ3I1
iVBHk8L1hGYcJtFCoQ+87SEjwt15d5RCfphndPkhWjwzBLnCT6NBoXw5ODMCXleK
/9XaJBvVhdenRSm7iSlQU5+/54w3uYIL1hWe
-----END CERTIFICATE----- -----BEGIN CERTIFICATE-----
MIIDKzCCAhOgAwIBAgIUQS18RjVkM6ydOI8PqbZxTH+1amAwDQYJKoZIhvcNAQEL
BQAwHTEbMBkGA1UEAxMSbGFiby5sb2NhbCBSb290IENBMB4XDTI2MTEwOTA1NDU1
MFoXDTM2MTEwNjA1NDYyMFowHTEbMBkGA1UEAxMSbGFiby5sb2NhbCBSb290IENB
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwMV3GCGQmRO2wX00Kc1a
Ee7h44rYaqNMJGPQRhmhmjPmvD2BHERAbzXiwECCIGZZieCWo9KWgb9UrJnqzM1f
PNqtllBxV57OtzgDe+nDFcpPVqvQyCdXD8+ATUx1bj+eDjM3Sg4jcXVu/bhp94nl
amln4IIPiW5YJFv2cb23wK89HWoVuLxfzqsjJ4HomV+t4k1bUrKijoiJb3dThirT
NMrueuV3Q83iL9wdqdpLtJerQZ8Wc8z61bsm78GS3nLuY8WInDhkLb8yuSm4yY9n
gMpG8tu5thf/Fxf5U44tZKxV/HAAyH2PJSep9jTWwa/xzVohW3S+rvBocLKm5Cia
UQIDAQABo2MwYTAOBgNVHQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNV
HQ4EFgQU5o9y6AeYRdgdiX8VfhK/5t6S0s0wHwYDVR0jBBgwFoAU5o9y6AeYRdgd
iX8VfhK/5t6S0s0wDQYJKoZIhvcNAQELBQADggEBAAOG5wy4w7DHjV2rswbdM82E
GF8NRsAulPoEBvA/x0FsIRH5Dp9xCo8KBBL/p67bZwfbu8CSd+u4oFR8unZhok03
DqIvJghmr/sOIzcLO0O6i5reCtc1xUsOt8JpOLgJbBeFN8WdmWB1a4ytAcg1uhiL
NFd9M+Cl5poCY6Gh8ftER8nq+B1jKUtl8t7FrgfxZb/8OqtXQxb6hmKodWKc7xx3
aCry3l94n5ad687mHuw4OLovKsLvPi+e5LxfeCUG9BMp2dLwbyynnWEtAzHrxLvY
nKyRxTmNlTV9fxc5Yk2u2Rio51qK+Fl4bk+6rea0Q5vtfhHHmGmfyZ+32MYIfFM=
-----END CERTIFICATE-----]
certificate         -----BEGIN CERTIFICATE-----
MIICjzCCAXegAwIBAgIURt2wSFrhDPmhHfKj6Cs1cCuo+uwwDQYJKoZIhvcNAQEL
BQAwJTEjMCEGA1UEAxMabGFiby5sb2NhbCBJbnRlcm1lZGlhdGUgQ0EwHhcNMjYx
MTA5MDYwNjAyWhcNMjYxMTEwMDYwNjMyWjAZMRcwFQYDVQQDEw5hcHAubGFiby5s
b2NhbDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABOCP+dtOA2I0L2R1MaclzAV1
FbVN/475mDs3KAyFwhYTbd/UA21uOl2ZiXRQDm/hgO65VzhTWFZfr2HVv+v7X16j
gY0wgYowDgYDVR0PAQH/BAQDAgOoMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEF
BQcDAjAdBgNVHQ4EFgQUsbs89YLven92DvS8JGAZsOVuixIwHwYDVR0jBBgwFoAU
j2QzO7kKOwJxGR4hsxLlvgqHB6YwGQYDVR0RBBIwEIIOYXBwLmxhYm8ubG9jYWww
DQYJKoZIhvcNAQELBQADggEBADzLs4e/MXb/Tkwu+Pnr+ua+C//lzPB/LFv7DOOo
erpLSPUBqSdI+na2qsnI/DOvCvJfTtZiJ7m1u04wG4H7YOu46kk6abeT12Uk8woY
yyOO0xQAxMxV4kEFk62R0B7wa/oKCkIL8W9TQdDCZCyiL5VCKz7fG5JGNctpBoOa
Sh5E/hMY5Eg0B+UjK1NNJDMZnIoh9ebC+Anq2BvFDeUE5L1mv0wUwRMxfVV6ey2h
GzfiXc3ERRQNrYN15YbjUsAQsAuWbDiI2+stUCPwLfue6+kv4XM8tZZ6mBM2M4Dg
sJcJm0W4bZnGZ0MEeVj+eM1zxoacJmM/TXbylJaaVM8EiDY=
-----END CERTIFICATE-----
expiration          1794290792
issuing_ca          -----BEGIN CERTIFICATE-----
MIIDpzCCAo+gAwIBAgIUL5YyawbMYPK1FwYKt2JwHA6flbYwDQYJKoZIhvcNAQEL
BQAwHTEbMBkGA1UEAxMSbGFiby5sb2NhbCBSb290IENBMB4XDTI2MTEwOTA1NTcz
M1oXDTMxMTEwODA1NTgwM1owJTEjMCEGA1UEAxMabGFiby5sb2NhbCBJbnRlcm1l
ZGlhdGUgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC3pT2VSSs8
3uuPyK2TG7u3fWoXG1birKmZfD8tgzHoS9MIANnTz5SdhNTXmxuwvWjiSGZHeNga
XXgz8DlLXrj47pywFgQUIpvkrgFxYQ1stNCj7uuASfU/J4NuGlJSUcDCJIzh1Hwf
2jOviufMvXWRcMR017ybXZXzK9QMxXulwnZBnylDwWARUWERMFTfh8zx25S/+cdA
GEzIEme0cVfz8oWOl5dq3FVHAC4XNhYQI6J4Uj8/d0w+2JnAnGKawuYsuTR6dKVw
X5e7N80TXWxcKDnC46tG9kK+kYn8ZfWQ8idZbEnP32aV2ItXMulmbXEeFSNlEjye
OrUHQOILFPdJAgMBAAGjgdYwgdMwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQF
MAMBAf8wHQYDVR0OBBYEFI9kMzu5CjsCcRkeIbMS5b4KhwemMB8GA1UdIwQYMBaA
FOaPcugHmEXYHYl/FX4Sv+bektLNMDwGCCsGAQUFBwEBBDAwLjAsBggrBgEFBQcw
AoYgaHR0cHM6Ly8xMjcuMC4wLjE6ODIwMC92MS9wa2kvY2EwMgYDVR0fBCswKTAn
oCWgI4YhaHR0cHM6Ly8xMjcuMC4wLjE6ODIwMC92MS9wa2kvY3JsMA0GCSqGSIb3
DQEBCwUAA4IBAQBCPSVbH+pZmq7tHMAYgrdEK1MJTBYcFIyv4lQEE9rQoCHbEUle
hOEoF6CeyLgoeQ033QW9dwXolrGkAyEW6Q9FinDW3egqVWox0ZFuOoMmPuciA/iN
FhENh1VWeXzkevMYnjCdw1USIHiHdSqFuEjsuALKQuRQ2YMSQ3fv30ZsqrGU5GDl
JRrBRap3dZcknIbceSpS/Opdlk/w5r9HCYcxhSx7WJkL6XX8VhpZ9TpPZRllJ3I1
iVBHk8L1hGYcJtFCoQ+87SEjwt15d5RCfphndPkhWjwzBLnCT6NBoXw5ODMCXleK
/9XaJBvVhdenRSm7iSlQU5+/54w3uYIL1hWe
-----END CERTIFICATE-----
private_key         -----BEGIN EC PRIVATE KEY-----
MHcCAQEEIJWKwPhqHooUGMFiQkUroMnedjNZ00IDH8L8X4akYGwMoAoGCCqGSM49
AwEHoUQDQgAE4I/5204DYjQvZHUxpyXMBXUVtU3/jvmYOzcoDIXCFhNt39QDbW46
XZmJdFAOb+GA7rlXOFNYVl+vYdW/6/tfXg==
-----END EC PRIVATE KEY-----
private_key_type    ec
serial_number       46:dd:b0:48:5a:e1:0c:f9:a1:1d:f2:a3:e8:2b:35:70:2b:a8:fa:ec
```


## 発行した証明書を Nginx に設定する手順


1. 発行結果をファイルに保存

JSON で受け取り、jq で証明書・秘密鍵・チェーンを取り出します。

```bash
vault write -format=json pki_int/issue/labo-local \
  common_name="app.labo.local" ttl=720h > issued.json

# サーバー証明書 + 中間CA証明書（チェーン）を連結
jq -r '.data.certificate, (.data.ca_chain[])' issued.json \
  | sudo tee /etc/nginx/ssl/app.labo.local.crt > /dev/null

# 秘密鍵
jq -r '.data.private_key' issued.json \
  | sudo tee /etc/nginx/ssl/app.labo.local.key > /dev/null

sudo chmod 600 /etc/nginx/ssl/app.labo.local.key
sudo chown root:root /etc/nginx/ssl/app.labo.local.*
rm issued.json   # 秘密鍵入りのため残さない
```

## 動作検証

```bash
openssl s_client -connect app.labo.local:443 -servername app.labo.local </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates

curl --cacert root_ca.crt https://app.labo.local/
```

クライアントにルートCAを信頼させる必要があります。 root_ca.crt を OS の信頼ストアへ登録するか、--cacert で指定します。Ubuntu の場合は次のとおりです。





**5. アクセス制御（最小権限）**

