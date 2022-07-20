FROM ubuntu:22.04

RUN set -x; apt-get -y update && \
	apt-get -y install apt-transport-https ca-certificates && \
	DEBIAN_FRONTEND=noninteractive TZ=Etc/UTC apt-get install -y tzdata && \
	apt-get install -y make python3 git libssl-dev bc bison flex cpio kmod \
	    dwarves xz-utils libelf-dev rsync clang llvm g++ libmnl-dev \
	    libmount-dev libcap-ng-dev libcap-dev libelf-dev \
	    libfuse-dev libpopt-dev libnuma-dev docutils-common && \
	if [ "$(uname -m)" = "x86_64" ]; then apt-get -y install gcc-multilib libc6-i386 libc6-dev-i386; fi && \
	apt-get clean
