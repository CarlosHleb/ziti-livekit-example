#!/bin/bash

docker compose down -v
rm -rf store/*
rm publisher/publisher.json
rm subscriber/subscriber.json