ARG DEBUG=false
# Build the submariner-operator binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers controllers/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on && \
    [ "$DEBUG" = "false" ] && \
    go build -o submariner-operator -a -ldflags -s -w main.go || \
    go build -o submariner-operator -a main.go


# Use distroless as minimal base image to package the submariner-operator binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /
COPY --from=builder /workspace/submariner-operator .
USER nonroot:nonroot

ENTRYPOINT ["/submariner-operator"]
