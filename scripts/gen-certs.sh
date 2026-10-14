#!/usr/bin/env bash
set -euo pipefail
OUT=certs; mkdir -p $OUT; cd $OUT

openssl genrsa -out ca.key 4096
openssl req -x509 -new -key ca.key -sha256 -days 3650 -subj "/CN=shop-ca" -out ca.crt

for svc in gateway user-service product-service order-service payment-service; do
  openssl genrsa -out $svc.key 2048
  openssl req -new -key $svc.key -subj "/CN=$svc" -out $svc.csr
  printf "subjectAltName=DNS:$svc,DNS:localhost\nextendedKeyUsage=serverAuth,clientAuth\n" > $svc.ext
  openssl x509 -req -in $svc.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -days 365 -sha256 -extfile $svc.ext -out $svc.crt
  rm $svc.csr $svc.ext
done
chmod 600 *.key