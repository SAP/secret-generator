# Build the webhook binary
FROM --platform=$BUILDPLATFORM golang:1.25.0 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the go module manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go sources
COPY cmd/ cmd/
COPY internal/ internal/
COPY Makefile Makefile

# Run tests and build
RUN make envtest \
 && CGO_ENABLED=0 KUBEBUILDER_ASSETS="/workspace/bin/k8s/current" go test ./... \
 && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o webhook ./cmd/webhook

# Use distroless as minimal base image to package the webhook binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/webhook .
USER 65532:65532

ENTRYPOINT ["/webhook"]