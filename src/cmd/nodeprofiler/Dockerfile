# Dockerfile for the Golang 1.16 container.

FROM golang:1.16 as nodeprofiler

# Retrieving application dependencies.
COPY . /work/

# Changing to the app directory.
WORKDIR /work/src/cmd/nodeprofiler
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o nodeprofiler main.go

# Dockerfile for the Nodeprofiler container using Debian slim image for a lean
# production container.

FROM debian:buster-slim

# Installing certificates containers need to log data to Google Cloud Logging
# as well as all the dependencies needed to run the profiler tool.
RUN apt-get update && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends -y \
    ca-certificates \
    sysstat=12.0.3-2 \
    procps && \
    rm -rf /var/lib/apt/lists/*

# Copying the nodeprofiler binary to the production image.
COPY --from=nodeprofiler /work/src/cmd/nodeprofiler/nodeprofiler /nodeprofiler

# Entrypoint sets the profiler image's main command.
ENTRYPOINT ["/nodeprofiler"]

# CMD sets the profiler image's default flags. If the user does not specify flags,
# containers shall display supported functionalities.
CMD ["--help"]
