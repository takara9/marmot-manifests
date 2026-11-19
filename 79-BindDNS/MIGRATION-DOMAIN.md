# labo.local から lab.internal への移行

`.local` は mDNS (RFC 6762) の予約名で、systemd-resolved や avahi を使うクライアントでは DNS ではなく mDNS で解決されることがある。
そのため `lab.internal` を追加し、最終的に `lab.internal` へ一本化する。移行には時間がかかるため、最初は2つのドメインを並行運用する。

## 方針

- [playbook/vars/zones.yaml](playbook/vars/zones.yaml) の `dns_domains` に並べたドメインごとにゾーンを作る。
- A / CNAME / MX レコードは全ドメインで同じ内容になる。レコードの変更は1か所の編集で両方に反映される。
- 逆引き (PTR) は1つのIPに対して1つの名前しか返せないため、`dns_ptr_domain` で指定したドメイン名だけを返す。
- 並行運用中は `dns_ptr_domain` を `labo.local` のままにして、既存クライアントの逆引き結果を変えない。

```yaml
dns_domains:
  - labo.local
  - lab.internal
dns_ptr_domain: labo.local
```

## 対象サーバー

| サーバー | アドレス | 対応 |
|---|---|---|
| ns1 (hmc) | 192.168.1.9 | BIND9 プライマリ。両ゾーンを配信する |
| ns2 | 192.168.1.244 | BIND9 セカンダリ。TSIG 認証のゾーン転送で両ゾーンを複製する |
| ns3 | 192.168.1.245 | CoreDNS セカンダリ。`dns_secondary_zones` に `lab.internal` を追加済み |

## フェーズ1: 並行運用の開始

1. `dns_zone_serial` を増やす ([install.yaml](playbook/install.yaml))。
2. 次の順で playbook を実行する。ns1 を必ず先に実行すること。

   ```
   ansible-playbook -i playbook/hosts-primary.yaml playbook/install.yaml   # ns1
   ansible-playbook -i playbook/hosts.yaml         playbook/install.yaml   # ns2
   ansible-playbook -i 80-CoreDNS/playbook/hosts.yaml 80-CoreDNS/playbook/install.yaml   # ns3
   ```

3. 確認する。

   ```
   dig @192.168.1.9   lab.internal SOA +short     # 全サーバーで同じシリアルになる
   dig @192.168.1.244 lab.internal SOA +short
   dig @192.168.1.245 lab.internal SOA +short
   dig @192.168.1.9   ca.lab.internal +short      # 192.168.1.250
   dig @192.168.1.9   -x 192.168.1.250 +short     # ca.labo.local. (dns_ptr_domain のまま)
   ```

## フェーズ2: クライアントの切り替え

並行運用中に、クライアントや各サーバーの設定を `lab.internal` へ順に切り替える。

- DHCP / cloud-init の検索ドメイン (`search`) と、証明書の SAN、各種設定ファイルに書かれた FQDN を `lab.internal` に変更する。
- 切り替えは1か所ずつ行い、`labo.local` でも引き続き解決できる状態を保つ。
- 次の検索で、`labo.local` を参照しているファイルを洗い出す。

  ```
  grep -rn "labo.local" .
  ```

- 全クライアントが `lab.internal` に切り替わり、`labo.local` への問い合わせがなくなったことを確認してから、フェーズ3に進む。DNS サーバーのクエリログや、BIND exporter (`:9119`) のメトリクスで確認できる。

## フェーズ3: labo.local の廃止

[zones.yaml](playbook/vars/zones.yaml) を次のように変更する。

1. `dns_ptr_domain` を `lab.internal` に変更する。
2. `dns_domains` から `labo.local` を削除する。
3. `dns_stub_zones` の `hv0`〜`hv4` を `lab.internal` 側のサブゾーンに置き換える。

その後、次を行う。

1. `dns_zone_serial` を増やし、ns1 → ns2 の順に playbook を再実行する。`labo.local` が `named.conf.local` から外れる。
2. ns3 ([80-CoreDNS/playbook/install.yaml](../80-CoreDNS/playbook/install.yaml)) を更新して再実行する。
   - `dns_secondary_zones` から `labo.local` を削除する。
   - `dns_domain` を `lab.internal` に変更する (確認タスクで使用)。
   - `dns_forward_zones` を `dns_stub_zones` と同じ内容に合わせる。
3. ns1 / ns2 に残った旧ゾーンファイル (ns1 の `/var/lib/bind/db.labo.local`、ns2 の `/var/cache/bind/db.labo.local`) を必要に応じて削除する。

## 注意点

- **サブゾーン (hv0〜hv4):** `dns_stub_zones` は `labo.local` 配下のみで、転送先 DNS は `lab.internal` の名前を持たない。`lab.internal` 側のサブゾーンは、転送先の DNS が対応してから追加する。
- **逆引きの切り替え:** `dns_ptr_domain` を変更すると、全 PTR が新しいドメイン名に変わる。変更前に、逆引き結果に依存するログ解析や設定がないか確認する。
- **ns3 の同期:** ns3 の確認タスクは `ca.<dns_domain>` の正引きと逆引きを調べる。`dns_ptr_domain` を変更したら、`dns_domain` も合わせること。
- **ロールバック:** フェーズ3の前なら、`labo.local` のゾーンは残っているため、クライアント側の設定を戻すだけでよい。フェーズ3の後は、`zones.yaml` の変更を戻して playbook を再実行する。
- **ns3 レコードの重複:** `zones.yaml` の `ns3` は重複して定義されているが、同じ内容のため動作に影響しない。
