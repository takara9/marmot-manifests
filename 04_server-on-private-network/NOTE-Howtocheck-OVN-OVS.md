以下を「各ノードで」実行すると、Geneveトンネルの健全性をOVN/OVSレベルで確認できます。

1. OVNがGeneveカプセル化を認識しているか
```bash
ovn-sbctl list encap
ovn-sbctl --columns=name,hostname,encaps list chassis
```
確認ポイント:
- `type: geneve` が出る
- 各Chassisに `encaps` が紐づいている

2. OVS側にGeneveポートが作られているか
```bash
ovs-vsctl --columns=name,type,options,ofport find Interface type=geneve
ovs-vsctl list Interface | egrep 'name|type|options|ofport|error'
```
確認ポイント:
- `type: geneve`
- `options: {remote_ip=..., key=flow}`（環境により差異あり）
- `ofport` が `-1` でない
- `error` が空

3. ブリッジ/トンネルの実体確認
```bash
ovs-vsctl show
ovs-appctl ofproto/list-tunnels br-int
ovs-appctl dpif/show
```
確認ポイント:
- `br-int` 上にGeneveトンネル関連が見える
- `dpif/show` でtunnel情報が見える

4. Geneve(UDP/6081)が受信できているか
```bash
ss -lunp | grep 6081
tcpdump -ni <underlay-if> udp port 6081
```
確認ポイント:
- `ss` で6081待受/関連プロセスが見える
- `tcpdump` で対向ノードとの6081トラフィックが流れる

5. OVS datapathフローにトンネル情報が出るか
```bash
ovs-appctl dpctl/dump-flows -m | egrep 'geneve|tun_id|tunnel'
```
確認ポイント:
- `tun_id` や `tunnel(...)` を含むフローがある

6. OVN論理経路の確認（必要時）
```bash
ovn-trace <NB_DB_NAME or logical-switch> 'inport=="<lsp-name>" && eth.src==<mac> && eth.dst==<mac> && ip4.src==<ip> && ip4.dst==<ip>'
```
確認ポイント:
- 論理パイプラインでdropせず、期待経路を通る

最短でまず見るなら、`ovn-sbctl list encap` → `ovs-vsctl find Interface type=geneve` → `tcpdump udp port 6081` の3点セットが有効です。必要なら、あなたの環境名（ブリッジ名/IF名）に合わせてコマンドを具体化します。