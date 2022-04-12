# Start from debian
FROM debian:bookworm

RUN apt-get update

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get install -y -qq --no-install-recommends crash gdb xz-utils curl vim \
      liblzo2-2 libsnappy1v5 zlib1g libncurses5 binutils apt-transport-https ca-certificates gnupg

# Installs gsutil, following:
# https://cloud.google.com/storage/docs/gsutil_install#sdk-install
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] \
    https://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | \
    apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - \
    && apt-get update && apt-get -y -qq install google-cloud-sdk=369.0.0-0 && apt-get clean

COPY ./kdump_debugger.sh /kdump_debugger.sh

ENTRYPOINT ["/kdump_debugger.sh"]
