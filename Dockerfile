# Multi-stage build for a tiny static image
FROM golang:1.26-alpine AS build
WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

# Install build tools (optional)
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# Module download
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Source
COPY . .

# Build a static binary matching the current buildx target platform.
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags "-s -w" -o /out/upstream ./main.go

# Final minimal image
FROM scratch
ENV BASE_PORT=9000
EXPOSE 9000 9001 9002 9003 9004 9005
COPY --from=build /out/upstream /upstream
ENTRYPOINT ["/upstream"]
