# Profilertest package
Profilertest package runs end to end tests for the APIs written in the profiler package. It does this by simulating perfomance regression for different components, generating their USE reports and checking that the stress on each component was reflected in the USE metrics collected.

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

# Run tests
To run the stress tests, the following command can be run from package profilertest
```
 go test . -v
 ```