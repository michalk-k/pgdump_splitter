#!/bin/bash
APP_VERSION="0.9.0"
DOCKER_NAME="registry.betsys.com/postgres/pgdump_splitter"


# docker build --output=bin --target=binaries --build-arg="APP_VERSION=v0.0.1" --build-arg="GO_VERSION=1.21" --platform=darwin/arm64,linux/amd64,linux/arm64 --tag=$DOCKER_NAME:$APP_VERSION .

docker buildx build --push --no-cache --build-arg="APP_VERSION=$APP_VERSION" --build-arg="GO_VERSION=1.21" --platform "linux/amd64,darwin/arm64" --tag=$DOCKER_NAME:$APP_VERSION .

