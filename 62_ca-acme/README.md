# CA

仮想マシンの起動と初期チェック
```console
ubuntu@ca:~$ hostnamectl
 Static hostname: ca
       Icon name: computer-vm
         Chassis: vm 🖴
      Machine ID: ac0639773b824922b0e9863ce24cf763
         Boot ID: 8f3eb5fa39c447918399049fbc88b051
  Virtualization: kvm
Operating System: Ubuntu 26.04.1 LTS
          Kernel: Linux 7.0.0-31-generic
    Architecture: x86-64
 Hardware Vendor: QEMU
  Hardware Model: Standard PC _Q35 + ICH9, 2009_
Hardware Version: pc-q35-4.2
Firmware Version: 1.16.3-debian-1.16.3-2
   Firmware Date: Tue 2014-04-01
    Firmware Age: 12y 5month 4w

ubuntu@ca:~$ ip -br addr
lo               UNKNOWN        127.0.0.1/8 ::1/128
enp1s0           UP             192.168.1.250/24 fe80::dc9b:b4ff:feaf:61aa/64
enp2s0           UP             10.245.0.7/16 fe80::6887:8ff:feb5:5751/64

ubuntu@ca:~$ dig +short ca.labo.local @192.168.1.9
192.168.1.250

ubuntu@ca:~$ timedatectl
               Local time: Mon 2026-09-28 07:01:13 UTC
           Universal time: Mon 2026-09-28 07:01:13 UTC
                 RTC time: Mon 2026-09-28 07:01:12
                Time zone: Etc/UTC (UTC, +0000)
System clock synchronized: yes
              NTP service: active
          RTC in local TZ: no

```

ホスト名をFQDN形式に設定
```console
ubuntu@ca:~$ hostname
ca
ubuntu@ca:~$ sudo hostnamectl set-hostname ca.labo.local
ubuntu@ca:~$ hostname
ca.labo.local
```

step-cli, step-caのリポジトリを登録
```console
sudo install -d -m 0755 /etc/apt/keyrings
sudo curl -fsSL https://packages.smallstep.com/keys/apt/repo-signing-key.gpg \
  -o /etc/apt/keyrings/smallstep.asc
sudo chmod 0644 /etc/apt/keyrings/smallstep.asc
```

正当性チェック
```console
gpg --show-keys --with-subkey-fingerprints /etc/apt/keyrings/smallstep.asc
```

step-cli/step-caのインストール
```console
sudo apt update
sudo apt install step-cli step-ca
```

バージョンチェックなど
```console
ubuntu@ca:~$ step version
Smallstep CLI/0.31.0 (linux/amd64)
Release Date: 2026-09-25T19:53:51Z

ubuntu@ca:~$ step-ca version
Smallstep CA/0.30.2 (linux/amd64)
Release Date: 2026-03-23T00:18:00Z

ubuntu@ca:~$ command -v step
/usr/bin/step

ubuntu@ca:~$ command -v step-ca
/usr/bin/step-ca
```

ルートCAで生成した証明書をチェック

```console
ubuntu@ca:~/pki-transfer$ openssl x509 -in ./Takara_Home_Lab_Root_CA_2026.crt -noout -subject -issuer -dates -fingerprin
t -sha256
subject=C=JP, ST=Chiba, L=Inzai, O=Takara Home Lab, CN=Takara Home Lab Root CA 2026
issuer=C=JP, ST=Chiba, L=Inzai, O=Takara Home Lab, CN=Takara Home Lab Root CA 2026
notBefore=Sep 28 06:06:00 2026 GMT
notAfter=Sep 28 06:06:00 2036 GMT
sha256 Fingerprint=DE:15:03:B5:CB:91:1E:35:E6:DD:97:3B:9A:A2:61:93:07:28:37:19:15:3C:DE:C1:F8:05:F5:7C:D6:EA:23:BE
```


## step-ca

step-caを初期化
```console
step ca init \
  --name "Takara Home Lab PKI" \
  --dns ca.labo.local \
  --address :443 \
  --provisioner "admin-jwk" \
  --acme
```

実行例
```console
ubuntu@ca:~$ step ca init \
  --name "Takara Home Lab PKI" \
  --dns ca.labo.local \
  --address :443 \
  --provisioner "admin-jwk" \
  --acme
✔ Deployment Type: Standalone
Choose a password for your CA keys and first provisioner.
✔ [leave empty and we'll generate one]:
✔ Password: DTdDbkyb9CnDyU2tIoYslU1jd0vbtriA

Generating root certificate... done!
Generating intermediate certificate... done!

✔ Root certificate: /home/ubuntu/.step/certs/root_ca.crt
✔ Root private key: /home/ubuntu/.step/secrets/root_ca_key
✔ Root fingerprint: b216ad74859d16175eecef58fe8ac7fdc7b9e44600e27545d4cbb07efce0a9ec
✔ Intermediate certificate: /home/ubuntu/.step/certs/intermediate_ca.crt
✔ Intermediate private key: /home/ubuntu/.step/secrets/intermediate_ca_key
✔ Database folder: /home/ubuntu/.step/db
✔ Default configuration: /home/ubuntu/.step/config/defaults.json
✔ Certificate Authority configuration: /home/ubuntu/.step/config/ca.json

Your PKI is ready to go. To generate certificates for individual services see 'step help ca'.

FEEDBACK 😍 🍻
  The step utility is not instrumented for usage statistics. It does not phone
  home. But your feedback is extremely valuable. Any information you can provide
  regarding how you’re using `step` helps. Please send us a sentence or two,
  good or bad at feedback@smallstep.com or join GitHub Discussions
  https://github.com/smallstep/certificates/discussions and our Discord
  https://u.step.sm/discord.
```

パスを表示
```console
ubuntu@ca:~$ step path
/home/ubuntu/.step
ubuntu@ca:~$ find "$(step path)" -maxdepth 2 -type f -printf '%M %p\n'
-rw-r--r-- /home/ubuntu/.step/config/defaults.json
-rw-r--r-- /home/ubuntu/.step/config/ca.json
-rw------- /home/ubuntu/.step/certs/intermediate_ca.crt
-rw------- /home/ubuntu/.step/certs/root_ca.crt
-rw------- /home/ubuntu/.step/secrets/root_ca_key
-rw------- /home/ubuntu/.step/secrets/intermediate_ca_key
```

自サイトで発行したルート証明書、中間証明書のセットを削除
```console
ubuntu@ca:~$ shred -u /home/ubuntu/.step/secrets/root_ca_key
ubuntu@ca:~$ shred -u /home/ubuntu/.step/secrets/intermediate_ca_key
ubuntu@ca:~$ rm /home/ubuntu/.step/certs/root_ca.crt
ubuntu@ca:~$ rm /home/ubuntu/.step/certs/intermediate_ca.crt
```

ルートCAで発行した証明書で置き換え
```console
install -m 0644 \
  /home/ubuntu/pki-transfer/Takara_Home_Lab_Root_CA_2026.crt \
  /home/ubuntu/.step/certs/root_ca.crt
```

確認コマンド
```console
step certificate inspect /home/ubuntu/.step/certs/root_ca.crt
```

確認コマンドの実行例
```console
ubuntu@ca:~$ step certificate inspect /home/ubuntu/.step/certs/root_ca.crt
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: 2569147661667404125 (0x23a7722fb536095d)
        Signature Algorithm: SHA256-RSA
        Issuer: C=JP,ST=Chiba,L=Inzai,O=Takara Home Lab,CN=Takara Home Lab Root CA 2026
        Validity
            Not Before: Sep 28 06:06:00 2026 UTC
            Not After : Sep 28 06:06:00 2036 UTC
        Subject: C=JP,ST=Chiba,L=Inzai,O=Takara Home Lab,CN=Takara Home Lab Root CA 2026
        Subject Public Key Info:
            Public Key Algorithm: RSA
                Public-Key: (4096 bit)
                Modulus:
                    ca:25:8b:06:53:44:10:1f:61:ea:27:72:68:1a:37:
                    6d:60:d8:f4:6a:91:59:b0:d9:f3:78:82:f1:01:42:
                    b2:48:40:0a:54:eb:eb:3c:97:12:ad:e1:f8:94:2d:
                    63:0a:0f:42:d0:03:7c:3a:4b:de:ec:24:ec:bf:78:
                    86:86:8c:dc:b1:4f:10:4b:d0:1a:17:ec:5c:7b:38:
                    f6:89:e9:5e:3e:3c:78:d6:63:38:b8:bd:89:55:06:
                    a0:31:ef:e3:ce:c3:2b:a9:cb:eb:29:5a:01:b4:57:
                    e3:b5:bf:b6:c4:00:64:5f:43:bf:54:73:72:7c:bb:
                    31:36:84:78:6e:b4:ac:42:4a:95:81:60:c2:4d:c0:
                    1b:01:e4:70:18:8a:cf:6e:18:24:d7:2b:22:67:db:
                    d6:3a:8d:c6:b4:2c:f4:66:0f:60:b2:89:d1:be:02:
                    05:ba:ba:76:10:40:d0:3d:b7:9b:c9:90:a3:1f:09:
                    c7:99:0f:a0:09:a3:ba:65:14:f6:bb:39:37:13:de:
                    17:fb:6f:81:c6:8a:be:38:5f:89:f1:72:8d:eb:54:
                    46:5f:b8:93:52:c7:08:22:e8:c1:62:d2:99:7f:b3:
                    7e:29:38:cb:33:52:ed:d7:54:24:66:6c:c2:74:24:
                    b1:6f:2f:0a:ad:7e:1e:ef:c3:06:e6:1f:c3:b0:5e:
                    72:1b:ac:83:65:99:57:23:88:9a:0a:d2:56:9c:c1:
                    4c:d3:ab:bc:4a:1a:0b:1a:14:cc:57:7f:5d:b4:f3:
                    e9:52:c0:0a:2b:44:e9:06:ec:3b:32:c1:10:b6:46:
                    6b:9e:33:17:d5:16:a8:a1:82:c4:ae:05:e6:3b:29:
                    08:20:aa:a7:b3:7f:7f:82:16:f9:3a:e9:3b:f5:2b:
                    4c:1e:f7:2a:b2:1a:76:af:ad:d4:d8:7a:36:64:44:
                    6f:a6:c5:53:11:89:e0:b2:12:ca:66:a1:51:b3:e8:
                    16:79:65:e7:24:3f:ae:c8:b5:e3:a4:a1:ef:3a:29:
                    e7:3e:e6:03:6a:cf:24:e4:96:e7:d6:fe:bf:aa:e8:
                    54:83:a4:5a:e3:45:dd:a2:e0:65:44:4f:eb:5a:20:
                    a2:74:bc:72:15:b2:fa:5d:d6:17:53:b7:b8:b1:53:
                    f2:64:b7:57:8a:0b:35:4c:73:31:90:71:a7:55:56:
                    47:bc:13:1f:4b:64:5c:1b:30:d4:89:5d:d4:bd:b1:
                    54:14:16:89:34:af:bc:dc:8a:5c:5c:f5:ac:78:45:
                    96:3c:ae:2c:ad:c8:d1:c9:4f:d1:36:c7:f2:bc:43:
                    a8:af:a1:7d:41:c8:6c:61:68:8f:28:67:2c:eb:0a:
                    a8:e9:5e:3a:5b:26:1d:78:41:69:9a:29:b1:f6:84:
                    d4:71
                Exponent: 65537 (0x10001)
        X509v3 extensions:
            X509v3 Basic Constraints: critical
                CA:TRUE
            X509v3 Subject Key Identifier:
                32:15:08:BD:7E:CF:D4:53:29:79:9A:62:2E:2F:E2:26:37:99:6D:63
            X509v3 Key Usage:
                Certificate Sign, CRL Sign
    Signature Algorithm: SHA256-RSA
         a6:e6:bd:b1:26:e0:f2:01:6d:54:73:e2:7c:99:5e:1f:f2:94:
         59:73:19:ae:ef:f1:5d:39:7e:c4:11:29:de:fe:54:e4:73:6c:
         82:85:d0:b0:40:af:14:ea:d2:0a:c7:cf:7d:ee:04:0c:e1:a1:
         4c:27:12:16:2a:ea:bd:f2:2a:88:08:25:f8:64:36:c9:0c:ba:
         f0:f4:b8:a0:fd:33:1c:b8:21:bc:00:6e:1e:ac:02:1c:7e:90:
         9e:15:3e:e7:5f:05:87:c0:32:58:ee:e9:2f:9a:c3:3b:3a:c9:
         7c:67:67:a4:89:83:cf:0a:94:0c:0d:a3:bf:ef:08:b0:4a:c8:
         81:b7:33:24:f8:ed:c1:bc:70:a7:fe:73:19:57:76:12:3a:f9:
         ea:99:02:ab:a0:6a:4c:28:29:86:ff:ee:1e:52:d2:d8:2b:f7:
         22:a8:b2:57:d1:ce:14:4b:e7:49:8a:44:72:5c:57:e3:86:ae:
         ad:ac:af:6e:2a:4d:df:53:3b:23:00:11:57:1f:26:7a:a7:a7:
         31:c9:23:3a:08:d7:c3:78:57:47:e6:c4:98:e5:29:c4:06:c4:
         7c:76:32:02:70:71:a5:8c:9c:8f:f1:cd:a7:c4:0c:f7:0b:00:
         1a:3b:fb:47:8e:b0:75:2a:5d:f1:a8:d1:0e:ec:b2:97:9d:f9:
         c3:6d:57:4e:1f:28:cb:cc:0d:e0:c6:33:1c:94:22:be:8d:93:
         73:4b:99:37:76:18:6b:d1:47:a6:95:48:ce:89:4c:cb:37:93:
         90:c9:12:1d:a4:85:50:3e:41:53:19:c0:d6:07:49:72:5c:6a:
         f3:56:de:02:83:5a:12:27:88:1c:15:b9:87:e7:e0:24:9d:9f:
         3c:10:f1:9a:64:b2:ab:82:45:44:6b:c5:60:bb:22:d9:bc:33:
         21:fe:43:ae:f6:e7:d0:08:87:78:0e:1c:62:7f:a3:f0:ad:84:
         95:0b:69:e0:9c:8a:93:57:1f:83:c8:89:af:73:e6:4f:e8:82:
         ed:28:19:33:26:3d:3d:90:15:87:cf:69:f5:10:1b:01:31:07:
         d6:41:b8:93:9a:1d:53:31:ea:cd:0a:f2:79:ad:4e:18:a1:5e:
         b0:e4:ee:d5:5d:95:2f:ff:48:83:95:7b:97:ba:74:0f:9b:89:
         3a:73:e7:e5:be:3b:c2:e8:c1:59:fc:2f:bb:e3:a5:4d:dd:ac:
         71:f4:d7:e7:5d:0b:a8:9b:62:10:ce:e3:81:20:bc:19:2b:ad:
         87:93:04:cf:94:23:c9:e6:ab:da:15:81:1e:cd:5c:7f:85:80:
         7f:fb:22:18:8f:8d:fb:55:c3:34:7b:34:4b:8e:5d:73:b3:79:
         c6:a1:36:7b:c5:f3:32:da
ubuntu@ca:~$
```

##  Intermediate秘密鍵とCSRを作成

中間認証局の鍵作成とCSR作成のコマンド
```console
step certificate create \
  "Takara Home Lab Issuing CA 2026" \
  /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.csr \
  /home/ubuntu/.step/secrets/intermediate_ca_key \
  --csr \
  --kty RSA \
  --size 4096
```

実行例、パスワードを求められるのでインプットする。
```console
ubuntu@ca:~$ step certificate create \
  "Takara Home Lab Issuing CA 2026" \
  /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.csr \
  /home/ubuntu/.step/secrets/intermediate_ca_key \
  --csr \
  --kty RSA \
  --size 4096
Please enter the password to encrypt the private key: pilot00
Your certificate signing request has been saved in /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.csr.
Your private key has been saved in /home/ubuntu/.step/secrets/intermediate_ca_key.
```

確認コマンド
```console
openssl req \
  -in /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.csr \
  -noout \
  -subject \
  -text
```

実行例
```console
ubuntu@ca:~$ openssl req \
  -in /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.csr \
  -noout \
  -subject \
  -text
Certificate Request:
    Data:
        Version: 1 (0x0)
        Subject: CN=Takara Home Lab Issuing CA 2026
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (4096 bit)
                Modulus:
                    00:d4:ba:23:5e:32:20:5f:f4:a2:c8:a4:f1:96:f1:
                    2c:6f:81:6f:90:ac:d5:5d:01:00:aa:51:65:9f:ed:
                    17:e8:a4:73:d2:1c:5b:65:3c:55:c4:4a:b9:ae:5a:
                    7a:33:3d:13:af:31:10:63:cd:99:fe:4d:05:08:93:
                    e3:6d:8c:f9:36:96:ff:60:7a:bb:e3:ab:2f:21:7c:
                    3e:16:32:b9:13:c1:cc:7f:52:d0:dc:ea:dd:c8:03:
                    ab:06:ef:3d:6a:63:3b:87:ad:37:d9:30:e7:d8:98:
                    e8:f1:e8:60:25:75:84:22:4b:79:94:c6:ce:a7:77:
                    4d:db:07:28:8f:0f:14:c8:30:d1:5c:0d:8c:12:1e:
                    a6:72:1d:e7:da:2e:a8:4f:de:b2:20:12:d9:4c:4e:
                    bf:2d:92:5e:f1:cc:a6:c2:da:0c:38:36:5a:ba:ca:
                    d8:89:7f:c0:5e:8c:a4:56:a5:b4:76:82:fc:9e:26:
                    9a:e6:3c:90:40:0d:3c:d1:b2:6c:31:48:bc:92:ee:
                    d4:34:cb:93:db:2b:a5:78:84:da:8c:ce:09:60:f0:
                    b5:e3:ee:16:d1:c7:a9:5f:da:d4:74:22:02:1e:c2:
                    03:fc:56:19:4c:66:f7:ec:d3:5f:a9:6d:37:d1:7c:
                    f5:86:c4:ae:2c:68:83:a1:74:9e:2d:09:e4:62:a3:
                    ca:da:0f:94:c8:aa:ae:06:50:d7:17:85:1d:05:c8:
                    4a:dd:6c:07:02:c6:1b:83:ee:31:5f:cf:91:65:15:
                    49:c1:6f:2d:f4:f6:88:9c:ba:53:f5:43:7d:33:ad:
                    d9:eb:f7:62:c1:9c:02:86:0b:a8:a4:5c:c9:e9:c3:
                    62:e5:df:f6:4f:ec:6e:fa:9f:47:e9:c1:41:0a:f6:
                    f2:b9:9a:66:fb:8f:50:81:da:73:a1:a3:c1:9e:82:
                    94:68:91:40:1d:1a:ed:23:6a:d9:43:c3:55:b2:46:
                    b1:e9:ac:7f:97:94:fb:68:45:db:52:a2:c2:24:0c:
                    d1:e0:7e:c8:ed:6e:7c:01:61:ea:a7:b0:0e:dc:f1:
                    f2:cf:0f:d0:df:cd:16:c8:55:bd:99:d0:aa:90:d5:
                    f9:e9:1a:88:88:98:88:23:45:9f:f7:66:ba:39:2f:
                    18:e8:86:e0:bb:4b:3b:04:8b:9e:5a:19:02:d9:91:
                    bd:1a:15:e7:ba:a3:94:ce:b4:99:00:9b:5b:fa:a3:
                    69:a5:1c:07:1e:f5:f7:27:02:6a:79:c0:38:59:5b:
                    36:5d:55:90:bc:0b:02:e8:85:5e:3e:22:d2:a4:48:
                    98:a3:30:81:9d:f6:a6:77:9c:f5:d9:f3:31:68:6e:
                    0d:cd:f7:f7:5e:02:02:9f:4c:30:60:cc:c6:0c:00:
                    d3:aa:61
                Exponent: 65537 (0x10001)
        Attributes:
            Requested Extensions:
                X509v3 Subject Alternative Name:
                    DNS:Takara Home Lab Issuing CA 2026
    Signature Algorithm: sha256WithRSAEncryption
    Signature Value:
        44:bb:e6:b0:79:0a:d7:88:ac:62:45:53:cd:f6:1e:a1:b1:02:
        2f:16:a6:37:f1:fc:24:19:74:24:be:21:ea:01:5f:ba:2c:3a:
        12:13:74:f5:1a:a6:97:77:4a:28:02:39:a8:ad:4a:81:ee:e5:
        ee:bb:23:fd:08:d0:62:cd:ae:5a:ea:52:47:6e:e0:d1:d8:1e:
        37:40:6c:7c:a6:20:f1:7f:ec:be:b4:fd:87:04:dd:4d:97:98:
        1b:34:1d:94:12:19:77:18:55:d9:e1:4b:02:3f:ac:5e:5c:a9:
        07:07:97:0f:df:6c:84:49:03:6f:e8:f8:c7:9b:dc:1e:05:f6:
        92:61:c6:70:05:42:5c:2b:af:39:74:6b:79:d6:b2:06:cd:a3:
        59:84:c4:ea:e3:57:fc:e7:2f:73:55:b2:74:42:99:0c:38:ec:
        b4:dc:5a:b1:f0:e3:a0:77:01:73:14:1c:c2:d4:fa:57:ec:6c:
        c3:14:30:b3:87:13:e4:2b:12:ff:e6:0c:83:80:35:74:ea:25:
        62:8d:ed:58:7b:b3:ee:2e:a7:61:42:78:e2:7c:dc:bf:7a:72:
        d0:3a:4e:86:39:a4:b3:88:e1:93:33:90:2a:92:97:47:1e:15:
        6a:2d:0f:4c:62:b4:da:f2:44:e3:a0:f7:ea:4b:3d:96:cc:ce:
        93:77:a8:68:7f:f6:92:03:3c:42:91:8f:89:0c:17:01:c7:c3:
        78:e2:c3:9a:5d:89:24:14:3d:a0:03:91:b3:1a:66:52:dd:f4:
        b9:39:af:4f:a7:a1:fe:b8:2a:e6:88:47:1a:82:6f:10:e2:5e:
        9f:eb:67:5f:79:ea:80:36:7e:45:bf:87:73:3a:f6:b7:99:66:
        08:a5:33:3c:49:41:e0:51:f9:de:11:bf:9e:fc:1b:16:6b:0a:
        70:4e:35:00:0d:d9:1d:8d:ee:9b:91:ca:9b:db:9a:8b:10:6d:
        73:aa:2f:71:4a:2a:3c:7f:bd:c1:28:eb:c3:e7:1c:ce:94:8c:
        98:a6:f1:b1:a7:f0:0a:2b:cc:d9:b8:48:b1:bd:de:cd:ba:38:
        a8:d8:d5:34:34:c3:39:c2:5b:c7:8b:ef:86:a8:fb:f7:ef:a5:
        eb:8b:4a:61:05:dc:09:ca:d3:72:30:e9:b2:31:2e:ae:2a:49:
        7c:3a:d1:b7:0b:55:e3:79:22:08:85:67:d1:3b:bf:85:55:7f:
        e8:ce:0c:09:88:c1:45:a0:b9:e4:5c:9b:19:08:3c:fd:94:17:
        3d:74:32:54:df:15:ee:5b:8d:49:2e:9f:3a:ab:e4:dc:57:49:
        8e:e3:e4:7a:99:12:23:40:62:a1:8b:17:16:f5:d4:17:0d:7a:
        95:ca:59:b8:ef:d9:44:13
subject=CN=Takara Home Lab Issuing CA 2026
ubuntu@ca:~$
```

CSRを Root CAに送り、署名してもらい、返却してもらう

自分のリクエストに署名したものかをチェックする
```console
openssl verify \
  -CAfile /home/ubuntu/.step/certs/root_ca.crt \
  /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.crt
```

実行例
```console
ubuntu@ca:~/pki-transfer$ openssl verify \
  -CAfile /home/ubuntu/.step/certs/root_ca.crt \
  /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.crt
/home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.crt: OK
```

表示して確認するコマンド
```console
openssl x509 \
  -in ./Takara_Home_Lab_Issuing_CA_2026.crt \
  -noout -subject -issuer -dates -text
```

実行例
```console
ubuntu@ca:~/pki-transfer$ ls -la
total 20
drwx------ 2 ubuntu ubuntu 4096 Sep 28 08:51 .
drwxr-x--- 7 ubuntu ubuntu 4096 Sep 28 08:16 ..
-rw-rw-r-- 1 ubuntu ubuntu 1939 Sep 28 08:51 Takara_Home_Lab_Issuing_CA_2026.crt
-rw------- 1 ubuntu ubuntu 1695 Sep 28 07:37 Takara_Home_Lab_Issuing_CA_2026.csr
-rw-rw-r-- 1 ubuntu ubuntu 2009 Sep 28 08:11 Takara_Home_Lab_Root_CA_2026.crt

ubuntu@ca:~/pki-transfer$ openssl x509 \
  -in ./Takara_Home_Lab_Issuing_CA_2026.crt \
  -noout -subject -issuer -dates -text
subject=CN=Takara Home Lab Issuing CA 2026
issuer=C=JP, ST=Chiba, L=Inzai, O=Takara Home Lab, CN=Takara Home Lab Root CA 2026
notBefore=Sep 28 08:07:00 2026 GMT
notAfter=Sep 28 08:07:00 2031 GMT
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: 4162480002246791696 (0x39c419821e965210)
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: C=JP, ST=Chiba, L=Inzai, O=Takara Home Lab, CN=Takara Home Lab Root CA 2026
        Validity
            Not Before: Sep 28 08:07:00 2026 GMT
            Not After : Sep 28 08:07:00 2031 GMT
        Subject: CN=Takara Home Lab Issuing CA 2026
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (4096 bit)
                Modulus:
                    00:d4:ba:23:5e:32:20:5f:f4:a2:c8:a4:f1:96:f1:
                    2c:6f:81:6f:90:ac:d5:5d:01:00:aa:51:65:9f:ed:
                    17:e8:a4:73:d2:1c:5b:65:3c:55:c4:4a:b9:ae:5a:
                    7a:33:3d:13:af:31:10:63:cd:99:fe:4d:05:08:93:
                    e3:6d:8c:f9:36:96:ff:60:7a:bb:e3:ab:2f:21:7c:
                    3e:16:32:b9:13:c1:cc:7f:52:d0:dc:ea:dd:c8:03:
                    ab:06:ef:3d:6a:63:3b:87:ad:37:d9:30:e7:d8:98:
                    e8:f1:e8:60:25:75:84:22:4b:79:94:c6:ce:a7:77:
                    4d:db:07:28:8f:0f:14:c8:30:d1:5c:0d:8c:12:1e:
                    a6:72:1d:e7:da:2e:a8:4f:de:b2:20:12:d9:4c:4e:
                    bf:2d:92:5e:f1:cc:a6:c2:da:0c:38:36:5a:ba:ca:
                    d8:89:7f:c0:5e:8c:a4:56:a5:b4:76:82:fc:9e:26:
                    9a:e6:3c:90:40:0d:3c:d1:b2:6c:31:48:bc:92:ee:
                    d4:34:cb:93:db:2b:a5:78:84:da:8c:ce:09:60:f0:
                    b5:e3:ee:16:d1:c7:a9:5f:da:d4:74:22:02:1e:c2:
                    03:fc:56:19:4c:66:f7:ec:d3:5f:a9:6d:37:d1:7c:
                    f5:86:c4:ae:2c:68:83:a1:74:9e:2d:09:e4:62:a3:
                    ca:da:0f:94:c8:aa:ae:06:50:d7:17:85:1d:05:c8:
                    4a:dd:6c:07:02:c6:1b:83:ee:31:5f:cf:91:65:15:
                    49:c1:6f:2d:f4:f6:88:9c:ba:53:f5:43:7d:33:ad:
                    d9:eb:f7:62:c1:9c:02:86:0b:a8:a4:5c:c9:e9:c3:
                    62:e5:df:f6:4f:ec:6e:fa:9f:47:e9:c1:41:0a:f6:
                    f2:b9:9a:66:fb:8f:50:81:da:73:a1:a3:c1:9e:82:
                    94:68:91:40:1d:1a:ed:23:6a:d9:43:c3:55:b2:46:
                    b1:e9:ac:7f:97:94:fb:68:45:db:52:a2:c2:24:0c:
                    d1:e0:7e:c8:ed:6e:7c:01:61:ea:a7:b0:0e:dc:f1:
                    f2:cf:0f:d0:df:cd:16:c8:55:bd:99:d0:aa:90:d5:
                    f9:e9:1a:88:88:98:88:23:45:9f:f7:66:ba:39:2f:
                    18:e8:86:e0:bb:4b:3b:04:8b:9e:5a:19:02:d9:91:
                    bd:1a:15:e7:ba:a3:94:ce:b4:99:00:9b:5b:fa:a3:
                    69:a5:1c:07:1e:f5:f7:27:02:6a:79:c0:38:59:5b:
                    36:5d:55:90:bc:0b:02:e8:85:5e:3e:22:d2:a4:48:
                    98:a3:30:81:9d:f6:a6:77:9c:f5:d9:f3:31:68:6e:
                    0d:cd:f7:f7:5e:02:02:9f:4c:30:60:cc:c6:0c:00:
                    d3:aa:61
                Exponent: 65537 (0x10001)
        X509v3 extensions:
            X509v3 Basic Constraints: critical
                CA:TRUE, pathlen:0
            X509v3 Key Usage: critical
                Certificate Sign, CRL Sign
            X509v3 Subject Alternative Name:
                DNS:Takara Home Lab Issuing CA 2026
    Signature Algorithm: sha256WithRSAEncryption
    Signature Value:
        3b:e3:1e:5a:e9:05:d5:0c:3d:52:1b:1e:f7:c6:d2:a7:02:b6:
        7c:a9:c0:0c:4d:5e:2e:0b:88:3f:ee:99:ee:9c:cb:12:27:3e:
        87:c7:9d:8b:d0:7c:4d:84:2e:57:3c:c6:aa:b7:d7:3e:60:bd:
        2a:2c:69:91:ba:75:5a:6b:3a:58:ee:aa:80:91:d4:6b:85:1f:
        13:9d:ad:56:87:91:45:4d:d2:5e:8b:22:07:56:4f:9b:4c:30:
        13:cc:09:d8:04:1d:cd:b3:63:68:88:32:40:db:52:07:7e:e0:
        17:bc:d0:b1:02:18:d8:82:26:6f:68:55:7e:a6:29:a5:aa:25:
        de:a5:0b:90:e1:4a:f0:8e:d8:e9:56:43:29:61:64:aa:c5:36:
        f9:49:31:6c:f5:d9:75:74:39:3b:1e:7a:69:22:42:0d:17:ce:
        ef:27:95:e4:a5:e2:02:d9:21:9d:75:5b:83:44:ec:1d:29:ee:
        90:8c:34:30:2d:b0:27:2f:5f:51:90:03:02:83:76:c7:0e:da:
        71:d2:0d:fc:28:19:1b:2a:1e:da:08:d0:2f:1f:fe:e2:fc:cd:
        7c:66:b6:f0:67:b4:53:6a:39:9f:52:55:9a:03:a4:d2:c6:3a:
        98:51:c2:5d:40:fd:a4:1f:1e:1e:76:90:ce:1a:82:dd:75:85:
        85:67:2b:ee:bf:89:ed:fa:7a:5d:ae:a9:75:b5:a5:c6:35:1c:
        60:ed:03:e5:cd:af:e2:49:df:c4:40:92:ca:55:7c:2d:3b:07:
        47:2f:16:62:ba:8b:fc:fd:6c:65:96:48:29:ed:73:ea:49:3f:
        0a:cc:aa:f0:fa:fc:b8:5e:b5:87:62:98:7b:79:36:1a:ef:cb:
        4b:dc:63:a3:66:03:d5:69:6e:fd:e1:ac:64:91:1b:be:43:d4:
        dd:44:e1:9b:7d:54:6a:92:22:93:a5:fe:6d:92:40:5c:26:46:
        7f:74:75:a2:4b:c1:9e:9b:40:74:21:b8:85:70:fd:e3:59:fb:
        6b:59:65:36:dc:0e:fe:49:64:41:a1:56:e2:88:26:fb:14:a5:
        21:bf:6e:ed:78:d0:94:4b:41:dc:4c:79:0c:e1:d3:05:58:68:
        f4:83:d6:c2:0a:db:ec:a7:a1:1f:02:97:60:01:63:d2:8c:73:
        80:73:6e:a4:6f:74:fa:8a:04:a9:52:c9:b4:d1:b4:a7:85:69:
        11:00:f9:ca:10:61:a8:f4:b0:06:2f:6b:7e:28:c8:d4:6f:ff:
        9e:e2:23:8d:93:70:90:a9:19:71:bb:c7:5e:60:c3:79:df:d6:
        a4:cd:57:3c:e6:11:a0:57:92:71:c5:14:37:79:f5:68:00:da:
        9c:8e:59:92:2b:34:f1:3a
```

##  秘密鍵と証明書の対応確認

中間CAの秘密鍵を取り出して、ハッシュを作成するコマンド
```console
openssl pkey \
  -in /home/ubuntu/.step/secrets/intermediate_ca_key \
  -pubout -outform DER |
sha256sum
```

実行例
```console
ubuntu@ca:~/pki-transfer$ openssl pkey \
  -in /home/ubuntu/.step/secrets/intermediate_ca_key \
  -pubout -outform DER |
sha256sum
Enter pass phrase for /home/ubuntu/.step/secrets/intermediate_ca_key:
c258efed50a55ea720bc3f6bbed4b99969d0e7e7490db71eab0fa3c395638f83  -
```

署名された証明書が自分が要求したものであるかを確認するコマンド
```console
~/pki-transfer$ openssl x509 -in ./Takara_Home_Lab_Issuing_CA_2026.crt \
   -pubkey -noout| \
   openssl pkey -pubin -outform DER | \
   sha256sum
```

実行例、中間証明書の秘密鍵のハッシュと一致しているので、正当なものであることが確認できた。
```console
ubuntu@ca:~/pki-transfer$ openssl x509 -in ./Takara_Home_Lab_Issuing_CA_2026.crt \
   -pubkey -noout| \
   openssl pkey -pubin -outform DER | \
   sha256sum
c258efed50a55ea720bc3f6bbed4b99969d0e7e7490db71eab0fa3c395638f83  -
```

## 正式配備
```console
install -m 0644 \
  /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.crt \
  /home/ubuntu/.step/certs/intermediate_ca.crt
```

実行例
```console
ubuntu@ca:~$ install -m 0644 \
  /home/ubuntu/pki-transfer/Takara_Home_Lab_Issuing_CA_2026.crt \
  /home/ubuntu/.step/certs/intermediate_ca.crt
```


## step-caのサービス化とACMEエンドポイントの公開

```console
ubuntu@ca:~$ ls
pki-transfer

ubuntu@ca:~$ find . -name ca.json -print
./.step/config/ca.json

ubuntu@ca:~$ cd .step/
ubuntu@ca:~/.step$ cd config/
ubuntu@ca:~/.step/config$ ls
ca.json  defaults.json

ubuntu@ca:~/.step/config$ cat ca.json |jq
{
  "root": "/home/ubuntu/.step/certs/root_ca.crt",
  "federatedRoots": null,
  "crt": "/home/ubuntu/.step/certs/intermediate_ca.crt",
  "key": "/home/ubuntu/.step/secrets/intermediate_ca_key",
  "address": ":443",
  "insecureAddress": "",
  "dnsNames": [
    "ca.labo.local"
  ],
  "logger": {
    "format": "text"
  },
  "db": {
    "type": "badgerv2",
    "dataSource": "/home/ubuntu/.step/db",
    "badgerFileLoadingMode": ""
  },
  "authority": {
    "provisioners": [
      {
        "type": "JWK",
        "name": "admin-jwk",
        "key": {
          "use": "sig",
          "kty": "EC",
          "kid": "NohBhDhYoNY7h6MqmIkDd3_iezmjc67Y7lCSsVV-kTU",
          "crv": "P-256",
          "alg": "ES256",
          "x": "JG-OwWXzZpl7ZgHdwv0RRoR_jJ13F9icXrCZEVJtlzg",
          "y": "pekRoafhCK1gVhh52WIO_0Mkwq8bq0a5bnCo2Q1syTQ"
        },
        "encryptedKey": "eyJhbGciOiJQQkVTMi1IUzI1NitBMTI4S1ciLCJjdHkiOiJqd2sranNvbiIsImVuYyI6IkEyNTZHQ00iLCJwMmMiOjYwMDAwMCwicDJzIjoieWZQZmJNZFJzVVRGOGFIOGNrZ1NsQSJ9.3ZQX_xMI51maQPDwQHq7meZElB2vADg1kxOeoQvPu3fssQdbMAyA7g.7KqQLtnvfZB5fCy8.Fu3xOEsbMZFatpiMznAC9FygpoQqqBSpoJbSnF-I68VUgo73Sf-2RQADYCD_R9nHmdhKI1Sm5ebN_Js1NGqRDLsmlec5EzjWapCdixjpBi68ATDyBCWhaN91VDxyIw0UKdea9FRLrSFAoSIAQ_XPxeHZjinj7yWAFY0H_KKvJuoU5HHvKMm8Xr6asMF-Dyr-LKdA-uljJbwL2x4tIZqUqFBcYv5yLopVeVpiJDjPaasZE3en1_wd2F83AZsmbD4Dset1iI0Eouva9G2a4q4tb12iIgghzf6ww5SilhbWc11hAKsAYnuA5aWAQtk8llIz5d5QJIIR2PuIFzRUjl8.aKr5TcL2hHWKyxCwF7gYag"
      },
      {
        "type": "ACME",
        "name": "acme"
      }
    ]
  },
  "tls": {
    "cipherSuites": [
      "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
      "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
    ],
    "minVersion": 1.2,
    "maxVersion": 1.3,
    "renegotiation": false
  }
}
```

## 権限の付与
 

```console
sudo setcap CAP_NET_BIND_SERVICE=+eip "$(command -v step-ca)"
getcap "$(command -v step-ca)"
```

実行例
```console
ubuntu@ca:~/.step/config$ sudo setcap CAP_NET_BIND_SERVICE=+eip "$(command -v step-ca)"
ubuntu@ca:~/.step/config$ getcap "$(command -v step-ca)"
/usr/bin/step-ca cap_net_bind_service=eip
```

## 起動コマンド

```console
step-ca /home/ubuntu/.step/config/ca.json
```

実行例、問題発生！
```console
ubuntu@ca:~$ step-ca /home/ubuntu/.step/config/ca.json
badger 2026/09/29 00:24:21 INFO: All 0 tables opened in 0s
error reading "/home/ubuntu/.step/certs/intermediate_ca.crt": no such file or directory

ubuntu@ca:~$ find . -name intermediate_ca.crt -print
```

問題の取り除いて、再トライ

```console
ubuntu@ca:~$ step-ca /home/ubuntu/.step/config/ca.json
badger 2026/09/29 00:32:47 INFO: All 0 tables opened in 0s
badger 2026/09/29 00:32:47 INFO: Replaying file id: 0 at offset: 0
badger 2026/09/29 00:32:47 INFO: Replay took: 22.612µs
Please enter the password to decrypt /home/ubuntu/.step/secrets/intermediate_ca_key:
2026/09/29 00:32:54 Building new tls configuration using step-ca x509 Signer Interface
2026/09/29 00:32:54 Starting Smallstep CA/0.30.2 (linux/amd64)
2026/09/29 00:32:54 Documentation: https://u.step.sm/docs/ca
2026/09/29 00:32:54 Community Discord: https://u.step.sm/discord
2026/09/29 00:32:54 Config file: /home/ubuntu/.step/config/ca.json
2026/09/29 00:32:54 The primary server URL is https://ca.labo.local:443
2026/09/29 00:32:54 Root certificates are available at https://ca.labo.local:443/roots.pem
2026/09/29 00:32:54 X.509 Root Fingerprint: de1503b5cb911e35e6dd973b9aa2619307283719153cdec1f805f57cd6ea23be
2026/09/29 00:32:54 Serving HTTPS on :443 ...
```


```console
step ca health \
  --ca-url https://ca.labo.local \
  --root /home/ubuntu/.step/certs/root_ca.crt
```

実行例
```console
ubuntu@ca:~$ step ca health \
  --ca-url https://ca.labo.local \
  --root /home/ubuntu/.step/certs/root_ca.crt
ok
```

## 権限解除

```console
sudo setcap -r "$(command -v step-ca)"
getcap "$(command -v step-ca)"
```

実行例、戻りメッセージは無し
```console
ubuntu@ca:~$ sudo setcap -r "$(command -v step-ca)"
getcap "$(command -v step-ca)"
```

## サービスユーザーの作成

```console
sudo useradd \
  --user-group \
  --system \
  --create-home \
  --home /etc/step-ca \
  --shell /bin/false \
  step
```

```console
sudo mv /home/ubuntu/.step/certs /etc/step-ca/
sudo mv /home/ubuntu/.step/config /etc/step-ca/
sudo mv /home/ubuntu/.step/db /etc/step-ca/
sudo mv /home/ubuntu/.step/secrets /etc/step-ca/
```

パスの更新
```console
jq '
  .root = "/etc/step-ca/certs/root_ca.crt" |
  .crt = "/etc/step-ca/certs/intermediate_ca.crt" |
  .key = "/etc/step-ca/secrets/intermediate_ca_key" |
  .address = ":443" |
  .dnsNames = ["ca.labo.local"] |
  .db.dataSource = "/etc/step-ca/db"
' /etc/step-ca/config/ca.json \
  > /tmp/step-ca-ca.json
```

```console
root@ca:/home/ubuntu# sudo install -o step -g step -m 0640 \
  /tmp/step-ca-ca.json \
  /etc/step-ca/config/ca.json

root@ca:/home/ubuntu# rm /tmp/step-ca-ca.json
```

```console
sudo chown -R step:step /etc/step-ca
sudo chmod 700 /etc/step-ca
sudo chmod 700 /etc/step-ca/secrets
sudo chmod 600 /etc/step-ca/secrets/intermediate_ca_key
```


```console
root@ca:/home/ubuntu# sudo install -o step -g step -m 0600 \
  /dev/null /etc/step-ca/password.txt
root@ca:/home/ubuntu# ls -la /etc/step-ca/
total 36
drwx------   6 step step 4096 Sep 29 01:25 .
drwxr-xr-x 107 root root 4096 Sep 29 01:14 ..
-rw-r--r--   1 step step  220 Feb 13  2026 .bash_logout
-rw-r--r--   1 step step 3771 Feb 13  2026 .bashrc
-rw-r--r--   1 step step  807 Feb 13  2026 .profile
drwx------   2 step step 4096 Sep 29 00:31 certs
drwx------   2 step step 4096 Sep 29 01:20 config
drwx------   2 step step 4096 Sep 29 00:44 db
-rw-------   1 step step    0 Sep 29 01:25 password.txt
drwx------   2 step step 4096 Sep 28 07:37 secrets
```

```console
root@ca:/etc/step-ca# vi password.txt
root@ca:/etc/step-ca# ls -la
total 40
drwx------   6 step step 4096 Sep 29 01:32 .
drwxr-xr-x 107 root root 4096 Sep 29 01:14 ..
-rw-r--r--   1 step step  220 Feb 13  2026 .bash_logout
-rw-r--r--   1 step step 3771 Feb 13  2026 .bashrc
-rw-r--r--   1 step step  807 Feb 13  2026 .profile
drwx------   2 step step 4096 Sep 29 00:31 certs
drwx------   2 step step 4096 Sep 29 01:20 config
drwx------   2 step step 4096 Sep 29 00:44 db
-rw-------   1 step step    8 Sep 29 01:32 password.txt
drwx------   2 step step 4096 Sep 28 07:37 secrets
```

```console
root@ca:/etc/systemd/system# systemctl status step-ca
● step-ca.service - step-ca service
     Loaded: loaded (/etc/systemd/system/step-ca.service; enabled; preset: enabled)
     Active: active (running) since Tue 2026-09-29 01:41:04 UTC; 14min ago
 Invocation: c623f60cbb354640ac552facc2a01fb5
       Docs: https://smallstep.com/docs/step-ca
   Main PID: 2282 (step-ca)
      Tasks: 6 (limit: 1486)
     Memory: 12.7M (peak: 13M)
        CPU: 2.167s
     CGroup: /system.slice/step-ca.service
             └─2282 /usr/bin/step-ca config/ca.json --password-file password.txt

Sep 29 01:41:05 ca.labo.local step-ca[2282]: badger 2026/09/29 01:41:05 INFO: Replay took: 4.759µs
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 Building new tls configuration using step-ca x509 Signer Interface
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 Starting Smallstep CA/0.30.2 (linux/amd64)
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 Documentation: https://u.step.sm/docs/ca
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 Community Discord: https://u.step.sm/discord
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 Config file: config/ca.json
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 The primary server URL is https://ca.labo.local:443
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 Root certificates are available at https://ca.labo.local:443/roots.pem
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 X.509 Root Fingerprint: de1503b5cb911e35e6dd973b9aa2619307283719153cdec1f805f57c>
Sep 29 01:41:05 ca.labo.local step-ca[2282]: 2026/09/29 01:41:05 Serving HTTPS on :443 ...
```

# Root CA をCAサーバ自身へ信頼登録

```console
sudo install -m 0644 \
  /etc/step-ca/certs/root_ca.crt \
  /usr/local/share/ca-certificates/takara-home-lab-root-ca-2026.crt

sudo update-ca-certificates
```

```console
ubuntu@ca:~$ sudo install -m 0644 \
  /etc/step-ca/certs/root_ca.crt \
  /usr/local/share/ca-certificates/takara-home-lab-root-ca-2026.crt

ubuntu@ca:~$ sudo update-ca-certificates
Updating certificates in /etc/ssl/certs...
rehash: warning: skipping ca-certificates.crt, it does not contain exactly one certificate or CRL
1 added, 0 removed; done.
Running hooks in /etc/ca-certificates/update.d...
done.
```

```
ubuntu@ca:~$ curl https://ca.labo.local/health
{"status":"ok"}
```


## 最終確認

```console
sudo openssl s_client \
  -connect ca.labo.local:443 \
  -servername ca.labo.local \
  -CAfile /etc/step-ca/certs/root_ca.crt \
  -verify_return_error </dev/null
```

実行例
```console
ubuntu@ca:~$ sudo openssl s_client \
  -connect ca.labo.local:443 \
  -servername ca.labo.local \
  -CAfile /etc/step-ca/certs/root_ca.crt \
  -verify_return_error </dev/null
Connecting to 192.168.1.250
CONNECTED(00000003)
depth=2 C=JP, ST=Chiba, L=Inzai, O=Takara Home Lab, CN=Takara Home Lab Root CA 2026
verify return:1
depth=1 CN=Takara Home Lab Issuing CA 2026
verify return:1
depth=0 CN=Step Online CA
verify return:1
---
Certificate chain
 0 s:CN=Step Online CA
   i:CN=Takara Home Lab Issuing CA 2026
   a:PKEY: EC, (prime256v1); sigalg: sha256WithRSAEncryption
   v:NotBefore: Sep 29 01:40:05 2026 GMT; NotAfter: Sep 30 01:41:05 2026 GMT
 1 s:CN=Takara Home Lab Issuing CA 2026
   i:C=JP, ST=Chiba, L=Inzai, O=Takara Home Lab, CN=Takara Home Lab Root CA 2026
   a:PKEY: RSA, 4096 (bit); sigalg: sha256WithRSAEncryption
   v:NotBefore: Sep 28 08:07:00 2026 GMT; NotAfter: Sep 28 08:07:00 2031 GMT
---
Server certificate
-----BEGIN CERTIFICATE-----
MIIDbTCCAVWgAwIBAgIRAJQX8CqTZVfPZ+zstTzjGYMwDQYJKoZIhvcNAQELBQAw
KjEoMCYGA1UEAxMfVGFrYXJhIEhvbWUgTGFiIElzc3VpbmcgQ0EgMjAyNjAeFw0y
NjA5MjkwMTQwMDVaFw0yNjA5MzAwMTQxMDVaMBkxFzAVBgNVBAMTDlN0ZXAgT25s
aW5lIENBMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE66MfgnXQXVRB3qZXmdVu
IRCCRITGCJjjFBSoBrpd/KkE3ExGoAgCaPgLvCqHiryVcU8eoLrTx58duuy+UHZm
mKNqMGgwDgYDVR0PAQH/BAQDAgeAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEF
BQcDAjAdBgNVHQ4EFgQU7HFHq86k0BwbnLqy/EgsAnioMuAwGAYDVR0RBBEwD4IN
Y2EubGFiby5sb2NhbDANBgkqhkiG9w0BAQsFAAOCAgEAhfw91ntejiJA4zUtqXtM
+QfvuweUp9eLxbYLWEOUCQyjbrIVkcwTuiz8k4DmdPrp2JWryjUd3UKfrLHysiot
K3gUWeGAXElQnNv0D5gRT6s1KVHWb6+DcNq6ynuVJxTI8Q1m1mJfPypuXlKxmdk/
jJja9/GkHNzLCEwvUO3gtIUTNT2pCHcA/UJJPD/ywNh6pBdrtg/evcHxV2vo+Ar1
qFIBG26btKzjPPVW3HogXRS/raABZ9kF3ym/wFYzk4J2nymi0QlsD2a4Fizq99ZS
oSxP4yzQDwviqD0/tD2ypyIKCZhYziHix2jE5GzGtA1G7mUEjmqioj80szawckPU
5ruGe4bmx7LDR+gTJmYuoleSMNgDKh1/i0qZXx5DL7ZXeHdV4+QaB8MWMd68jC0e
tCtiJqZmQ6q4L9lgOVbw3HqPCK1ifsLQmI1mmNVRzlMGyCXNXRmykvFyerEKjsP/
u0r/fTIm+KaKlxwdUTuGWaBo+1zMqwpsbhJSxsnqnpymkxKoITbgxKS9ReC5XSNV
SsP0PKIlbtHvj6htJcFS7rm3ozHvW0v/GB9l09CFkTLX8cikqa+OwZ5Jx+jcOzzl
50qI7p998Up+B29fqZ4TA3RiAdDvFqo1easbO3+j1g7bmt47TevkOHsxGIPaxBTa
huDTeDtMaa4DjsHNv795+Wc=
-----END CERTIFICATE-----
subject=CN=Step Online CA
issuer=CN=Takara Home Lab Issuing CA 2026
---
Acceptable client certificate CA names
C=JP, ST=Chiba, L=Inzai, O=Takara Home Lab, CN=Takara Home Lab Root CA 2026
CN=Takara Home Lab Issuing CA 2026
Requested Signature Algorithms: RSA-PSS+SHA256:ECDSA+SHA256:ed25519:RSA-PSS+SHA384:RSA-PSS+SHA512:ECDSA+SHA384:ECDSA+SHA512
Shared Requested Signature Algorithms: RSA-PSS+SHA256:ECDSA+SHA256:ed25519:RSA-PSS+SHA384:RSA-PSS+SHA512:ECDSA+SHA384:ECDSA+SHA512
Peer signing digest: SHA256
Peer signature type: ecdsa_secp256r1_sha256
Negotiated TLS1.3 group: X25519MLKEM768
---
SSL handshake has read 3977 bytes and written 1649 bytes
Verification: OK
---
New, TLSv1.3, Cipher is TLS_AES_128_GCM_SHA256
Protocol: TLSv1.3
Server public key is 256 bit
This TLS version forbids renegotiation.
Compression: NONE
Expansion: NONE
No ALPN negotiated
Early data was not sent
Verify return code: 0 (ok)
---
---
Post-Handshake New Session Ticket arrived:
SSL-Session:
    Protocol  : TLSv1.3
    Cipher    : TLS_AES_128_GCM_SHA256
    Session-ID: 76180417B799C1E70521BBFB9A0F245CA5DA1E52485B8028E1D528624D6B653E
    Session-ID-ctx:
    Resumption PSK: AA39F557487251DBC9DCEFBA60BA98CBCADEC807D234810B3B2A0CF414A1D6E4
    PSK identity: None
    PSK identity hint: None
    SRP username: None
    TLS session ticket lifetime hint: 604800 (seconds)
    TLS session ticket:
    0000 - 0a 9c a3 d4 47 92 d2 8c-c8 07 6b 0c ac f6 d1 1a   ....G.....k.....
    0010 - ca c7 c8 0e 73 50 be 00-4e ca 0a 5d fe a3 a5 7d   ....sP..N..]...}
    0020 - d7 84 f8 ee e6 3f 04 3e-3e b4 71 9f e0 98 b7 d9   .....?.>>.q.....
    0030 - 8c 8f dc 93 55 88 50 4e-09 79 31 02 6c b0 15 2b   ....U.PN.y1.l..+
    0040 - 30 64 3c b2 72 de a2 c4-38 db bb 25 86 d1 d6 0b   0d<.r...8..%....
    0050 - 6d 06 2e e5 db 79 09 f3-11 b6 97 76 61 17 0a 98   m....y.....va...
    0060 - 34 04 d2 58 30 b0 77 ff-64                        4..X0.w.d

    Start Time: 1790647179
    Timeout   : 7200 (sec)
    Verify return code: 0 (ok)
    Extended master secret: no
    Max Early Data: 0
---
read R BLOCK
DONE
ubuntu@ca:~$
```



```console
curl https://ca.labo.local/acme/acme/directory
```

```console
ubuntu@ca:~$ curl -s https://ca.labo.local/acme/acme/directory |jq
{
  "newNonce": "https://ca.labo.local/acme/acme/new-nonce",
  "newAccount": "https://ca.labo.local/acme/acme/new-account",
  "newOrder": "https://ca.labo.local/acme/acme/new-order",
  "revokeCert": "https://ca.labo.local/acme/acme/revoke-cert",
  "keyChange": "https://ca.labo.local/acme/acme/key-change"
}
```