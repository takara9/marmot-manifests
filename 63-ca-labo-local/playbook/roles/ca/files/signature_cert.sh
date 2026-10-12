#!/bin/bash
##
## 証明書 署名
##
. environment

WORK_HOME=$CRT_HOME/$FQDN
. "$WORK_HOME/info.txt"

echo "サーバー証明書 署名"
echo FQDN: $FQDN

if [ ! -f "${WORK_HOME}/${FQDN}.serial" ]; then
    openssl rand -hex 10 > "${WORK_HOME}/${FQDN}.serial"
fi

if [ -f "${WORK_HOME}/${FQDN}.crt" ]; then
    echo "証明書は作成済みです"
else
    openssl x509 -req -sha512 -days 825 \
            -extfile "${WORK_HOME}/v3ext" \
	    -CAserial "${WORK_HOME}/${FQDN}.serial" \
            -CA "${CA_HOME}/ca.crt" -CAkey "${CA_HOME}/ca.key" \
            -in "${WORK_HOME}/${FQDN}.csr" \
            -out "${WORK_HOME}/${FQDN}.crt"

    openssl pkcs12 -export \
	    -out "${WORK_HOME}/${FQDN}.p12" \
	    -in "${WORK_HOME}/${FQDN}.crt" \
	    -inkey "${WORK_HOME}/${FQDN}.key" \
	    -certfile "${CA_HOME}/ca.crt" \
	    -passin pass:root \
	    -passout pass:root

    # IIS はインポート時に .pfx 拡張子を期待するため、同一内容を別名で用意
    cp "${WORK_HOME}/${FQDN}.p12" "${WORK_HOME}/${FQDN}.pfx"
fi


openssl req -text -noout -in "${WORK_HOME}/${FQDN}.csr" > "${WORK_HOME}/${FQDN}_csr.txt"
openssl x509 -text -fingerprint -noout -in "${WORK_HOME}/${FQDN}.crt" >  "${WORK_HOME}/${FQDN}_cert.txt"
chmod a+r "${WORK_HOME}/${FQDN}.key"


