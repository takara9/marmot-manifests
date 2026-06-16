# Grafana のデプロイ

この Ansible プレイブックは、メトリックス分析のためのウェブUI画面 Grafanaをデプロイする。

Grahana の 公開ポート番号は 30000 に設定した。

http://node_ip_address:30000/ でアクセスできる。




### Step 1 ウェブサーバーへアクセス

https://k8s.labs.local:30000/

### Step 2　パスワードの取得

~~~
kubectl get secret --namespace prometheus grafana -o jsonpath="{.data.admin-password}" \
| base64 --decode ; echo
~~~


### Step 3 データソースの設定

* データソースに Prometheus を選択
* HTTP URL に　http://prometheus-server をセットする
* Save & Test をクリック


### Step 4 ダッシュボードをロード


* 11074 1 Node Exporter for Prometheus Dashboard English Version UPDATE 1102
* 10000 Cluster Monitoring for Kubernetes
* 8685 K8s Cluster Summary
* 1860 Node Exporter Full
* 7249 Kubernetes Cluster
