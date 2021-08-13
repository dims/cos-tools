# Dockerfile for the golang 1.16 container
FROM golang:1.16

# Retrieve tools directory
COPY . /work/

# Installing dependencies
RUN apt-get update && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends -y \
    ca-certificates \
    sysstat=12.0.3-2 \
    procps \
    stress-ng
