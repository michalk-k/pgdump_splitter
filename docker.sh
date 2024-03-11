#!/bin/bash
APP_VERSION="0.9.0"
DOCKER_NAME="registry.betsys.com/postgres/pgdump_splitter"

docker build -ldflags="-X 'main.version=$(cat VERSION)'" --build-arg="GO_VERSION=1.21" --platform=linux/amd64,linux/arm64,darwin/arm64 --tag=$DOCKER_NAME:$APP_VERSION .

docker push $DOCKER_NAME:$APP_VERSION

