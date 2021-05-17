# COS Kdump Debugger container used in COS kdump

## Overview

This is a docker image used by COS kdump. It includes
scripts and necessary dependencies for inspecting a kernel crash dump from a COS
instance based on COS images.

## Building COS Kdump Debugger Docker image

### Locally (for testing the image)

For testing, you can simply build and test this docker container locally on your
workstation:

```shell
  $ docker build -t cos-kdump-debugger:dev .
```

## Using cos-kdump-debugger Image

This container requires two mount points from the host:

1.  It needs the gcloud credential from your host machine. (Not required if
    using a GCE VM, because GCE VMs get their credential from metadata server.)
2.  It needs a sosreport tarball containing the kdump.

Let's say you have the sosreport tarball located at
`~/debug/sosreport-kdump-next-20190130220549.tar.xz`, then you should run:

```shell
  $ docker run --rm -it \
  $ -v ~/.config/gcloud:/root/.config/gcloud \
  $ -v ~/debug/:/sos \
  $ cos-kdump-debugger:dev \
  $ --sos sosreport-kdump-next-20190130220549.tar.xz
```

If you only want to use the container to run some simply crash commands
(useful for testing), you can run:

```shell
  $ docker run --rm -it \
  $ -v ~/.config/gcloud:/root/.config/gcloud \
  $ -v ~/debug/:/sos \
  $ cos-kdump-debugger:dev \
  $ --crash_command "bt" \
  $ --sos sosreport-kdump-next-20190130220549.tar.xz
```

The `kdump_debugger.sh` script requires the matching vmlinux for the COS kernel
being used. By default, the script will fetch the vmlinux for you by looking in
the GCS bucket `gs://cos-tools`. You can also explicitly set the path to the
matching vmlinux in GCS by setting the `--vmlinux_path` flag. 

For example:

```shell
  $ docker run --rm -it \
  $ -v ~/.config/gcloud:/root/.config/gcloud \
  $ -v ~/debug/:/sos \
  $ cos-kdump-debugger:dev \
  $ --crash_command "bt" \
  $ --sos sosreport-kdump-next-20190130220549.tar.xz
  $ --vmlinux_path gs://<path-to-vmlinux-in-storage-bucket>
```
