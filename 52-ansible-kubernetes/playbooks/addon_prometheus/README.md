# Grafana and Prometheus

http://grafana.monitoring.k8s1.labo.local/login

username admin
password admin


kubectl create ns monitoring
kubectl config set-context mon --namespace=monitoring --cluster=kubernetes --user=kubernetes-admin
kubectl config use-context mon
kubectl config get-contexts


git clone -b release-0.10 https://github.com/prometheus-operator/kube-prometheus
cd kube-prometheus
kubectl apply --server-side -f manifests/setup
kubectl apply -f manifests/

kubectl patch service grafana  -n monitoring --type merge --patch-file patch-file.yaml

    ports:
    - name: https
      port: 8443
      protocol: TCP
      targetPort: https