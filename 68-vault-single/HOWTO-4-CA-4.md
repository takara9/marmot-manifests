# Kubernetes 運用における Root CA 信頼の考慮点

[HOWTO-4-CA-3.md](HOWTO-4-CA-3.md) では Windows / Linux / macOS のクライアントに Vault の Root CA を登録しました。
Kubernetes では、信頼させる対象が「ノード OS」「Pod 内のアプリケーション」「クラスタ内部の PKI」の 3 層に分かれます。**どれか 1 つに登録しても、他の層には効きません。**

## 1. ノード OS（kubelet / コンテナランタイム）

- **対象:** イメージ pull 時の TLS 検証です。プライベート CA で運用しているレジストリ（Harbor、GitLab Registry など）を使う場合に必要です。
- **登録先:** 全ワーカーノードとコントロールプレーンに、HOWTO-4-CA-3.md の Linux 手順（`update-ca-certificates` など）で Root CA を登録します。
- **ランタイムの再起動:** containerd や CRI-O は起動時に信頼ストアを読み込みます。登録後に再起動が必要です。
- **レジストリごとの設定（containerd）:** `/etc/containerd/certs.d/<registry>/hosts.toml` の `ca = "..."` で、レジストリ単位に CA を指定できます。
- **ノード増設時の漏れ防止:** cloud-init、Ansible、特権 DaemonSet などで配布を自動化します。

## 2. Pod 内のアプリケーション

- **ノードの CA は Pod に引き継がれません。** コンテナはイメージ内の信頼ストアを使い、その場所はベースイメージ（alpine、distroless、debian など）ごとに異なります。
- **配布方法:**
  - ConfigMap に Root CA を入れて Pod にマウントし、`SSL_CERT_FILE`、`REQUESTS_CA_BUNDLE`、`NODE_EXTRA_CA_CERTS` などで指定します。
  - イメージに焼き込む方法もありますが、CA 更新のたびにイメージの再ビルドが必要です。
- **trust-manager（cert-manager 関連）:** `Bundle` リソースで、全 Namespace に ConfigMap として CA を自動配布できます。クラスタ内の複数アプリで CA を共有する場合に最も運用しやすい方法です。
- **言語ごとの差:** Java は独自の `cacerts`（JKS/PKCS12）を使うため、`keytool` などでの登録が必要です。Go は `SSL_CERT_FILE` や `/etc/ssl/certs` を参照します。
- **配布するのは公開証明書のみ:** バンドルには Root CA（必要なら中間CA）だけを入れ、秘密鍵は含めません。

## 3. クラスタ内部の PKI（Vault の CA とは別物）

- Kubernetes 自身のクラスタ CA は、Vault の Root CA とは別です。主に次を署名しています。
  - kube-apiserver の証明書
  - kubelet の証明書
  - etcd の証明書
  - service account まわり
- **kubeconfig の `certificate-authority-data`** は、このクラスタ CA を指します。
- **kube-apiserver の外向き証明書に Vault の CA を使う場合:**
  - `--tls-cert-file` を、Vault 発行の証明書に差し替えます。
  - kubectl を使うクライアントに、その Root CA を信頼させる必要があります。
  - 内部のクラスタ CA は、コンポーネント間通信用として残します。
- **kubeadm で外部 CA を使う場合:** クラスタ CA 自体を Vault の中間CAから発行する構成も可能です。その場合は、Root CA の信頼チェーンが内部通信にも及びます。

## 4. Ingress / cert-manager と Vault の連携

- **cert-manager の Vault Issuer:** `Issuer` / `ClusterIssuer` に Vault の署名エンドポイント（`pki_int/sign/...`）を設定すると、Ingress や Pod 用の証明書を自動で発行・更新できます。
- **Vault への接続自体の信頼:** Issuer には `caBundle` で、Vault サーバーの TLS を検証するための CA を指定します。Vault の証明書が自前 CA 発行なら、ここにも Root CA が必要です。
- **`ca_chain` を含める:** `certificate` と `ca_chain` を連結したものを `tls.crt` として Secret に入れます。中間CAを含めないと、クライアントの検証に失敗します。
- **TTL とローテーション:** [HOWTO-4-CA-2.md](HOWTO-4-CA-2.md) の短 TTL（24h〜720h）の証明書は cert-manager が更新します。Pod 側にもリロードが必要なため、reloader などの利用を検討します。

## 5. Root CA 更新（ローテーション）時の注意

- **信頼の重なり期間を設ける:** 新 Root CA を旧 Root CA と並べてバンドルに追加し、全ノードと全 Pod に配布してから、新 Root からの発行へ切り替えます。旧 Root は、旧証明書がすべて失効してから削除します。
- **更新箇所が多い:** ノード OS、ConfigMap / trust-manager、イメージ焼き込み、レジストリごとの設定、CI の runner、kubeconfig など多岐にわたります。事前に一覧化しておきます。
- **有効期限の監視:** Root CA の期限切れは全面障害につながります。期限の十分前から更新を始めます。

## 6. セキュリティ面

- **信頼範囲の最小化:** Root CA は任意のドメインの証明書を署名できます。Vault ロールの `allowed_domains` を絞り、中間CAに Name Constraints（`permitted_dns_domains` など）を設定して `labo.local` に限定することを検討します。
- **Root CA の秘密鍵:** Vault に置いたままにせず、中間CAへの署名後はオフラインで保管します。運用では中間CAのみを稼働させます。
- **Secret の扱い:** 発行した `private_key` は Kubernetes Secret に格納されます。RBAC を絞り、etcd の暗号化（EncryptionConfiguration）を有効にします。

## 7. 推奨構成（まとめ）

| 層 | 方法 |
|---|---|
| ノード OS・ランタイム | Ansible / cloud-init で Root CA を配布し、containerd を再起動 |
| Pod | trust-manager で ConfigMap を配布し、環境変数またはマウントで指定 |
| 証明書発行 | cert-manager + Vault Issuer（中間CAから発行、`ca_chain` を含める） |
| ローテーション | 旧・新 Root を併存させて配布し、段階的に切り替える |
