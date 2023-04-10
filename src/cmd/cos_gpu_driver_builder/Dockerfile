FROM golang:1.18 as go-builder
COPY . /work/
WORKDIR /work/src/cmd/cos_gpu_driver_builder
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -o cos-gpu-driver-builder main.go
FROM ubuntu:latest
LABEL maintainer="cos-containers@google.com"
COPY --from=go-builder /work/src/cmd/cos_gpu_driver_builder/cos-gpu-driver-builder /cos-gpu-driver-builder
RUN apt-get update && apt-get -y install ca-certificates xz-utils kmod make libc-dev libc6-dev libelf-dev binutils
ENTRYPOINT ["/cos-gpu-driver-builder"]
