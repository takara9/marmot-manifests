#!/bin/bash

export MY_DOMAIN=dify.local

#
export MY_HOST=ca
export FQDN=$MY_HOST.$MY_DOMAIN
export CSR_SUBJ="/C=JP/ST=Tokyo/OU=CA/O=System Labo/CN=$FQDN"
export V3EXT=v3ext_ca.txt
#./create_csr.sh
./create_csr_noip.sh
./signature_cert.sh


# dify.local 配下の全サービスを1枚の証明書(SAN)にまとめて発行する
export MY_HOST=dify
export FQDN=$MY_DOMAIN
export CSR_SUBJ="/C=JP/ST=Tokyo/OU=Dify Service/O=System Labo/CN=$MY_DOMAIN"
export V3EXT=v3ext.txt
export SAN_HOSTS="console.$MY_DOMAIN app.$MY_DOMAIN api.$MY_DOMAIN upload.$MY_DOMAIN enterprise.$MY_DOMAIN trigger.$MY_DOMAIN"
./create_csr_noip.sh
./signature_cert.sh


