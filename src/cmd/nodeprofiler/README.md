# COS Node Profiler

cos\_node\_profiler\_agent is a docker container that can streamline the process of
collecting debugging information for COS Nodes on GKE. It will collect important
information on the system that can be used to help COS on call developers diagnose
performance issues. The agent will provide functionality to upload all collected
information to Google Cloud Logging to make it easier for developers on call to
access and examine debugging information.

## Usage

cos\_node\_profiler\_agent is deployed as a docker container. To run the
app, open `nodeprofiler` directory located in the COS `tools` repository under
src/cmd/, then run `make`. This will produce an executable that takes in flags as follows:
```
 ./<executable name> --project=<name of project to write logs to> \
 --cmd=<name of the command to run> \
 --cmd-count=<the number of time to execute raw commands> \
 --cmd-interval=<the interval between the collection of raw command output in
 seconds> \
 --cmd-timeout=<the amount of time in second it will take for the agent
 to kill the execution of raw command if there is no output> --profiler-count=<the number of time to generate USEReport> \
 --profiler-interval=<the interval between the collection of USEReport in
 seconds>
```

Example:
```
./nodeprofiler --project="interns-playground" \
--cmd="lscpu" \
--cmd-count=3  \
--cmd-interval=2 \
--cmd-timeout=5 \
--profiler-count=3 \
--profiler-interval=60
```

If the `--cmd` flag is set and the user did not provide a cmd-count and/or a
cmd-interval, then the `--cmd-count` will be set to 1 by default and the
`--cmd-interval` to 0. If the `--cmd` flag is empty , then the user should not
provide any `--cmd-count` or `--cmd-interval` flags. Otherwise, the program will
return an error. Note that the `--cmd-timeout` flag will be set to 300 seconds
by default if the user did not specify a timeout option. The profiler-count will
be set to 1 by default if the user did not specify how often to run the profiler.

## Instruction for building the COS Node Profiler Docker image

To build a docker image for the node profiler, it is important to be in the
tools directory in which `go.mod` and `go.sum` files are present because the
Docker image needs to copy dependencies from the aforementioned files and get
context from them. There are two ways to build the image for the profiler agent:
The first is to use the `make image` command from `tools/src/cmd/nodeprofiler`
directory. The second way is to manually build the image by running the
following command from the `tools` directory:
`docker build -t <image-name> -f src/cmd/nodeprofiler/Dockerfile .` These two
options will produce a docker image locally.

## Instructions for running the COS Node Profiler Docker container

There are two ways to run a docker container with the COS Node Profiler image on
it. The first is to use the `make container` command from
`tools/src/cmd/nodeprofiler` directory. The second is to manually run the
container using the following shell command (note that a docker image has to be
created first):
```
docker run -it <image-name> --project=<name of project to write logs to> \
 --cmd=<name of the command to run> \
 --cmd-count=<the number of time to execute raw commands> \
 --cmd-interval=<the interval between the collection of raw command output in
 seconds> \
 --cmd-timeout=<the amount of time in second it will take for the agent to kill the execution of raw command if there is no output> \
 --profiler-count=<the number of time to generate USEReport> \
 --profiler-interval=<the interval between the collection of USEReport in
 seconds>
```
The above command will spin up a Docker Container with the COS Node Profiler
image on it and will execute commands specified by the flags provided (To set
flags, simply replace the prepopulated values but your custom values).

## Instructions for pushing the container to GCR

To push the container to Google Container Registry (GCR), run the following
commands:
Tag the image: `docker tag <image-name> gcr.io/<project-id>/<image-name>`
Push container: `docker push gcr.io/<project-id>/<image-name>`
Further instructions can be found at: https://cloud.google.com/container-registry/docs/pushing-and-pulling

## Instructions for pulling the latest COS Node Profiler image from GCR

To pull the Node Profiler image from GCR, run one of the following two options:
- Pulling by tag:
`docker pull gcr.io/cos-interns-playground/cos_node_profiler:latest`
- Pulling by digest:
`docker pull gcr.io/cos-interns-playground/cos_node_profiler@sha256:057cf8603afd62195eec8ed6afe25fbbc0af64bec1029b6f634de3ea8e1eb799`
Note that these references may be updated. They are valid as of July 29, 2021.
In general, pulling an image follows this pattern:
`docker pull gcr.io/<name of project where the image is>/<name of the image>:tag`
If you want a specific version of the image, then you can tag that version. If
you wish to get the latest version then you can replace `tag` with `latest`.

## Instructions for deploying the COS Node Profiler Docker Container on GKE

To deploying the Profiler on a GKE cluster, ensure that a daemonset similar to
the one below is present in the working directory.

### Sample Daemonset to deploy the COS Profiler on a GKE Cluster.

```
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cos-node-profiler
spec:
  selector:
      matchLabels:
        name: cos-node-profiler # Label selector that determines which Pods belong to the DaemonSet
  template:
    metadata:
      labels:
        name: cos-node-profiler # Pod template's label selector
    spec:
      containers:
      - name: cos-node-profiler
        image: gcr.io/cos-interns-playground/cos_node_profiler
        command: ["/nodeprofiler"]
        args:
        - "--project=<name of project to write logs to>"
        - "--cmd=<name of the command to run>"
        - "--cmd-count=<the number of time to execute raw commands>"
        - "--cmd-interval=<the interval between the collection of raw command output in seconds>"
        - "--cmd-timeout=<the amount of time in second it will take for the agent to kill the execution of raw command if there is no output> --profiler-count=<the number of time to generate USEReport>"
        - "--profiler-interval=<the interval between the collection of USEReport in seconds>"
```
Once this file is present in the working directory, create a GKE cluster with
the following command: `gcloud container clusters create <name of cluster>`.
Then, run: `kubectl create -f <daemonsetfile.yml>`. Note that the `<daemonsetfile>`
should be a `.yml` file.

If you wish to get information about your daemonset, run the following command:
`kubectl describe ds cos-node-profiler`. This will print the status of the
daemonset. To view the outputs of the application running in individuals nodes
on your GKE cluster, you can first get the name of the pods containing those
nodes by running: `kubect get pods`. This will generate a list of all pods from
which you can collect logs.
Here is a sample output (note that the STATUS Column can vary):
```
$ kubectl get pods
NAME                      READY   STATUS              RESTARTS   AGE
cos-node-profiler-5jfp6   0/1     Completed           0          6s
cos-node-profiler-c54jb   0/1     ContainerCreating   0          6s
cos-node-profiler-rrmc9   0/1     ContainerCreating   0          6s
```
To get logs for the first pod in this scenario, one can simply run `$kubectl logs cos-node-profiler-5jfp6`.
Then, repeat the process to find logs for other pods.

To delete the daemonset in this scenario, a user can just run `kubectl delete ds cos-node-profiler`. This will
delete all pods and nodes associated with the daemonset.

More information about kubernetes daemonsets can be found at: https://kubernetes.io/docs/reference/kubectl/overview/
