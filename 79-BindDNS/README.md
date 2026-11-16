# BIND9 DNS サーバー (ns1: プライマリ / ns2: セカンダリ)

ns1 (hmc) で動いていた CoreDNS の設定を BIND9 に引き継いだ構成。

| サーバー | アドレス | 役割 |
|---|---|---|
| ns1 (hmc) | 192.168.1.9 | BIND9 プライマリ。CoreDNS を置き換える |
| ns2 | 192.168.1.244 | BIND9 セカンダリ。TSIG 認証のゾーン転送で複製する |
| ns3 | 192.168.1.245 | CoreDNS セカンダリ。[80-coredns](../80-coredns/README.md) を参照 |

## 構成

| ファイル | 内容 |
|---|---|
| [bind.yaml](bind.yaml) | ns2 の仮想サーバー定義。作成時に playbook を実行する |
| [playbook/install.yaml](playbook/install.yaml) | 構築 playbook。インベントリの `dns_role` で役割が決まる |
| [playbook/hosts.yaml](playbook/hosts.yaml) | ns2 (secondary) |
| [playbook/hosts-primary.yaml](playbook/hosts-primary.yaml) | ns1 (primary) |
| [playbook/vars/zones.yaml](playbook/vars/zones.yaml) | `labo.local` のレコード。旧 `/coredns/labo.db` から移行 |
| playbook/files/xfr.key | ゾーン転送用の TSIG 鍵。初回実行時に自動生成する。Git 管理外 |

## 構築手順

必ず ns1 を先に実行する。

```
ansible-playbook -i playbook/hosts-primary.yaml playbook/install.yaml   # ns1 (CoreDNS から切り替え)
ansible-playbook -i playbook/hosts.yaml         playbook/install.yaml   # ns2
```

ns1 の切り替えは次の順で行う。

1. BIND9 をインストールする。このとき `policy-rc.d` で named の自動起動を止め、CoreDNS の 53 番ポートと衝突させない。
2. 設定とゾーンを配置し、`named-checkconf -z` で検証する。
3. CoreDNS を停止・無効化し、BIND を起動する。
4. BIND が応答しなければ、自動で CoreDNS に戻して失敗する。

`/coredns` 配下の Corefile と labo.db は削除しない。手動で戻すには次を実行する。

```
sudo systemctl disable --now named && sudo systemctl enable --now coredns
```

TSIG 鍵 (`playbook/files/xfr.key`) は ns1 と ns2 で同じものを使う。2 つの playbook は同じ場所から実行すること。

ns3 (CoreDNS) は TSIG に対応していないため、ns1 は `dns_ip_secondary_ips` の IP (192.168.1.245) からの転送を IP のみで許可する。
ns3 を追加・変更した場合は、ns1 の playbook を再実行すること。

## CoreDNS 設定の問題点と対処

| 問題点 | 対処 |
|---|---|
| オープンリゾルバー | `acl "trusted"` で問い合わせ・再帰・キャッシュ参照を `dns_trusted_networks` のみに制限 |
| root で動作 | named は `bind` ユーザーで動作 |
| ゾーン転送が IP 制限のみ | TSIG 鍵で認証 (`allow-transfer { key xfr-key; }`) |
| `ns1` が CNAME で SOA/NS に使えない | `ns1` を A レコードにし、NS を `ns1`/`ns2` に変更 |
| 逆引きゾーンがなく PTR が外部に漏れる | `1.168.192` / `0.1.10` / `16.172` の各逆引きゾーンを A レコードから自動生成。他の RFC1918 は BIND の empty zone が NXDOMAIN を返す |
| シリアルが手作業の連番 | `dns_zone_serial` を `YYYYMMDDNN` 形式に変更 |
| バックアップが古い | レコードを Git 管理 (`vars/zones.yaml`) |
| サブゾーン転送先が単一 / `errors` が無効 | `static-stub` で同じ宛先に転送。失敗は named のログ (`journalctl -u named`) に出る |
| `cache` なし | BIND は標準でキャッシュする (`max-cache-size`) |
| 監視なし | `statistics-channels` (127.0.0.1:8053) と `prometheus-bind-exporter` (:9119) を導入 |
| Corefile の変更に再起動が必要 | 設定やゾーンが変わると playbook が named を再起動する |
| 外部の転送先が 2 つだけ | `192.168.1.1` (ルーター) を追加。`forward first` のため全滅時は再帰解決に切り替わる |
| unit の起動引数が不自然 | 標準の named.service を使用 |
| クライアントが ns1 しか使っていない | 下記「運用上の注意」を参照 |

## 運用上の注意

- **レコードの変更:** [playbook/vars/zones.yaml](playbook/vars/zones.yaml) を編集し、`dns_zone_serial` を増やして ns1 の playbook を再実行する。ns2 には NOTIFY で自動反映される。
- **同じ IP に複数の名前:** PTR を付けない側に `ptr: false` を指定する (例: `ns1` は `hmc` と同じ IP)。
- **逆引きゾーンの追加:** `dns_reverse_zones` に `zone` と `prefix` を追加する。
- **クライアントの設定:** DHCP や cloud-init で DNS を 192.168.1.9 と 192.168.1.244 の 2 つ配ること。
- **Prometheus:** ns1/ns2 の `:9119` を scrape 対象に追加する。
- **サブゾーンの転送先:** `hv0-4`、`k8s1-4` は各 1 台のみで冗長化されていない。ns2 から `10.1.0.x` と `172.16.x.x` に届くことを確認すること。
- **ドメイン名:** `.local` は mDNS (RFC 6762) の予約名で、systemd-resolved や avahi を使うクライアントでは DNS ではなく mDNS で解決されることがある。今回はドメインを変更していない。`home.arpa` などへの移行は別途検討すること。

## 確認

```
dig @192.168.1.9   labo.local SOA +short
dig @192.168.1.244 labo.local SOA +short     # 同じシリアルになる
dig @192.168.1.9   -x 192.168.1.250 +short   # ca.labo.local.
dig @192.168.1.9   labo.local AXFR           # 鍵なしは Transfer failed
```
