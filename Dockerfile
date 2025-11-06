# Multi-stage build for a tiny static image
FROM golang:1.25-alpine AS build
WORKDIR /src

# Install build tools (optional)
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# Module download
COPY go.mod .
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Source
COPY . .

# Build static binary for linux/amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o /out/upstream ./main.go

# Final minimal image
FROM scratch
ENV BASE_PORT=9000
EXPOSE 9000 9001 9002 9003 9004 9005
COPY --from=build /out/upstream /upstream
ENTRYPOINT ["/upstream"]
