# dns-01 対応の Playbook

dns-01 にすると、**80番ポートの開放とチャレンジファイルの配置が不要**になり、複数ドメイン（SAN）やワイルドカードにも対応できます。DNS サーバーが何かで更新方法が変わるため、ここでは社内 DNS が BIND などの **RFC 2136（動的更新 + TSIG 鍵）** に対応しているものとして書いています。他の DNS を使っている場合は、4. の DNS 更新タスクだけを差し替えます（末尾の表を参照）。

## 全体の流れ

| 手順 | 実行場所 | 内容 |
|---|---|---|
| 1 | 対象ホスト | 鍵・CSR を作成 |
| 2 | 対象ホスト | `acme_certificate` で第1段階（チャレンジ取得） |
| 3 | コントロールノード（`delegate_to: localhost`） | `_acme-challenge` の TXT レコードを追加 |
| 4 | 対象ホスト | 第2段階で検証・証明書取得 |
| 5 | コントロールノード | TXT レコードを削除（失敗時も実行） |
| 6 | 対象ホスト | Nginx へ配置・reload |

TSIG の秘密鍵は各 Nginx ホストに配らず、コントロールノードだけで使います。

## 事前準備

**BIND 側（例）: 更新用 TSIG 鍵と権限を作る**

```bash
tsig-keygen -a hmac-sha256 acme-update   # 出力された key ブロックを named.conf へ、secret を控える
```

```
zone "labo.local" {
    type primary;
    file "labo.local.zone";
    update-policy {
        grant acme-update name _acme-challenge.app.labo.local. TXT;   # 対象名だけ許可
    };
};
```

**CoreDNS 側（例）: etcd プラグインで TXT を更新できるようにする**

CoreDNS は RFC 2136（動的更新）にも TSIG 鍵による更新にも対応していません（`tsig` プラグインはクエリや転送の認証用です）。そのため、BIND 用の `nsupdate` は使えません。代わりに、レコードを **etcd に保存して CoreDNS が参照する構成（`etcd` プラグイン）** にし、`etcdctl` で `_acme-challenge` の TXT を追加・削除します。

```
labo.local:53 {
    etcd {
        path /skydns
        endpoint https://192.168.1.10:2379
        tls /etc/coredns/etcd-client.crt /etc/coredns/etcd-client.key /etc/coredns/etcd-ca.crt
    }
    log
    errors
}
```

`_acme-challenge.app.labo.local` の TXT は、ドメイン名を逆順にしたキーに JSON で保存します。同じ名前に複数の値を置く場合（SAN やワイルドカード）は、末尾の識別子（`x1` など）を変えて別キーにします。

```bash
export ETCDCTL_ENDPOINTS=https://192.168.1.10:2379

# 追加
etcdctl put /skydns/local/labo/app/_acme-challenge/x1 '{"text":"<TXT値>","ttl":60}'

# 確認
dig +short TXT _acme-challenge.app.labo.local @192.168.1.1

# 削除
etcdctl del --prefix /skydns/local/labo/app/_acme-challenge/
```

etcd の権限は、このキー配下だけに限定したロールを作って、コントロールノード用のユーザーに付与してください（BIND の `update-policy` に相当します）。

```bash
etcdctl role add acme-update
etcdctl role grant-permission acme-update --prefix=true readwrite /skydns/local/labo/
etcdctl user add acme-update
etcdctl user grant-role acme-update acme-update
etcdctl auth enable    # 事前に root ユーザーを作成しておくこと
```

Playbook では、3. の TXT 追加と 5. の TXT 削除の `community.general.nsupdate` を、次のタスクに置き換えます（`etcdctl` はコントロールノードに必要です）。`nsupdate` 用の `dnspython` や `tsig_*` 変数は不要になります。

```yaml
- name: Add _acme-challenge TXT records (CoreDNS + etcd)
  ansible.builtin.command:
    argv:
      - etcdctl
      - put
      - "/skydns/{{ item.0.key.split('.') | reverse | join('/') }}/{{ (item.1 | hash('sha1'))[:12] }}"
      - "{{ {'text': item.1, 'ttl': 60} | to_json }}"
  environment:
    ETCDCTL_ENDPOINTS: "{{ etcd_endpoints }}"
  loop: "{{ acme_challenge.challenge_data_dns | dict2items | subelements('value') }}"
  delegate_to: localhost
  become: false
  changed_when: true

# always セクション側
- name: Remove _acme-challenge TXT records (CoreDNS + etcd)
  ansible.builtin.command:
    argv:
      - etcdctl
      - del
      - "/skydns/{{ item.0.key.split('.') | reverse | join('/') }}/{{ (item.1 | hash('sha1'))[:12] }}"
  environment:
    ETCDCTL_ENDPOINTS: "{{ etcd_endpoints }}"
  loop: "{{ acme_challenge.challenge_data_dns | dict2items | subelements('value') }}"
  delegate_to: localhost
  become: false
  changed_when: true
```

`etcd_endpoints` は vars に追加してください（例: `https://192.168.1.10:2379`）。認証を有効にした場合は、`ETCDCTL_USER` などの認証情報も `environment` に加えます。

> `etcdctl put` / `del` と、CoreDNS（v1.12.1）が `dig` の TXT 問い合わせに応答することは動作確認済みです。上記の Ansible タスクそのものは未実行です。

**コントロールノード側**

```bash
ansible-galaxy collection install community.crypto community.general
sudo apt install python3-dnspython          # 基本はこちらを利用
#pip install dnspython                      # nsupdate モジュールに必要
ansible-vault encrypt_string '<TSIG secret>' --name tsig_key_secret   # 出力を vars に貼る
```

**Vault 側（必要な場合のみ）**: Vault が社内 DNS を引けないときは、ACME 検証用のリゾルバーを指定します。

```bash
vault write pki_int/config/acme enabled=true dns_resolver=192.168.1.1:53 \
  default_directory_policy="role:labo-local"
```

既存の設定値がある場合は、同じパラメータも併記してください。

## Playbook

```yaml
---
- name: Renew certificate via Vault ACME (dns-01)
  hosts: nginx_servers
  become: true
  vars:
    domains:                          # 複数指定すると SAN になる（先頭が CN）
      - app.labo.local
    primary_domain: "{{ domains[0] }}"
    acme_directory: https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory
    acme_dir: /etc/ssl/acme
    cert_dir: /etc/nginx/ssl
    renew_before_days: 10

    dns_server: 192.168.1.1           # 動的更新を受け付ける権威 DNS
    dns_zone: labo.local
    tsig_key_name: acme-update
    tsig_key_algorithm: hmac-sha256
    # tsig_key_secret: !vault |  ← ansible-vault encrypt_string の出力を貼る
    dns_propagation_wait: 10          # TXT 追加後の待ち秒数

  tasks:
    # --- 1. 前提（ルートCA、鍵、CSR） ---
    - name: Set CA trust store settings (Debian/Ubuntu)
      ansible.builtin.set_fact:
        ca_trust_dir: /usr/local/share/ca-certificates
        ca_trust_update_cmd: update-ca-certificates
      when: ansible_facts['os_family'] == 'Debian'

    - name: Set CA trust store settings (RedHat/Rocky)
      ansible.builtin.set_fact:
        ca_trust_dir: /etc/pki/ca-trust/source/anchors
        ca_trust_update_cmd: update-ca-trust extract
      when: ansible_facts['os_family'] == 'RedHat'

    - name: Fail on unsupported OS family
      ansible.builtin.fail:
        msg: "Unsupported OS family: {{ ansible_facts['os_family'] }}"
      when: ca_trust_dir is not defined

    - name: Install root CA
      ansible.builtin.copy:
        src: root_ca.crt
        dest: "{{ ca_trust_dir }}/labo-root-ca.crt"
        mode: "0644"
      register: root_ca

    - name: Update CA store
      ansible.builtin.command: "{{ ca_trust_update_cmd }}"
      when: root_ca is changed

    - name: Create certificate directory for nginx
      ansible.builtin.file:
        path: "{{ cert_dir }}"
        state: directory
        mode: "0755"

    - name: Create ACME directory
      ansible.builtin.file:
        path: "{{ acme_dir }}"
        state: directory
        mode: "0700"

    - name: ACME account key
      community.crypto.openssl_privatekey:
        path: "{{ acme_dir }}/account.key"
        type: ECC
        curve: secp256r1

    - name: Server private key
      community.crypto.openssl_privatekey:
        path: "{{ acme_dir }}/{{ primary_domain }}.key"
        type: ECC
        curve: secp256r1

    - name: CSR
      community.crypto.openssl_csr:
        path: "{{ acme_dir }}/{{ primary_domain }}.csr"
        privatekey_path: "{{ acme_dir }}/{{ primary_domain }}.key"
        common_name: "{{ primary_domain }}"
        subject_alt_name: "{{ domains | map('regex_replace', '^', 'DNS:') | list }}"

    # --- 2. 第1段階: チャレンジ取得 ---
    - name: Request certificate (step 1)
      community.crypto.acme_certificate:
        acme_version: 2
        acme_directory: "{{ acme_directory }}"
        validate_certs: true
        account_key_src: "{{ acme_dir }}/account.key"
        terms_agreed: true
        challenge: dns-01
        csr: "{{ acme_dir }}/{{ primary_domain }}.csr"
        dest: "{{ acme_dir }}/{{ primary_domain }}.crt"
        fullchain_dest: "{{ acme_dir }}/{{ primary_domain }}-fullchain.crt"
        remaining_days: "{{ renew_before_days }}"
      register: acme_challenge

    # --- 3〜5. TXT 追加 → 検証 → TXT 削除（更新が必要なときだけ） ---
    - name: Validate with dns-01
      when: acme_challenge is changed and acme_challenge.challenge_data_dns is defined
      block:
        - name: Add _acme-challenge TXT records
          community.general.nsupdate:
            server: "{{ dns_server }}"
            zone: "{{ dns_zone }}"
            record: "{{ item.key[:-(dns_zone | length + 1)] }}"
            type: TXT
            ttl: 60
            value: "{{ item.value }}"
            key_name: "{{ tsig_key_name }}"
            key_secret: "{{ tsig_key_secret }}"
            key_algorithm: "{{ tsig_key_algorithm }}"
            state: present
          loop: "{{ acme_challenge.challenge_data_dns | dict2items }}"
          delegate_to: localhost
          become: false

        - name: Wait for DNS propagation
          ansible.builtin.pause:
            seconds: "{{ dns_propagation_wait }}"

        - name: Finish validation and fetch certificate (step 2)
          community.crypto.acme_certificate:
            acme_version: 2
            acme_directory: "{{ acme_directory }}"
            validate_certs: true
            account_key_src: "{{ acme_dir }}/account.key"
            terms_agreed: true
            challenge: dns-01
            csr: "{{ acme_dir }}/{{ primary_domain }}.csr"
            dest: "{{ acme_dir }}/{{ primary_domain }}.crt"
            fullchain_dest: "{{ acme_dir }}/{{ primary_domain }}-fullchain.crt"
            data: "{{ acme_challenge }}"
            remaining_days: "{{ renew_before_days }}"

      always:
        - name: Remove _acme-challenge TXT records
          community.general.nsupdate:
            server: "{{ dns_server }}"
            zone: "{{ dns_zone }}"
            record: "{{ item.key[:-(dns_zone | length + 1)] }}"
            type: TXT
            value: "{{ item.value }}"
            key_name: "{{ tsig_key_name }}"
            key_secret: "{{ tsig_key_secret }}"
            key_algorithm: "{{ tsig_key_algorithm }}"
            state: absent
          loop: "{{ acme_challenge.challenge_data_dns | dict2items }}"
          delegate_to: localhost
          become: false

    # --- 6. Nginx へ配置 ---
    - name: Deploy fullchain
      ansible.builtin.copy:
        src: "{{ acme_dir }}/{{ primary_domain }}-fullchain.crt"
        dest: "{{ cert_dir }}/{{ primary_domain }}.crt"
        remote_src: true
        mode: "0644"
      notify: Reload nginx

    - name: Deploy private key
      ansible.builtin.copy:
        src: "{{ acme_dir }}/{{ primary_domain }}.key"
        dest: "{{ cert_dir }}/{{ primary_domain }}.key"
        remote_src: true
        owner: root
        mode: "0600"
      notify: Reload nginx

  handlers:
    - name: Reload nginx
      ansible.builtin.service:
        name: nginx
        state: reloaded
```

## root CA の証明書の取得とセット
playbook ディレクトリの下で、認証不要の公開エンドポイントから取得

```bash
export VAULT_ADDR=https://vault.labo.local:8200
curl -sk $VAULT_ADDR/v1/pki/ca/pem -o files/root_ca.crt
```

または、vault login が出来ていれば、以下のでも取得できる。
```bash
vault read -field=certificate pki/cert/ca > files/root_ca.crt
```

内容を確認するには
```bash
openssl x509 -in files/root_ca.crt -noout -subject -dates
```

## 実行

```bash
ansible-playbook -i inventory renew-cert.yml --ask-vault-pass
# 強制更新して動作確認する場合
ansible-playbook -i inventory renew-cert.yml --ask-vault-pass -e renew_before_days=9999
```

## 設計上のポイントと注意点

- **TXT の値は `challenge_data_dns` から取得しています。** キーはレコード名（例: `_acme-challenge.app.labo.local`）、値は TXT 値のリストです。ゾーン名を除いた相対名に変換して `nsupdate` に渡しています。
- **TXT の削除を `always` にしています。** 検証が失敗しても残骸が残りません。
- **`dns_propagation_wait` は DNS 構成に合わせて調整してください。** セカンダリへの反映遅延がある場合は長くするか、権威サーバーへ `dig +short TXT ... @<server>` で確認する再試行ループを入れます。
- **Vault から DNS が引けることが条件です。** 社内ゾーンは Vault ホストの既定リゾルバーでは引けないことが多く、その場合は上記の `dns_resolver` 設定が必要です。
- **ワイルドカードも dns-01 で可能です。** `domains` に `*.labo.local` を加えます。ロール側で許可（`allow_wildcard_certificates`）が必要で、TXT のレコード名は `_acme-challenge.labo.local` になります。
- **DNS の更新権限は最小限にしてください。** TSIG 鍵に `_acme-challenge` の TXT だけを許可しておくと、鍵が漏れても影響を限定できます。

## DNS が RFC 2136 でない場合の差し替え先

| DNS | 差し替えるモジュール |
|---|---|
| Route 53 | `amazon.aws.route53` |
| Azure DNS | `azure.azcollection.azure_rm_dnsrecordset` |
| Cloudflare | `community.general.cloudflare_dns` |
| PowerDNS | `community.general.pdns_*` 系、または API を `ansible.builtin.uri` で直接呼ぶ |
| CoreDNS | `etcd` プラグイン構成にして `etcdctl` で更新（「事前準備」の CoreDNS 側を参照） |
| 内部 DNS が dnsmasq のみ | 動的更新が使えないため、ゾーンファイルを `template` で更新する方式になる |

使っている DNS 製品を教えていただければ、手順3と5をその DNS 用に書き直します。