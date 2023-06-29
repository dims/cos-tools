FROM ubuntu:22.04

RUN apt-get update && apt-get install -y make git
RUN apt-get install -y apt-transport-https ca-certificates gnupg curl sudo python3
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -
RUN apt-get update && apt-get install -y google-cloud-cli
COPY ./run_config.sh /run_config.sh

ENTRYPOINT ["/run_config.sh"]
