# Step 1: build all bundled programs built with Bazel
FROM gcr.io/cloud-builders/bazel AS bazel_builder
ARG _BUILD_TOOLS_CACHE

ADD . /workspace
WORKDIR /workspace
RUN bazel build \
  --remote_cache=$_BUILD_TOOLS_CACHE \
  //src/cmd/cos_customizer:cos_customizer \
  //src/cmd/provisioner:provisioner_amd64 \
  //src/cmd/provisioner:provisioner_arm64 \
  //src/cmd/metadata_watcher:metadata_watcher_amd64 \
  //src/cmd/metadata_watcher:metadata_watcher_arm64 \
  //src/cmd/handle_disk_layout:handle_disk_layout_bin_amd64 \
  //src/cmd/handle_disk_layout:handle_disk_layout_bin_arm64 \
  @com_github_googlecloudplatform_docker_credential_gcr//:docker-credential-gcr_amd64 \
  @com_github_googlecloudplatform_docker_credential_gcr//:docker-credential-gcr_arm64; \
  mkdir -p _out/amd64; \
  cp \
    $(bazel cquery --output=files //src/cmd/cos_customizer:cos_customizer) \
    $(bazel cquery --output=files //src/cmd/provisioner:provisioner_amd64) \
    $(bazel cquery --output=files //src/cmd/metadata_watcher:metadata_watcher_amd64) \
    $(bazel cquery --output=files //src/cmd/handle_disk_layout:handle_disk_layout_bin_amd64) \
    $(bazel cquery --output=files @com_github_googlecloudplatform_docker_credential_gcr//:docker-credential-gcr_amd64) \
    _out/amd64; \
  mkdir -p _out/arm64; \
  cp \
    $(bazel cquery --output=files //src/cmd/provisioner:provisioner_arm64) \
    $(bazel cquery --output=files //src/cmd/metadata_watcher:metadata_watcher_arm64) \
    $(bazel cquery --output=files //src/cmd/handle_disk_layout:handle_disk_layout_bin_arm64) \
    $(bazel cquery --output=files @com_github_googlecloudplatform_docker_credential_gcr//:docker-credential-gcr_arm64) \
    _out/arm64

# Step 2: build CIDATA and SCRATCH images
FROM debian:bookworm AS cidata_builder

RUN dpkg --add-architecture arm64; apt-get update; \
  apt-get install -y --no-install-recommends dosfstools mtools

# Output in /opt/veritysetup_amd64/veritysetup_amd64.tar.gz
RUN mkdir -p /opt/veritysetup_amd64/{root,debs}
WORKDIR /opt/veritysetup_amd64/debs
RUN apt-get download \
  coreutils:amd64 \
  tar:amd64 \
  libacl1:amd64 \
  libattr1:amd64 \
  libc6:amd64 \
  libselinux1:amd64 \
  libpcre3:amd64 \
  cryptsetup-bin:amd64 \
  libblkid1:amd64 \
  libcryptsetup12:amd64 \
  libpopt0:amd64 \
  libuuid1:amd64 \
  libdevmapper1.02.1:amd64 \
  libgcrypt20:amd64 \
  libargon2-1:amd64 \
  libjson-c5:amd64 \
  libudev1:amd64 \
  libssl3:amd64 \
  libpcre2-8-0:amd64
WORKDIR /opt/veritysetup_amd64
RUN for f in debs/*.deb; do dpkg-deb --extract $f root; done; \
  tar czf veritysetup_amd64.tar.gz -C root .

# Output in /opt/veritysetup_arm64/veritysetup_arm64.tar.gz
RUN mkdir -p /opt/veritysetup_arm64/{root,debs}
WORKDIR /opt/veritysetup_arm64/debs
RUN apt-get download \
  coreutils:arm64 \
  tar:arm64 \
  libacl1:arm64 \
  libattr1:arm64 \
  libc6:arm64 \
  libselinux1:arm64 \
  libpcre3:arm64 \
  cryptsetup-bin:arm64 \
  libblkid1:arm64 \
  libcryptsetup12:arm64 \
  libpopt0:arm64 \
  libuuid1:arm64 \
  libdevmapper1.02.1:arm64 \
  libgcrypt20:arm64 \
  libargon2-1:arm64 \
  libjson-c5:arm64 \
  libudev1:arm64 \
  libssl3:arm64 \
  libpcre2-8-0:arm64
WORKDIR /opt/veritysetup_arm64
RUN for f in debs/*.deb; do dpkg-deb --extract $f root; done; \
  tar czf veritysetup_arm64.tar.gz -C root .

# Output in /opt/cidata/cidata.img
RUN mkdir -p /opt/cidata/deps
WORKDIR /opt/cidata
COPY --from=bazel_builder /workspace/src/data/startup.yaml deps/user-data
COPY --from=bazel_builder /workspace/_out/amd64 deps/amd64
COPY --from=bazel_builder /workspace/_out/arm64 deps/arm64
RUN mkfs.fat -n CIDATA -S 512 -s 8 -C cidata.img 131072; \
  touch meta-data; \
  mcopy -i cidata.img deps/user-data ::/user-data; \
  mcopy -i cidata.img meta-data ::/meta-data; \
  mcopy -i cidata.img deps/amd64/provisioner_amd64 ::/provisioner_amd64; \
  mcopy -i cidata.img deps/arm64/provisioner_arm64 ::/provisioner_arm64; \
  mcopy -i cidata.img deps/amd64/metadata_watcher_amd64 ::/metadata_watcher_amd64; \
  mcopy -i cidata.img deps/arm64/metadata_watcher_arm64 ::/metadata_watcher_arm64; \
  mcopy -i cidata.img deps/amd64/docker-credential-gcr_amd64 ::/docker-credential-gcr_amd64; \
  mcopy -i cidata.img deps/arm64/docker-credential-gcr_arm64 ::/docker-credential-gcr_arm64; \
  mcopy -i cidata.img deps/amd64/handle_disk_layout_bin_amd64 ::/handle_disk_layout_bin_amd64; \
  mcopy -i cidata.img deps/arm64/handle_disk_layout_bin_arm64 ::/handle_disk_layout_bin_arm64; \
  mcopy -i cidata.img /opt/veritysetup_amd64/veritysetup_amd64.tar.gz ::/veritysetup_amd64.tar.gz; \
  mcopy -i cidata.img /opt/veritysetup_arm64/veritysetup_arm64.tar.gz ::/veritysetup_arm64.tar.gz

# Output in /opt/scratch/scratch.img
RUN mkdir -p /opt/scratch
WORKDIR /opt/scratch
RUN fallocate -l 512M scratch.img; \
  mkfs.ext4 scratch.img; \
  tune2fs -c0 -i0 scratch.img; \
  e2label scratch.img SCRATCH

# Step 3: build the cos-customizer container image
FROM gcr.io/compute-image-tools/daisy@sha256:a23774074d5941ed9e25f64ee7e02f96d2f8e09a4d7cee7131b49664267c33c7 AS daisy
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
  tar \
  mtools \
  ca-certificates
RUN mkdir -p /tmp; mkdir -p /workspace
COPY --from=bazel_builder /workspace/src/data /data
COPY --from=bazel_builder /workspace/_out/amd64/cos_customizer /cos_customizer
COPY --from=daisy /daisy /daisy
COPY --from=cidata_builder /opt/cidata/cidata.img /cidata.img
COPY --from=cidata_builder /opt/scratch/scratch.img /scratch.img
ENTRYPOINT ["/cos_customizer"]
