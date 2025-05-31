#!/bin/bash

CERT_DIR="../certs"
PASSWORD=changeit  # do not use in production! for testing only!

echo "Creating certs in: $CERT_DIR"
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# I. generating self-signed CA for server (brokers)

## 1. create a key-pair for the CA and store it in the server.ca.p12 file
## it will be used for signing certificates
keytool -genkeypair -keyalg RSA -keysize 2048 -keystore server.ca.p12 \
-storetype PKCS12 -storepass $PASSWORD -keypass $PASSWORD \
-alias ca -dname "CN=BrokerCA" -ext bc=ca:true -validity 365

## 2. Export CA's public certificate to server.ca.crt
## it will be included in the trust stores and certificate chains
keytool -export -file server.ca.crt -keystore server.ca.p12 \
-storetype PKCS12 -storepass $PASSWORD -alias ca -rfc


# II. creating keystore for the broker

## 1. Generating private key for broker and store it in the server.ks.p12 file
keytool -genkey -keyalg RSA -keysize 2048 -keystore server.ks.p12 \
-storepass $PASSWORD -keypass $PASSWORD -alias broker \
-storetype PKCS12 -dname "CN=broker,O=Confluent,C=GB" -validity 365

## 2. Generate the certificate signing request
keytool -certreq -file server.csr -keystore server.ks.p12 -storetype PKCS12 \
-storepass $PASSWORD -keypass $PASSWORD -alias broker

## 3. Use the CA keystore to sign the broker's certificate. Signed cert is stored in server.crt
keytool -gencert -infile server.csr -outfile server.crt \
-keystore server.ca.p12 -storetype PKCS12 -storepass $PASSWORD \
-alias ca -ext SAN=DNS:broker.example.com -validity 365

## 4. Import the broker's certificate chain into the broker's keystore
cat server.crt server.ca.crt > serverchain.crt
keytool -importcert -file serverchain.crt -keystore server.ks.p12 \
-storepass $PASSWORD -keypass $PASSWORD -alias broker \
-storetype PKCS12 -noprompt


# III. Generate brokers trust store

keytool -import -file server.ca.crt -keystore server.ts.p12 \
-storetype PKCS12 -storepass $PASSWORD -alias broker -noprompt


# IV. Generate trust store for clients with the broker's CA certificate

keytool -import -file server.ca.crt -keystore client.ts.p12 \
-storetype PKCS12 -storepass $PASSWORD -alias ca -noprompt


# V. Generating self-signed CA for clients

## 1. generate key-cert pair
keytool -genkeypair -keyalg RSA -keysize 2048 -keystore client.ca.p12 \
-storetype PKCS12 -storepass $PASSWORD -keypass $PASSWORD \
-alias ca -dname "CN=ClientCA" -ext bc=ca:true -validity 365

## 2. export certificate
keytool -export -file client.ca.crt -keystore client.ca.p12 -storetype PKCS12 \
-storepass $PASSWORD -alias ca -rfc

# VI. Create key store for clients

## 1. generate certificate
keytool -genkey -keyalg RSA -keysize 2048 -keystore client.ks.p12 \
-storepass $PASSWORD -keypass $PASSWORD -alias client \
-storetype PKCS12 -dname "CN=kafka-exporter,O=Confluent,C=GB" -validity 365

## 2. generate certificate signing request
keytool -certreq -file client.csr -keystore client.ks.p12 -storetype PKCS12 \
-storepass $PASSWORD -keypass $PASSWORD -alias client

## 3. sign certificate
keytool -gencert -infile client.csr -outfile client.crt \
-keystore client.ca.p12 -storetype PKCS12 -storepass $PASSWORD \
-alias ca -validity 365

## 4. import the chain
cat client.crt client.ca.crt > clientchain.crt
keytool -importcert -file clientchain.crt -keystore client.ks.p12 \
-storepass $PASSWORD -keypass $PASSWORD -alias client \
-storetype PKCS12 -noprompt

# VII. Add client CA certificate to broker's trust store

keytool -import -file client.ca.crt -keystore server.ts.p12 -alias client \
-storetype PKCS12 -storepass $PASSWORD -noprompt


# VIII. Creating Kafka credential file
cat <<EOF > broker_creds
$PASSWORD
EOF
