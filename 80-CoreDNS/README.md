# ns3: ns1 のレプリカとなる CoreDNS セカンダリDNSサーバー

| サーバー | アドレス | 役割 |
|---|---|---|
| ns1 | 192.168.1.9 | BIND9 プライマリ ([79-bind](../79-bind/README.md)) |
| ns3 | 192.168.1.245 | CoreDNS セカンダリ (本 playbook) |

## 構築手順

1. ns1 に [79-bind](../79-bind/README.md) の playbook を再実行する。ns3 の NS/A レコード、ns3 の IP からの転送許可、NOTIFY の宛先が反映される。
   ```
   ansible-playbook -i 79-bind/playbook/hosts-primary.yaml 79-bind/playbook/install.yaml
   ```
2. ns3 を作成する。`coredns.yaml` の `ansible:` により playbook が実行される。
   ```
   marmot server create -f coredns.yaml
   # または手動実行
   ansible-playbook -i playbook/hosts.yaml playbook/install.yaml
   ```

## 構成

| ファイル | 内容 |
|---|---|
| [coredns.yaml](coredns.yaml) | ns3 の仮想サーバー定義 |
| [playbook/install.yaml](playbook/install.yaml) | CoreDNS の導入と設定 |
| [playbook/templates/Corefile.j2](playbook/templates/Corefile.j2) | Corefile |
| [playbook/templates/coredns.service.j2](playbook/templates/coredns.service.j2) | systemd unit |

- CoreDNS は公式リリースのバイナリをダウンロードして導入する (バージョンと SHA-256 は `install.yaml` で固定)。
- 専用の `coredns` ユーザーで動作する (`CAP_NET_BIND_SERVICE` のみ付与)。
- `labo.local` と、逆引きゾーン `1.168.192` / `0.1.10` / `16.172` を ns1 から AXFR で複製する。
- ns1 に無い RFC1918 の逆引きは、外部に転送せず NXDOMAIN を返す。
- `hv0-4`、`k8s1-4` のサブゾーンは、ns1 と同じ宛先へ転送する。
- `acl` で `dns_trusted_networks` 以外からの問い合わせを拒否する (オープンリゾルバー対策)。
- メトリクスは `:9153` (Prometheus)、ヘルスチェックは `127.0.0.1:8080/health`、`127.0.0.1:8181/ready`。

## 注意点

- **TSIG 非対応:** CoreDNS の `secondary` プラグインは TSIG に対応していないため、ns1 は ns3 の IP からの転送を IP のみで許可する。
- **ゾーンの保持:** `secondary` はゾーンをディスクに保存しない。CoreDNS を再起動すると ns1 から再取得し、ns1 が停止していると再取得できるまで (10 秒ごとに再試行) 応答しない。
- **更新の反映:** ns1 の NOTIFY で即時に更新される。届かなくても SOA の refresh (30 分) で更新される。
- **待ち受けアドレス:** systemd-resolved のスタブ (`127.0.0.53:53`) と衝突しないよう、`127.0.0.1` と ns3 の IP だけで待ち受ける。
- **設定の検証:** CoreDNS には設定の検証フラグがないため、Corefile の変更後は playbook の確認タスクで検証する。
- **サブゾーンの同期:** `dns_forward_zones` は [79-bind の dns_stub_zones](../79-bind/playbook/vars/zones.yaml) と同じ内容にすること。

## 確認

```
dig @192.168.1.245 labo.local SOA +short     # ns1 と同じシリアルになる
dig @192.168.1.245 ca.labo.local +short
dig @192.168.1.245 -x 192.168.1.250 +short
dig @192.168.1.245 -x 192.168.2.1            # NXDOMAIN
```
