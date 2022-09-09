FROM golang:1.16 as cosovaconverter

COPY . /work/
WORKDIR /work/src/cmd/cos_ova_converter
RUN go build -o  cos_ova_converter .

FROM gcr.io/compute-image-tools/daisy as daisyworkflow

FROM debian:buster-slim

RUN apt-get update && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends -y -qq \
    ca-certificates \
    apt-transport-https \
    gnupg \
    curl \
    qemu-utils \
    python3 \
    python3-pip python3-setuptools\
    xmlstarlet \
    git

COPY /src/cmd/cos_ova_converter/requirements.txt /work/src/cmd/cos_ova_converter/requirements.txt
RUN pip3 install --require-hashes -r /work/src/cmd/cos_ova_converter/requirements.txt

RUN git clone https://cos.googlesource.com/third_party/platform/crosutils.git
RUN cd crosutils && git checkout 74d0afda96dc8c58863f76b2e144c373f92451f6

COPY --from=cosovaconverter /work/src/cmd/cos_ova_converter/cos_ova_converter /cos_ova_converter
COPY --from=daisyworkflow  /daisy /daisy
COPY --from=daisyworkflow /workflows /workflows

ENTRYPOINT ["/cos_ova_converter"]
