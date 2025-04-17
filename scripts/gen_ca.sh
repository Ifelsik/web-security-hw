#!/bin/sh

cert_name=$1

openssl req -x509 -newkey rsa:2048 \
    -keyout "${cert_name}.key" \
    -out "${cert_name}.crt" \
    -days 3650 \
    -nodes \
    -subj "/C=RU/ST=Moscow/L=Moscow/CN=Ifelsik" \
    -sha256
