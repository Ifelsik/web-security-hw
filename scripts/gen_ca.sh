#!/bin/sh

CERT_NAME=$1

openssl req -x509 -newkey rsa:2048 \
    -keyout "${CERT_NAME}.key" \
    -out "${CERT_NAME}.crt" \
    -days 3650 \
    -nodes \
    -subj "/C=RU/ST=Moscow/L=Moscow/CN=Ifelsik" \
    -sha256
