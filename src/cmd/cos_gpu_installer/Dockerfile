FROM golang:1.16 as cos-gpu-installer-go-builder
COPY . /work/
WORKDIR /work/src/cmd/cos_gpu_installer
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o cos-gpu-installer main.go

# Dockerfile for the COS GPU Installer container.
FROM debian:bookworm
LABEL maintainer="cos-containers@google.com"

COPY --from=cos-gpu-installer-go-builder /work/src/cmd/cos_gpu_installer/cos-gpu-installer /cos-gpu-installer

# Install minimal tools needed to build kernel modules.
RUN apt-get update -qq && \
    apt-get install -y xz-utils kmod git make bc curl ccache \
    libc6-dev pciutils gcc libelf-dev libssl-dev bison flex keyutils python2.7-minimal zlib1g-dev && \
    rm -rf /var/lib/apt/lists/* && \
    # x86_64-cros-linux-gnu-clang depends on /usr/bin/python2
    ln -s /usr/bin/python2.7 /usr/bin/python2

ENTRYPOINT ["/cos-gpu-installer"]
CMD ["install"]
