# Start from google/cloud-sdk:slim base.
FROM google/cloud-sdk:slim

RUN echo 'deb http://deb.debian.org/debian buster-backports main' > /etc/apt/sources.list.d/buster-backports.list
RUN apt-get update && apt-get install -y make gcc git libssl-dev bc \
          libelf-dev/buster-backports bison flex cpio dwarves/buster-backports kmod

COPY ./devenv.sh /devenv.sh

ENTRYPOINT ["/devenv.sh"]
