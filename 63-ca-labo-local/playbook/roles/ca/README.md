## 使用法メモ

issue_cert.shを編集して実行



#
# Dockerdへの証明書追加
#
mkdir -p "/etc/docker/certs.d/{{ my_domain }}"
cp "{{ my_domain }}.crt" "/etc/docker/certs.d/{{ my_domain }}/{{ my_domain }}.cert"
cp "{{ my_domain }}.key" "/etc/docker/certs.d/{{ my_domain }}"
cp ca.crt "/etc/docker/certs.d/{{ my_domain }}"
