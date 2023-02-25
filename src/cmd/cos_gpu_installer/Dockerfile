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
    apt-get install -y xz-utils kmod git make bc curl ccache gpg \
    libc6-dev pciutils gcc libelf-dev libssl-dev bison flex keyutils python3-minimal zlib1g-dev && \
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg \
    --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg && \
    curl -s -L https://nvidia.github.io/libnvidia-container/debian11/libnvidia-container.list \
    | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
    | tee /etc/apt/sources.list.d/nvidia-container-toolkit.list && apt-get update \
    && apt-get install -y nvidia-container-toolkit

ENTRYPOINT ["/cos-gpu-installer"]
CMD ["install"]
