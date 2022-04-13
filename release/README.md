# Release process

Container images for container source code present in [cos/tools](https://cos.googlesource.com/cos/tools) are built utilizing the Google [cloud build](https://cloud.google.com/build) service. An automated build system which utilizes cloud build [triggers](https://cloud.google.com/build/docs/automating-builds/create-manage-triggers) builds the container images whenever changes are pushed on the cos/tools git repo. Any change to the container images source code automatically triggers new builds for all the containers in this repo.

The recipes for building the containers are specified as [dockerfiles](https://docs.docker.com/engine/reference/builder) (for eg. toolbox [Dockerfile](https://cos.googlesource.com/cos/tools/+/refs/heads/master/src/cmd/toolbox/Dockerfile)). Further [buildx](https://docs.docker.com/buildx/working-with-buildx/) is used to create images for multiple target architectures(x86 64, ARM64).

For each new build executed by the automated triggers, the built container images are pushed to an internal google container image registry with unique tag labels which are the GIT commit sha of the change being updated.

The release process is a multi-party code reviewed automated process(see source [here](https://cos.googlesource.com/cos/tools/+/refs/heads/master/release)). This is also achieved by using the cloud build triggers on the cos/tools repository. The release candidates (state of the releases) live in a config file([source](https://cos.googlesource.com/cos/tools/+/refs/heads/master/release/release-versions.yaml)). When changes are made to the release candidates, an automated cloud build process copies the specified container images from the internal google container image registry to the public [cos-cloud](gcr.io/cos-cloud) container registry.

## Validating release config - TODO (rnv) - convert to presubmit cloud build
task

After making changes to the release
[state](https://cos.googlesource.com/cos/tools/+/refs/heads/master/release/release-versions.yaml)
the validity of the changes can be checked by running the following command from
the root of the repo:

`gcloud builds submit --config=presubmit.yaml`

where presubmit.yaml could look like the following:

```
steps:
- name: 'gcr.io/google.com/cloudsdktool/cloud-sdk:latest'
  entrypoint: 'bash'
  args: ['-c',
  'pip3 install -r release/requirements.txt && python3 release/release.py --verify'
  ]
```
