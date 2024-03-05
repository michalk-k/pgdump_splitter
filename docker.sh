#!/bin/bash
APP_VERSION="0.9.1"
DOCKER_NAME="registry.betsys.com/postgres/pgdump_splitter"

docker buildx build --push --no-cache --build-arg="APP_VERSION=$APP_VERSION" --build-arg="GO_VERSION=1.21" --platform "linux/amd64,darwin/arm64" --tag=$DOCKER_NAME:$APP_VERSION --tag=$DOCKER_NAME:latest .

