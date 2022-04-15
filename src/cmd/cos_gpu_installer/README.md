# COS GPU installer V2

cos\_gpu\_installer is a docker containers that can be used to download,
compile and install GPU drivers on Container-Optimized OS images.

## Usage

cos\_gpu\_installer should run on COS VM instances. Once you connect to your
COS VM, run the following command to start a cos\_gpu\_installer container:
```
  /usr/bin/docker run --rm \
    --name="cos-gpu-installer" \
    --privileged \
    --net=host \
    --pid=host \
    --volume /dev:/dev \
    --volume /:/root \
    "gcr.io/cos-cloud/cos-gpu-installer:<tag>" install "-host-dir=/var/lib/nvidia"
```

To see all available flags, run the following command:

```
/usr/bin/docker run --rm "gcr.io/cos-cloud/cos-gpu-installer:<tag>" help
```

## Test

### Source code
Currently only unittest is available. Use `go test` to run unittest.

### GPU drivers availability
The test `test/check_drivers_test.go` is available for checking GPU drivers
availability. It checks which drivers are available for live COS images.
Use `test/run_test.sh` to run the test.
