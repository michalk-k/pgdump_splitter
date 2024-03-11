#!/bin/bash
DOCKER_NAME="registry.betsys.com/postgres/pgdump_splitter"
APP_VERSION="$(cat ./VERSION)"

docker buildx build --push --no-cache  --build-arg="APP_VERSION=$APP_VERSION" --build-arg="GO_VERSION=1.21" --platform=linux/amd64,linux/arm64,darwin/arm64  --tag=$DOCKER_NAME:$APP_VERSION --tag=$DOCKER_NAME:latest .
