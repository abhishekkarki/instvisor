FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN make build-all

# Runtime image
FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite-libs

# Create user
RUN addgroup -g 1000 instvisor && \
    adduser -D -u 1000 -G instvisor instvisor

# Create directories
RUN mkdir -p /var/lib/instvisor /etc/instvisor && \
    chown -R instvisor:instvisor /var/lib/instvisor /etc/instvisor

# Copy binaries from builder
COPY --from=builder /build/build/instvisor-agent /usr/local/bin/
COPY --from=builder /build/build/instvisor-analyze /usr/local/bin/
COPY --from=builder /build/configs/agent.yaml /etc/instvisor/

# Switch to non-root user
USER instvisor

VOLUME ["/var/lib/instvisor"]

EXPOSE 9090

ENTRYPOINT ["/usr/local/bin/instvisor-agent"]
CMD ["-config", "/etc/instvisor/agent.yaml"]