#!/bin/bash

CA_CERT=$1 # path with file extension .crt
CA_KEY=$2 # path with file extension .key

CERT_PATH=$3
CERT_DOMAIN=$4 # domain name, e.g. *.mail.ru
CERT_SN=$5 # cert's serial number
CERT_KEY=$6

openssl req -new \
    -key "${CERT_KEY}" \
    -subj "/CN=${CERT_DOMAIN}" \
    -addext "subjectAltName = DNS:${CERT_DOMAIN}, DNS:*.${CERT_DOMAIN}" \
    -sha256 | \
openssl x509 -req \
    -days 3650 \
    -CA "${CA_CERT}" -CAkey "${CA_KEY}" \
    -set_serial "${CERT_SN}" \
    -copy_extensions copy \
    -out "${CERT_PATH}.crt"