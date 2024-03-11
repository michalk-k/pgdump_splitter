# syntax=docker/dockerfile:1
ARG GO_VERSION=1.21
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS base

WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

FROM base AS build-executable

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X main.version=$APP_VERSION" -o /bin/pgdump_splitter

FROM scratch as client
COPY --from=build-executable /bin/pgdump_splitter /pgdump_splitter
ENTRYPOINT [ "/pgdump_splitter" ]

