FROM debian:bullseye-backports

# Prepare the image.
ENV DEBIAN_FRONTEND noninteractive

# Google Cloud SDK pre requisites.
RUN apt-get update && apt-get install -y -qq --no-install-recommends apt-transport-https \
    ca-certificates gnupg curl

ARG TARGETARCH

RUN cd $(mktemp -d) && curl -L -O "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v2.1.5/docker-credential-gcr_linux_$TARGETARCH-2.1.5.tar.gz" && \
    tar xvf "docker-credential-gcr_linux_$TARGETARCH-2.1.5.tar.gz" && \
    mv docker-credential-gcr /usr/bin/ && rm -r $(pwd)

# Install the Google Cloud SDK.
ENV HOME /
ENV CLOUDSDK_PYTHON_SITEPACKAGES 1
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] \
    https://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | \
    apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - \
    && apt-get update && apt-get -y -qq install google-cloud-sdk=369.0.0-0 && apt-get clean

# Various networking and other tools. net-tools installs arp, netstat, etc.
RUN apt-get install -u -qq vim \
    net-tools netcat ipset conntrack inetutils-traceroute bridge-utils \
    ebtables iputils-ping \
    && apt-get clean

# These packages are required or extracting source tarballs and building the kernel.
RUN apt-get update && \
    apt-get install -u -qq \
        xz-utils make gcc python3-minimal bc libelf-dev libssl-dev \
        crash bison flex dwarves libdw1 && \
    apt-get clean
COPY cos-kernel /usr/local/bin

VOLUME ["/.config"]

CMD ["/bin/bash"]
