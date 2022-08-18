# COS Customizer Design

The design can be thought of as steps that aren't the `finish-image-build` step
and the step that is.

## Steps that aren't `finish-image-build`

The start-image and all optional steps are involved in creating and modifying a pair of configs
called the `prov.config` and `build.config`. Both these configs live on the Cloud Build VM (the builder VM),
The `build.config` instructs
[Daisy](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy)
on which resources (disks, vm's) to create and which project to create them in.
The `prov.config` file is used to instruct the provisioner, a binary that is executed
on the "preload VM", a VM whose bootdisk is exported as the customized image.
The provisioner is the binary that executes all optional steps that a user will have
specified. For example, run-script, install-gpu, seal-oem, and anthos-install are all
steps that the provisioner has implemented.

## The `finish-image-build` step

This step is implemented in two phases:

1. The preloading phase
2. The provisioner phase

The *preloading phase* calls a command line tool called 
[daisy](https://github.com/GoogleCloudPlatform/compute-daisy)
and is implemented here
[preloader](https://cos.googlesource.com/cos/tools/+/refs/heads/master/src/pkg/preloader).
The preloader generates a daisy config file based on the buildspec and calls daisy which
creates all the GCP resources necessary for creating a custom COS image. A few disks
and a VM are created, and the disks are mounted to the VM. One of these disks is the boot
disk which is what will eventually be exported as the customized COS image. Another disk is called "cidata"
which packages the provisioner and the provisioner's configs which will be used in the next step.

The *provisioner phase* calls a command line tool called the 
[provisioner](https://cos.googlesource.com/cos/tools/+/refs/heads/master/src/cmd/provisioner)
that gets pulled into the preload VM from the previously mentioned "cidata" disk. 
The provisioner is what executes the optional
steps that you have specified in your `cloudbuild.yaml` file. It will run any scripts
install any artifacts, or seal any partitions that were specified prior to this
step. Once it finishes executing, the preloading phase will proceed to cleanup any left over resources
and exit the build.
