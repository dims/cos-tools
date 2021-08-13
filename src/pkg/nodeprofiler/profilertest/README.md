# Profilertest package
Profilertest package runs end to end tests for the APIs written in the profiler package. It does this by simulating perfomance regression for different components, generating their USE reports and checking that the stress on each component was reflected in the USE metrics collected.

# Usage
End to end tests can be run in a Docker container, which can be built using the
Dockerfile in the `profilertest` package. This ensures that end to end tests are
isolated from other processes running on the host machine. To build a docker
image for the profilertest package, it is important to be in the tools directory
in which go.mod and go.sum files are present because the Docker image needs to
copy dependencies from the aforementioned files and get context from them. Then,
run the following commands:
```
docker build -t <image-name> -f src/pkg/nodeprofiler/profilertest/Dockerfile .
```
The above command will create a Docker image that can be used to create a Docker
container. To run the container, it is best to set runtime options with memory
and cpu limits. This can be done as follows:
```
docker run -it --cpus="<percentage of cpu to use>" -m=<number of megabytes to
give to the container>m <image-name>
```

Example:
The following will allocate 50% of the CPU every second and 200 megabytes of
memory to the container with `profiler-test` image on it:
```
docker run -it --cpus=".5" -m=200m profiler-test
```
The above command will create an interactive shell from which we can navigate
to the profilertest package as follows:
```
cd /work/src/pkg/nodeprofiler/profilertest/
```
To run the tests, use the following command:
```
 go test . -v
```

More information about specifying runtime options with memory and cpu
information can be found at: https://docs.docker.com/config/containers/resource_constraints/

# Tools
To accomplish this, the tests makes use of the Linux package "stress-ng" which can stress test a Linux computer system in various selectable ways. The tool can be run as a shell command and can be configured by passing in a number of options. This configuration can be general options to modify the behavior of stress-ng or it can be specific to the type of stressor being used. Here's a general format of how to use the command:
```
stress-ng <general stress-ng options> <type of stressor> <number of workers> <stressor-specific options>
```

Examples: 
```
stress-ng --cpu 2 --cpu-load 95 -t 10s
```
load CPU, and specifically 2 processors, with 95 percent loading for the 2 CPU stress workers (each for 1 processor) for 10 seconds

```
stress-ng --vm-bytes 256M -m 1 --vm-keep
```
start 1 stress worker that will continually write to the allocated memory thus overloading memory capacity

```
stress-ng --help
```
shows more information about package usage


Further documentation on the stress-ng package can be found by typing "man stress-ng" or on ubuntu's man pages: https://manpages.ubuntu.com/manpages/artful/man1/stress-ng.1.html

# Installing stress-ng 
The version of "stress-ng" used to run tests in package profilertest is 0.09.50.
To download the "stress-ng" package, the following command can be used:
```
 $ sudo apt install stress-ng (on Ubuntu/Debian)
 $ yum install stress-ng (on Fedora/CentOS)
```
