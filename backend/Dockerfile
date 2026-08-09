# syntax=docker/dockerfile:1

FROM golang:1.26.2-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build \
    -tags timetzdata \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w" \
    -o /out/daftar-api \
    ./cmd/api

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/daftar-api /daftar-api

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/daftar-api"]
