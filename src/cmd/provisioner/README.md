# The Provisioner

This tool is not meant to be invoked directly and is instead meant to run on the preload vm
created by the `finish-image-build` step in the
[cos-customizer](https://cos.googlesource.com/cos/tools/+/refs/heads/master/src/cmd/cos_customizer/)

The command line tool and it's configs are packaged in an image named cidata that is formatted with the
FAT filesystem. This image is uploaded and mounted onto the preload VM at `/mnt/disks/cidata` and the
provisioner is executed by a cloud-init script called
[startup.yaml](https://cos.googlesource.com/cos/tools/+/refs/heads/master/src/data/startup.yaml).
This cloud-init file creates two files:

- `/tmp/startup.sh`
- `/etc/systemd/system/customizer@.service`

The systemd service customizer@.service calls the startup.sh script which will then call the provisioner.
The provisioner reads from a file called the `prov.config`, also packaged in the cidata disk, which contains all
the optional steps specified by the cos_customizer. The implementation of these steps are located at //src/pkg/provisioner.
Here are the available implementations of those steps:

- disable_auto_update_step.go
- install_gpu_step.go
- install_packages_step.go
- run_script_step.go
- seal_oem_step.go

You'll notice that these steps map exactly to the optional steps in the
[cos-customizer](https://cos.googlesource.com/cos/tools/+/refs/heads/master/src/cmd/cos_customizer/).