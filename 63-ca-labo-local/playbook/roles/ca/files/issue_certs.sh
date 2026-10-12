#!/bin/bash

export MY_DOMAIN=labo.local

#
export MY_HOST=ca
export FQDN=${MY_HOST}.${MY_DOMAIN}
export CSR_SUBJ="/C=JP/ST=Tokyo/OU=CA/O=System Labo/CN=${FQDN}"
export V3EXT=v3ext_ca.txt
#./create_csr.sh
./create_csr_noip.sh
./signature_cert.sh

#
export MY_HOST=w1
export FQDN=${MY_HOST}.${MY_DOMAIN}
export CSR_SUBJ="/C=JP/ST=Tokyo/OU=Web Service/O=System Labo/CN=${MY_DOMAIN}"
export V3EXT=v3ext.txt
export SAN_HOSTS="{$MY_HOST}.${MY_DOMAIN}"
./create_csr_noip.sh
./signature_cert.sh

#
export MY_HOST=w2
export FQDN=${MY_HOST}.${MY_DOMAIN}
export CSR_SUBJ="/C=JP/ST=Tokyo/OU=Web Service/O=System Labo/CN=${MY_DOMAIN}"
export V3EXT=v3ext.txt
export SAN_HOSTS="{$MY_HOST}.${MY_DOMAIN}"
./create_csr_noip.sh
./signature_cert.sh

