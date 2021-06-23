# COS Kernel Devenv container

## Overview

## Building COS Kernel Devenv Image

### Locally (for testing the image)

For testing, you can simply build and test this docker container locally on your
workstation:

```shell
  $ docker build -t gcr.io/cloud-kernel-build/cos-kernel-devenv:dev .
```

### Production (push into GCR)


```shell
  $ VERSION=<version>  # e.g., 20171008
  $ docker build -t gcr.io/cloud-kernel-build/cos-kernel-devenv:$VERSION .
  $ docker build -t gcr.io/cloud-kernel-build/cos-kernel-devenv:latest .
  $ gcloud docker -- push gcr.io/cloud-kernel-build/cos-kernel-devenv:$VERSION
  $ gcloud docker -- push gcr.io/cloud-kernel-build/cos-kernel-devenv:latest
```

## Using COS Kernel Devenv Image

### Download

Get the latest version of the `cos-kernel-devenv` container by running the
following command:

```shell
$ docker pull gcr.io/cloud-kernel-build/cos-kernel-devenv
```

### Overview

`cos-kernel-devenv` provides the development environment for building Linux
kernel or standalone kernel modules. The container supports three operation
modes: automatic kernel build, automatic module build, and an interactive shell.

The development environment includes a cross-compilation toolchain and an
optional kernel headers. Kernel headers only provided if the environment
replicates the environment of the specific COS build.

Kernel sources or module sources are passed to the container as a volume
and automatic modes assume that the working directory is the top-level
build directory.

In the interactive mode the contianer provides `kmake` shell function to
simplify kernel builds. It's a wrapper around `make command with arguments
specific for kernel build: toolchain and arch-specific. It also configures
standard `make` specific shell variables like `CC` or `LD`.

There are three ways to initialize dev environment:

### Modes of Operation
#### Generic Cross-compilation Environment

Unless no arguments specifying COS build are passed to the container,
env is initialized only with a cross-compilation toolchain

```
$ docker run --rm -ti -v $(pwd):/src -w /src gcr.io/cloud-kernel-build/cos-kernel-devenv
... some output ...

root@daf505e41e51:/src# echo $CC
x86_64-cros-linux-gnu-clang

root@daf505e41e51:/src# $CC -v
Chromium OS 12.0_pre422132_p20210405-r9 clang version 13.0.0 (/var/tmp/portage/sys-devel/llvm-12.0_pre422132_p20210405-r9/work/llvm-12.0_pre422132_p20210405/clang cd442157cff4aad209ae532cbf031abbe10bc1df)
Target: x86_64-cros-linux-gnu
Thread model: posix
InstalledDir: /build/x86_64-cros/usr/bin
Found candidate GCC installation: /build/x86_64-cros/usr/bin/../lib/gcc/x86_64-cros-linux-gnu/10.2.0
Selected GCC installation: /build/x86_64-cros/usr/bin/../lib/gcc/x86_64-cros-linux-gnu/10.2.0
Candidate multilib: .;@m64
Selected multilib: .;@m64

root@daf505e41e51:/src# kmake -j48 defconfig && kmake -j48 bzImage
... some output ...
Kernel: arch/x86/boot/bzImage is ready  (#2)
```

By default container sets environment for an x86_64 build but can also target ARM64
builds if passed `-A arm64` command-line argument:

```
$ docker run --rm -ti -v $(pwd):/src -w /src gcr.io/cloud-kernel-build/cos-kernel-devenv -A arm64
...
root@099747a46b80:/src# kmake -j48 defconfig && kmake -j48 Image
...
  OBJCOPY arch/arm64/boot/Image
root@099747a46b80:/src#
```

#### Officially Released COS build

When passed `-R <release>` command-line argument, the container reproduces
the build environment of that specific releases. The generated devenv can
be used both for building a modified kernel for troubleshooting or building
a kernel module that can be used with the specified release. The `<release>`
argument should be in the form of `MAJOR.MINOR.PATCH`, i.e.: `16442.0.0`.

```
$ docker run --rm -ti -v $(pwd):/src -w \
    /src gcr.io/cloud-kernel-build/cos-kernel-devenv -R 16442.0.0
...
root@384c89409064:/src# kmake -C $KHEADERS M=$(pwd) modules
make: Entering directory '/build/x86_64-16442.0.0/kernel-headers/usr/src/linux-headers-5.4.120+'
  CC [M]  /src/samplemodule.o
  Building modules, stage 2.
  MODPOST 1 modules
  CC [M]  /src/samplemodule.mod.o
  LD [M]  /src/samplemodule.ko
make: Leaving directory '/build/x86_64-16442.0.0/kernel-headers/usr/src/linux-headers-5.4.120+'
root@384c89409064:/src#
```
#### COS CI build

It's also possible to reproduce kernel dev environment for the particular CI
build by specifying the voard and the build ID in the container's arguments:
`-b <board> -B <buildid>`. The `<board>` argument is the board name (for
example it's `lakitu` for COS on GCE`). The `<buildid>` is the combination of both
milestone information and the build number, for instance: `R93-16623.0.0`

For this use case it's also neccessary to pass gcloud config information to the
container. It's required to get access to the build artifacts that are stored in
the non-public GCS bucket.

```
$ docker run --rm -ti -v $(pwd):/src -w /src \
    -v ~/.config/gcloud:/root/.config/gcloud \
    gcr.io/cloud-kernel-build/cos-kernel-devenv -b lakitu -B R93-16623.0.0
```
### Automated Builds

In addition to an interactive environment `cos-kernel-devenv` also provides
a batch mode for building a kernel package and modules.

#### Automated Kernel Build

By passing `-k` command-line argument to the container developer can run an
automated kernel build that consists of two steps: `kmake defconfig` and
`kmake tarxz-pkg`. The kernel code is assumed to be in the working directory
specified by `-w` docker argument.

TODO(ovt): implement and document option to specify kernel configs

#### Automated Kernel Build

```
$ docker run --rm -ti -v $(pwd):/src -w \
    /src gcr.io/cloud-kernel-build/cos-kernel-devenv -R 16442.0.0 -k
... skipped ...
'./arch/x86/boot/bzImage' -> './tar-install/boot/vmlinuz-5.10.59'
Tarball successfully created in ./linux-5.10.59-x86.tar.xz
```

#### Automated Modules Build

```
$ docker run --rm -ti -v $(pwd):/src -w \
    /src gcr.io/cloud-kernel-build/cos-kernel-devenv -R 16442.0.0 -m
... skipped ...
  LD [M]  /src/samplemodule.ko
make: Leaving directory '/build/x86_64-16442.0.0/kernel-headers/usr/src/linux-headers-5.4.120+'
```
