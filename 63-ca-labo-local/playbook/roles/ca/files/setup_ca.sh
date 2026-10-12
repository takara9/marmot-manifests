#!/bin/bash

. environment

##
## 認証局　自己署名証明書 作成
##
# CN  URLのFQDN      例 www.yahoo.co.jp
# OU  組織単位名     例 Technical Division
# O   会社や団体名   例 ACME Japan Co.,Ltd.
# L   市区町村名     例 Nihonbash,Hakozaki-cho,Chuo-ku
# ST  都道府県名     例 Tokyo
# C   国             例 JP
CA_SUBJ="/C=JP/ST=Tokyo/OU=Certification Authority Services/O=System Labo/CN=System Labo Environment"

#
# 認証局の開設時のみ必要
#
echo プライベート認証局開設

if [ -d $CA_HOME ]; then
    echo "認証局は設定されています"
else
    mkdir -p $CA_HOME
    openssl genrsa -out "$CA_HOME/ca.key" 4096
    openssl req -x509 -new -nodes -sha512 -days 4000 \
        -subj "$CA_SUBJ" \
        -key "$CA_HOME/ca.key" \
        -out "$CA_HOME/ca.crt"
    touch ~/.rnd
fi
