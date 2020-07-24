# Dockerfile for the COS GPU Installer container.

FROM debian:9
LABEL maintainer="cos-containers@google.com"

# Install minimal tools needed to build kernel modules.
RUN apt-get update -qq && \
    apt-get install -y xz-utils kmod git make bc curl ccache \
    libc6-dev pciutils gcc libelf-dev libssl-dev bison flex keyutils python2.7-minimal && \
    rm -rf /var/lib/apt/lists/* && \
    # x86_64-cros-linux-gnu-clang depends on /usr/bin/python2
    ln -s /usr/bin/python2.7 /usr/bin/python2

COPY cos-gpu-installer /cos-gpu-installer
ENTRYPOINT ["/cos-gpu-installer"]
