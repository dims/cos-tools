FROM ubuntu:21.10

RUN apt-get -y update && \
	apt-get -y install apt-transport-https ca-certificates gnupg curl && \
	echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
	    > /etc/apt/sources.list.d/google-cloud-sdk.list && \
	curl -o /usr/share/keyrings/cloud.google.gpg \
	    https://packages.cloud.google.com/apt/doc/apt-key.gpg && \
	apt-get update -y && \
	DEBIAN_FRONTEND=noninteractive TZ=Etc/UTC apt-get install -y tzdata && \
	apt-get install -y make python3 git libssl-dev bc bison flex cpio kmod \
	    dwarves google-cloud-cli xz-utils libelf-dev && \
	apt-get clean

COPY ./devenv.sh /devenv.sh

ENTRYPOINT ["/devenv.sh"]
