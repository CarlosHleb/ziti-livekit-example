#!/bin/bash

source .env

# livekit
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
 -keyout ./store/livekit.key \
 -out ./store/livekit.crt -subj "/CN="${ZITI_SERVICE_LIVEKIT} \
 -addext "subjectAltName=DNS:"${ZITI_SERVICE_LIVEKIT}",IP:${ZITI_SERVICE_LIVEKIT_ADDRESS}"

# livekit nginx server - necessary so livekit-egress-server can connect to livekit api
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
 -keyout ./store/livekit-nginx-server.key \
 -out ./store/livekit-nginx-server.crt -subj "/CN=livekit-nginx-server" \
 -addext "subjectAltName=DNS:livekit-nginx-server,IP:0.0.0.0"

# turn
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
 -keyout ./store/turn.key \
 -out ./store/turn.crt -subj "/CN="${ZITI_SERVICE_TURN} \
 -addext "subjectAltName=DNS:"${ZITI_SERVICE_TURN}",IP:"${ZITI_SERVICE_TURN_ADDRESS}""