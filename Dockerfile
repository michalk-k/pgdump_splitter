# syntax=docker/dockerfile:1
ARG GO_VERSION=1.21
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS base

WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

FROM base AS build-client

ARG APP_VERSION="v0.0.0"
ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X main.version=$APP_VERSION" -o /bin/client

FROM scratch as client
COPY --from=build-client /bin/client /bin
ENTRYPOINT [ "/bin" ]

