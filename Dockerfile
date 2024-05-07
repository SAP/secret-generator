### build go executable
FROM --platform=$BUILDPLATFORM golang:1.22.3 as build
ARG TARGETOS TARGETARCH

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY Makefile Makefile

RUN make envtest \
 && CGO_ENABLED=0 KUBEBUILDER_ASSETS="/workspace/bin/k8s/current" go test ./... \
 && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o ./bin/webhook ./cmd/webhook

### final image
FROM scratch

ENTRYPOINT ["/app/bin/webhook"]

COPY --from=build /workspace/bin/webhook /app/bin/webhook
