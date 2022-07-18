FROM golang:1.18 as go-builder
COPY . /work/
WORKDIR /work/src/cmd/cos_gpu_config_builder
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -o cos-gpu-config-builder main.go
FROM gcr.io/distroless/static-debian11
LABEL maintainer="cos-containers@google.com"
COPY --from=go-builder /work/src/cmd/cos_gpu_config_builder/cos-gpu-config-builder /cos-gpu-config-builder
ENTRYPOINT ["/cos-gpu-config-builder"]
