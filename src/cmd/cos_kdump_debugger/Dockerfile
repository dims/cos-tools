# Start from ubuntu
FROM ubuntu:latest

RUN apt-get update

ARG DEBIAN_FRONTEND=noninteractive
# This is required because of https://github.com/crash-utility/crash/issues/74
RUN apt update && BUILD_DEPS="binutils binutils-dev bison gawk \
    flex zlib1g-dev libncurses5-dev liblzo2-dev libsnappy-dev \
    make wget git" \
    && apt-get install -y --ignore-missing $BUILD_DEPS \
    && git clone -b 7.2.9 https://github.com/crash-utility/crash.git /tmp/crash \
    && git config --global user.email "foo@bar.local" \
    && git config --global user.name "Foo Bar" \
    && git -C /tmp/crash cherry-pick a5531b2 71e159c \
    && make -C /tmp/crash extensions lzo snappy \
    && mkdir -p /usr/lib/crash/extensions/ \
    && cp -f /tmp/crash/crash /usr/bin \
    && cp -f /tmp/crash/extensions/*.so /usr/lib/crash/extensions/ \
    && strip /usr/bin/crash /usr/lib/crash/extensions/*.so \
    && rm -rf /tmp/crash \
    && apt-get purge -y --auto-remove $BUILD_DEPS
RUN apt-get install -y -qq --no-install-recommends gdb xz-utils python curl vim \
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
