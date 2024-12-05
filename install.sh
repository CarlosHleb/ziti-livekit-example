#!/bin/bash

source .env

function dockercomp {
  docker compose "$@";
}

function zitiEx {
  dockercomp \
    exec -it ziti-edge-router \
    /var/openziti/ziti-bin/ziti "$@";
}

# Login
zitiEx edge login ${ZITI_CTRL_EDGE_ADVERTISED_ADDRESS}:$ZITI_CTRL_EDGE_ADVERTISED_PORT -u $ZITI_USER -p $ZITI_PWD --yes

# Create zac service
zitiEx edge create config ${ZITI_SERVICE_ZAC}.host.config host.v1 '{"protocol":"tcp", "address":"'"ziti-console"'", "port":'${ZITI_CONSOLE_PORT}'}'
zitiEx edge create config ${ZITI_SERVICE_ZAC}.int.config  intercept.v1 '{"protocols":["tcp"],"addresses":["'${ZITI_SERVICE_ZAC}'"], "portRanges":[{"low":'${ZITI_CONSOLE_PORT}', "high":'${ZITI_CONSOLE_PORT}'}]}'
zitiEx edge create service ${ZITI_SERVICE_ZAC} --configs ${ZITI_SERVICE_ZAC}".host.config",${ZITI_SERVICE_ZAC}".int.config"
zitiEx edge create service-policy ${ZITI_SERVICE_ZAC}".bind" Bind --service-roles "@"${ZITI_SERVICE_ZAC}"" --identity-roles "#"${ZITI_SERVICE_ZAC}".bind"
zitiEx edge create service-policy ${ZITI_SERVICE_ZAC}".dial" Dial --service-roles "@"${ZITI_SERVICE_ZAC}"" --identity-roles "#"${ZITI_SERVICE_ZAC}".dial"

# Create livekit api rtc service
zitiEx edge create config ${ZITI_SERVICE_LIVEKIT_RTC}.host.config host.v1 '{
    "forwardProtocol": true,
    "forwardPort": true,
    "address": "livekit-server",
    "allowedPortRanges": [
      {
        "low": 80,
        "high": 80
      },
      {
        "low": 3478,
        "high": 3478
      },
      {
        "low": 50000,
        "high": 60000
      },
      {
        "low": 30000,
        "high": 40000
      }
    ],
    "allowedProtocols": [
      "tcp",
      "udp"
    ]
  }'
zitiEx edge create config ${ZITI_SERVICE_LIVEKIT_RTC}.int.config intercept.v1 '{
  "protocols":["tcp", "udp"],
  "addresses":["'${ZITI_SERVICE_LIVEKIT_RTC_ADDRESS}'", "'${ZITI_SERVICE_LIVEKIT_RTC}'"], 
  "portRanges":[{"low":80, "high":80}, {"low":3478, "high":3478}, {"low": 50000, "high": 60000}, {"low": 30000, "high": 40000}]
}'
zitiEx edge create service ${ZITI_SERVICE_LIVEKIT_RTC} --configs ${ZITI_SERVICE_LIVEKIT_RTC}".host.config",${ZITI_SERVICE_LIVEKIT_RTC}".int.config"
zitiEx edge create service-policy ${ZITI_SERVICE_LIVEKIT_RTC}".bind" Bind --service-roles "@"${ZITI_SERVICE_LIVEKIT_RTC} --identity-roles "#"${ZITI_SERVICE_LIVEKIT_RTC}".bind"
zitiEx edge create service-policy ${ZITI_SERVICE_LIVEKIT_RTC}".dial" Dial --service-roles "@"${ZITI_SERVICE_LIVEKIT_RTC} --identity-roles "#"${ZITI_SERVICE_LIVEKIT_RTC}".dial"

# Create livekit api service
zitiEx edge create config ${ZITI_SERVICE_LIVEKIT}.host.config host.v1 '{
    "address": "livekit-nginx-server",
    "port": 7880,
    "protocol": "tcp"
}'
zitiEx edge create config ${ZITI_SERVICE_LIVEKIT}.int.config intercept.v1 '{
  "protocols":["tcp"],
  "addresses":["'${ZITI_SERVICE_LIVEKIT_ADDRESS}'", "'${ZITI_SERVICE_LIVEKIT}'"],
  "portRanges":[{"low":7880, "high":7880}]
}'
zitiEx edge create service ${ZITI_SERVICE_LIVEKIT} --configs ${ZITI_SERVICE_LIVEKIT}".host.config",${ZITI_SERVICE_LIVEKIT}".int.config"
zitiEx edge create service-policy ${ZITI_SERVICE_LIVEKIT}".bind" Bind --service-roles "@"${ZITI_SERVICE_LIVEKIT} --identity-roles "#"${ZITI_SERVICE_LIVEKIT}".bind"
zitiEx edge create service-policy ${ZITI_SERVICE_LIVEKIT}".dial" Dial --service-roles "@"${ZITI_SERVICE_LIVEKIT} --identity-roles "#"${ZITI_SERVICE_LIVEKIT}".dial"

# Create turn service
zitiEx edge create config ${ZITI_SERVICE_TURN}.host.config host.v1 '{
  "forwardProtocol": true,
  "forwardPort": true,
  "address": "livekit-server",
  "allowedPortRanges": [
    {
      "low": 3478,
      "high": 3478
    },
    {
      "low": 5349,
      "high": 5349
    },
    {
      "low": 30000,
      "high": 40000
    }
  ],
  "allowedProtocols": [
    "tcp",
    "udp"
  ]
}'
zitiEx edge create config ${ZITI_SERVICE_TURN}.int.config  intercept.v1 '{
  "protocols":["tcp", "udp"],
  "addresses":["'${ZITI_SERVICE_TURN_ADDRESS}'", "'${ZITI_SERVICE_TURN}'"],
  "portRanges":[{"low":3478, "high":3478}, {"low":5349, "high":5349}, {"low": 30000, "high": 40000}]
}'
zitiEx edge create service ${ZITI_SERVICE_TURN} --configs ${ZITI_SERVICE_TURN}".host.config",${ZITI_SERVICE_TURN}".int.config"
zitiEx edge create service-policy ${ZITI_SERVICE_TURN}".bind" Bind --service-roles "@"${ZITI_SERVICE_TURN} --identity-roles "#"${ZITI_SERVICE_TURN}".bind"
zitiEx edge create service-policy ${ZITI_SERVICE_TURN}".dial" Dial --service-roles "@"${ZITI_SERVICE_TURN} --identity-roles "#"${ZITI_SERVICE_TURN}".dial"

# Update edge router
zitiEx edge update identity ${ZITI_ROUTER_NAME} \
  -a ${ZITI_SERVICE_ZAC}.bind \
  -a ${ZITI_SERVICE_LIVEKIT}.bind -a ${ZITI_SERVICE_TURN}.bind \
  -a ${ZITI_SERVICE_LIVEKIT_RTC}.bind

# Add ziti-edge-controller-root-ca to host trusted crt's
dockercomp cp ziti-controller:/persistent/pki/ziti-edge-controller-root-ca/certs/ziti-edge-controller-root-ca.cert \
  ./store/ziti-edge-controller-root-ca.crt
sudo cp ./store/ziti-edge-controller-root-ca.crt /usr/local/share/ca-certificates
sudo cp ./store/livekit.crt /usr/local/share/ca-certificates
sudo cp ./store/turn.crt /usr/local/share/ca-certificates
sudo update-ca-certificates -f

# Create admin1 in case need a test identity
zitiEx edge create identity "admin1" -a ${ZITI_SERVICE_ZAC}.dial \
  -a ${ZITI_SERVICE_LIVEKIT}.dial -a ${ZITI_SERVICE_TURN}.dial \
  -a ${ZITI_SERVICE_LIVEKIT_RTC}.dial -o /persistent/admin1.jwt --admin

zitiEx edge enroll /persistent/admin1.jwt -o /persistent/admin1.json
# dockercomp cp ziti-edge-router:/persistent/admin1.json /opt/openziti/etc/identities
# sudo systemctl restart ziti-edge-tunnel.service

# Create publisher identity
zitiEx edge create identity "publisher" \
  -a ${ZITI_SERVICE_LIVEKIT}.dial -a ${ZITI_SERVICE_TURN}.dial \
  -a ${ZITI_SERVICE_LIVEKIT_RTC}.dial -o /persistent/publisher.jwt --admin

zitiEx edge enroll /persistent/publisher.jwt -o /persistent/publisher.json
dockercomp cp ziti-edge-router:/persistent/publisher.json ./store/publisher.json
dockercomp cp ./store/publisher.json publisher-app:/work/publisher/publisher.json

dockercomp cp store/livekit.crt \
  publisher-app:/usr/local/share/ca-certificates
dockercomp exec publisher-app /usr/sbin/update-ca-certificates

# Create subscriber identity
zitiEx edge create identity "subscriber" \
  -a ${ZITI_SERVICE_LIVEKIT}.dial -a ${ZITI_SERVICE_TURN}.dial \
  -a ${ZITI_SERVICE_LIVEKIT_RTC}.dial -o /persistent/subscriber.jwt --admin

zitiEx edge enroll /persistent/subscriber.jwt -o /persistent/subscriber.json
dockercomp cp ziti-edge-router:/persistent/subscriber.json ./store/subscriber.json
dockercomp cp ./store/subscriber.json subscriber-app:/work/subscriber/subscriber.json

dockercomp cp store/livekit.crt \
  subscriber-app:/usr/local/share/ca-certificates
dockercomp exec subscriber-app /usr/sbin/update-ca-certificates

