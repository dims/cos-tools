# Start from google/cloud-sdk:slim base.
FROM google/cloud-sdk:slim

RUN echo 'deb http://deb.debian.org/debian buster-backports main' > /etc/apt/sources.list.d/buster-backports.list
RUN apt-get update && apt-get install -y make gcc git libssl-dev bc bison flex cpio kmod cmake \
	  libdw1/buster-backports libdw-dev/buster-backports  libelf-dev/buster-backports && \
	  git clone --depth 1 -b v1.17 https://git.kernel.org/pub/scm/devel/pahole/pahole.git /tmp/pahole && \
	  cd /tmp/pahole && \
	  cmake -D__LIB=lib -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE="None" -DLIBBPF_EMBEDDED=off . && \
	  make -C /tmp/pahole install && cd - && rm -rf /tmp/pahole  && \
	  apt-get clean

COPY ./devenv.sh /devenv.sh

ENTRYPOINT ["/devenv.sh"]
