# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src/go-achat-node

# Cache dependencies first.
COPY go.mod go.sum ./
COPY app/achat/go.mod ./app/achat/go.mod

WORKDIR /src/go-achat-node/app/achat
ARG GOPROXY
ENV GOPROXY=${GOPROXY}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy full source and build.
WORKDIR /src/go-achat-node
COPY . .

WORKDIR /src/go-achat-node/app/achat
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w" -o /out/achat ./cmd/achat


FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/achat /usr/local/bin/achat

EXPOSE 24000/tcp
EXPOSE 9990/tcp

ENTRYPOINT ["/usr/local/bin/achat"]
