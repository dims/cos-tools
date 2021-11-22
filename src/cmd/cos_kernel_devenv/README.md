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
  $ VERSION=<version>  # e.g., v20211031
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
kernel or standalone kernel modules. The container supports four operational
modes:
  - wrapper for make command with kernel-specific arguments and env variables
  - automatic kernel build
  - automatic module out-of-tree module build
  - interactive shell

The development environment includes a cross-compilation toolchain and
optional kernel headers. Kernel headers only provided if the environment
replicates the environment of an official COS release (specified
using `-R` command-line switch) or of an CI build (specified by
using `-b` and `-B` command-line switches).

### Preparations

Kernel sources or module sources are passed to the container as a volume
(directory mapped from the host machine into a container). Non-interactive
operational modes assume that the working directory is the top-level build
directory.

`cos-kernel-devenv` image does not contain any actual toolchain and populates
binaries and kernel headers from network sources to `/build` location in the
container. In order to save time on every invocation users can map a host
directory to `/build` volume to make installed files persistent between runs.

Some of the network sources require access to GCS buckets so to pass gcloud
config and credentials from the host into a container `~/.config/gcloud`
host directory needs to be mapped into `/root/.config/gcloud` volume.

To hide all this complexity you can use the code snippet below that wraps it all
up in a convenient shell function.

```shell
# cache directory for toolchains
mkdir -p ~/cos-build

function cos_kmake() { 
    docker run --rm -ti \
      -v ~/.config/gcloud:/root/.config/gcloud \
      -v ~/cos-build:/build -v $(pwd):/src -w /src \
      gcr.io/cloud-kernel-build/cos-kernel-devenv \
      "$@"
}
export -f cos_kmake
```

At this point you should be able to run `cos_kmake -h` and get a list of
available command-line options:

```
Usage: /devenv.sh [-k | -m | -i] [-Hcd] [-A <x86|arm64>]
    [-C <kernelconfig>] [-O  <objdir>]
    [-B <build> -b <board> | -R <release> | -G <bucket>]
    [-t <toolchain_version>] [VAR=value ...] [target ...]

Options:
  -A <arch>     target architecture. Valid values are x86 and arm64.
  -B <build>    seed the toolchain from the COS build <build>.
                Example: R93-16623.0.0 Requires -b option.
  -C <config>   kernel config target. Example: lakitu_defconfig
  -G <bucket>   seed the toolchain and kernel headers from the custom
                GCS bucket <bucket>. Directory structure needs to conform
                to the COS standard. 
  -H            create a package with kernel headers for the respective
                kernel package. Should be used only with -k option.
  -O <objdir>   value for KBUILD_OUTPUT to separate obj files from
                sources
  -R <release>  seed the toolchain and kernel headers from the
                specified official COS release. Example: 16442.0.0
  -b <board>    specify board for -B argument. Example: lakitu
  -c            perform "mrproper" step when building a kernel package or
                "clean" step when building a module.
                Should be used only with -k and -m option.
  -d            create a pakcage with debug symbols for the respective
                kernel package. Should be used only with -k option.
  -h            show this message.
  -i            invoke interactive shell with kernel development
                environment initialized.
  -k            build a kernel package for sources mapped from the host
                to the current working directory.
  -m            build an out-of-tree module for sources mapped from
                the host to the current working directory.
                This mode requires either -R or -B/b options.
  -t            seed the toolchain from the Chromium OS upstream.
                Example: 2021.06.26.094653
```

### Building the COS Kernel

#### Getting the Source Code

`cos_kmake` should be used at the top-level of the checked out kernel source tree:

```
$ git clone -b cos-5.10 https://cos.googlesource.com/third_party/kernel cos-kernel
$ cd cos-kernel
```

#### Pass-through make

Unless one of `-i`, `-k` or `-m` command-line switches is specified the container
acts as a wrapper for a `make` command so the build procedure for the
kernel looks the same as a normal build:

```
cos_kmake mrproper
cos_kmake lakitu_defconfig
cos_kmake bzImage modules
```

`cos-kernel-devenv` supports two target arcitectgures: `x86` and `arm64` that
can be set for an invocation by passing `-A <arch>` argument to the command.
Unless specified the `x86` target is the default one.

To build ARM64 kernel and modules you can use the following sequence of commands:

```
cos_kmake -A arm64 mrproper
cos_kmake -A arm64 lakitu_defconfig
cos_kmake -A arm64 Image modules
```

All examples below build binaries for `x86` architecture but can be converted to
`arm64` by adding `-A arm64` command-line switch.

#### Building Kernel Packages

In addition to acting as a `make` wrapper, `cos-kernel-devenv` can also create
kernel packages: archives with the kernel, modules, and debug symbols. These
packages can be used to inject kernel into a custom image or in a VM in
development mode.

To build kernel package run `cos_kmake -k`. This command produces
`cos-kernel-<version>-<arch>.txz` file that contains `/boot` and `/lib/modules`
directories that can be used as a drop-in replacement for the COS image/VM.

By default kernel configuration step uses `defconfig` as a target. You can
override this by passing `-C <kernelconfig>` argument, i.e.:
`-C lakitu_defconfig`,  `-C olddefconfig`.

To ensure clean build you can also add `make mrproper` step to the beginning
of the build sequence by passing `-c` command line switch.

Debug symbols package can be produced by using `-d` command-line switch.
When this switch passed to the container `cos-kernel-devenv` also generates
`cos-kernel-debug-<version>-<arch>.txz` archive with debug symbols
for the kernel and modules. This package can be used for kernel instrumention
if required.

#### Building Out of Tree Modules

The main purpose of this mode is to build out-of-tree kernel module
for the specific COS version. The version can be either an official
COS release or a CI build. To use an official COS release pass it
as an argument to `-R` command line switch, i.e.: 
```
cos_kmake -m -R 16442.0.0
```

CI builds is identified by a combination of board and build number
passed as arguments for `-b` and `-B` switches respectively:

```
cos_kmake -m -b lakitu -B R93-16623.0.0

```

`cos_kmake -m ...` is an equivalent of running two commands in the working
directory:

```
make -C /path/to/kheaders M=$(pwd) clean
make -C /path/to/kheaders M=$(pwd) modules
```

#### Interactive Mode

For more complex use cases expert developers can use an interactive shell,
activated by `-i` command-line switch. The shell has kernel and development
environment variable pre-configured. It also defines a `make` wrapper `kmake`
that can be used as a shorthand to run make for kernel-related tasks:

```
% cos_kmake -i -A arm64
** Kernel architecture: arm64
** Toolchain architecture: aarch64
Mode: cross
[INFO    2021-10-29 18:41:02 UTC] Configuring environment variables for
cross-compilation
Starting interactive shell for the kernel devenv
root@0244bcbd8239:/src# $CC -v
Chromium OS 12.0_pre422132_p20210405-r9 clang version 13.0.0
(/var/tmp/portage/sys-devel/llvm-12.0_pre422132_p20210405-r9/work/llvm-12.0_pre422132_p20210405/clang
cd442157cff4aad209ae532cbf031abbe10bc1df)
Target: aarch64-cros-linux-gnu
Thread model: posix
InstalledDir: /build/cros-2021.06.26.094653-aarch64/usr/bin
Found candidate GCC installation:
/build/cros-2021.06.26.094653-aarch64/usr/bin/../lib/gcc/aarch64-cros-linux-gnu/10.2.0
Selected GCC installation:
/build/cros-2021.06.26.094653-aarch64/usr/bin/../lib/gcc/aarch64-cros-linux-gnu/10.2.0
Candidate multilib: .;@m64
Selected multilib: .;@m64
root@0244bcbd8239:/src# kmake kernelrelease
5.10.75-ovt
root@0244bcbd8239:/src# 
```

When interactive shell invoked with `-R` or `-b/-B` switches the location of
kernel headers for the specified COS version is accessible as `KHEADERS` env
variable.
