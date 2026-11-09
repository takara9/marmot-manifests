# Ansible で ACME 経由の Nginx 証明書更新

**可能です。** `community.crypto` コレクションの `acme_certificate` モジュールが ACME（RFC 8555）に対応しており、Vault の ACME エンドポイントを `acme_directory` に指定すれば、標準のクライアントとして使えます。certbot などを入れる必要はありません。

## 構成と前提

| 項目 | 内容 |
|---|---|
| 実行場所 | Nginx を動かすホスト上（`hosts: nginx_servers`）。チャレンジファイルをそのホストの webroot に置くため |
| チャレンジ | http-01（Vault → `app.labo.local:80` に到達できること） |
| 必要なもの | `ansible-galaxy collection install community.crypto`、対象ホストの Python `cryptography` |
| Vault の TLS | 対象ホストの OS の信頼ストアにルートCAを登録済みであること（下記タスク1） |
| 冪等性 | 残り日数が `remaining_days` 以下のときだけ更新する。普段の実行では何も変わらない |

## Playbook 例

```yaml
---
- name: Renew app.labo.local certificate via Vault ACME
  hosts: nginx_servers
  become: true
  vars:
    domain: app.labo.local
    acme_directory: https://vault.labo.local:8200/v1/pki_int/roles/labo-local/acme/directory
    acme_dir: /etc/ssl/acme
    webroot: /var/www/html
    cert_dir: /etc/nginx/ssl
    renew_before_days: 10          # ロール ttl=720h なら発行後およそ20日で更新

  tasks:
    # 1) Vault の TLS を信頼させる（ルートCAを OS に登録）
    - name: Install root CA
      ansible.builtin.copy:
        src: root_ca.crt
        dest: /usr/local/share/ca-certificates/labo-root-ca.crt
        mode: "0644"
      register: root_ca

    - name: Update CA store
      ansible.builtin.command: update-ca-certificates
      when: root_ca is changed

    # 2) 鍵と CSR の準備
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
        path: "{{ acme_dir }}/{{ domain }}.key"
        type: ECC
        curve: secp256r1

    - name: CSR
      community.crypto.openssl_csr:
        path: "{{ acme_dir }}/{{ domain }}.csr"
        privatekey_path: "{{ acme_dir }}/{{ domain }}.key"
        common_name: "{{ domain }}"
        subject_alt_name: "DNS:{{ domain }}"

    # 3) ACME 発行（第1段階: チャレンジ情報の取得）
    - name: Request certificate (step 1)
      community.crypto.acme_certificate:
        acme_version: 2
        acme_directory: "{{ acme_directory }}"
        validate_certs: true
        account_key_src: "{{ acme_dir }}/account.key"
        terms_agreed: true
        challenge: http-01
        csr: "{{ acme_dir }}/{{ domain }}.csr"
        dest: "{{ acme_dir }}/{{ domain }}.crt"
        fullchain_dest: "{{ acme_dir }}/{{ domain }}-fullchain.crt"
        remaining_days: "{{ renew_before_days }}"
      register: acme_challenge

    # 4) チャレンジファイルを配置し、検証を完了（第2段階）
    - name: Place http-01 challenge file
      ansible.builtin.copy:
        dest: "{{ webroot }}/{{ acme_challenge.challenge_data[domain]['http-01'].resource }}"
        content: "{{ acme_challenge.challenge_data[domain]['http-01'].resource_value }}"
        mode: "0644"
      when: acme_challenge is changed and domain in acme_challenge.challenge_data

    - name: Finish validation and fetch certificate (step 2)
      community.crypto.acme_certificate:
        acme_version: 2
        acme_directory: "{{ acme_directory }}"
        validate_certs: true
        account_key_src: "{{ acme_dir }}/account.key"
        terms_agreed: true
        challenge: http-01
        csr: "{{ acme_dir }}/{{ domain }}.csr"
        dest: "{{ acme_dir }}/{{ domain }}.crt"
        fullchain_dest: "{{ acme_dir }}/{{ domain }}-fullchain.crt"
        data: "{{ acme_challenge }}"
        remaining_days: "{{ renew_before_days }}"
      when: acme_challenge is changed and domain in acme_challenge.challenge_data

    # 5) Nginx へ配置（変更があるときだけ reload）
    - name: Deploy fullchain
      ansible.builtin.copy:
        src: "{{ acme_dir }}/{{ domain }}-fullchain.crt"
        dest: "{{ cert_dir }}/{{ domain }}.crt"
        remote_src: true
        mode: "0644"
      notify: Reload nginx

    - name: Deploy private key
      ansible.builtin.copy:
        src: "{{ acme_dir }}/{{ domain }}.key"
        dest: "{{ cert_dir }}/{{ domain }}.key"
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

Nginx 側は前回と同じく、`ssl_certificate /etc/nginx/ssl/app.labo.local.crt;` と `ssl_certificate_key /etc/nginx/ssl/app.labo.local.key;` を指定し、80番で `/.well-known/acme-challenge/` が `webroot` から配信されるようにしておきます。

## 実行方法

```bash
ansible-galaxy collection install community.crypto
ansible-playbook -i inventory renew-cert.yml --check   # まず差分確認（チャレンジ処理は dry-run では一部スキップされる）
ansible-playbook -i inventory renew-cert.yml
```

## 定期実行

この Playbook は「期限が近いときだけ更新する」作りなので、**毎日実行して問題ありません**。

| 方法 | 内容 |
|---|---|
| cron / systemd timer | コントロールノードから `ansible-playbook` を毎日実行 |
| AWX / Ansible Automation Platform | スケジュールジョブとして登録 |
| `ansible-pull` | 各 Nginx ホストが自分で Playbook を取得して実行 |

## 注意点

- **Vault の TLS を信頼させる手順は必須です。** タスク1を省くと、`acme_certificate` が Vault の証明書検証で失敗します。`validate_certs: false` は避けてください。
- **`remaining_days` はロールの `ttl` に合わせます。** TTL が 30日なら 10日前後を目安にします。TTL が短い（数日）ほど、Playbook の実行頻度を上げる必要があります。
- **Playbook の実行失敗は、Nginx が起動していること、webroot とチャレンジ URL の整合、Vault から 80番への到達性を確認します。** 前回までの確認項目と共通です。
- **`--check` ではチャレンジファイルの配置や第2段階が実行されないことがあります。** 本番の確認には、一度実際に実行し、`remaining_days` を大きくして更新を強制させる方法が確実です（例: `-e renew_before_days=9999`）。
- **別案として、ACME を使わず Vault の API で直接発行する方法もあります。** `community.hashi_vault` コレクションで `pki_int/issue/labo-local` を呼びます。ACME のチャレンジ処理が不要になる一方、Vault のトークン管理（AppRole 等）が必要になります。

Nginx の複数サーバー、SAN（複数ドメイン）、dns-01 に対応する形が必要でしたら、その前提に合わせて Playbook を調整します。