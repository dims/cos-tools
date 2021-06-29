# COS Node Profiler

cos\_node\_profiler\_agent is a docker container that can streamline the process of
collecting debugging information for COS Nodes on GKE. It will collect important
information on the system that can be used to help the oncall diagnose
performance issues. The agent will provide functionality to upload all collected
information to Google Cloud Logging to make it easier for the oncall to
access and examine debugging information.

## Usage

cos\_node\_profiler\_agent is deployed as a docker container. To run the
program, open `cos_node_profiler` directory, then run `make`. This will
produce an executable. The executable takes in flags as follows:
```
 ./<executable name> -project <name of project to write logs to>
```
or
```
 ./<executable name> --project <name of project to write logs to>
```
