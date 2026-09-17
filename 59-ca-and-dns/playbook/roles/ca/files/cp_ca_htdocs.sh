#!/bin/bash
. environment
cp  "$CA_HOME/ca.crt" "$CRT_HOME/ca.pem"
chmod a+r "$CRT_HOME/ca.pem"
openssl x509 -text -fingerprint -noout -in "$CRT_HOME/ca.pem" >  "$CRT_HOME/ca_pem.txt"

openssl pkcs12 -export \
    -out "$CRT_HOME/ca.p12" \
    -in "$CA_HOME/ca.crt" \
    -inkey "$CA_HOME/ca.key" \
    -passin pass:root \
    -passout pass:root

chmod a+r "$CRT_HOME/ca.p12"


cp  "$CA_HOME/ca.key" "$CRT_HOME/ca.key"
chmod a+r "$CRT_HOME/ca.key"
