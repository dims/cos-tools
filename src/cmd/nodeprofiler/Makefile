#
# Makefile to run the COS_Node_Profiler Application that utilizes the profiler
# package to fetch performance data from a GKE Cluster, the cloudlogger package
# to export that data to Google Cloud Logging and the utils package that defines
# the interface between the first two packages.
#

# Building the binary file
build:
	go build
# Running the profiler application
run:
	go build && ./nodeprofiler --project cos-interns-playground
# Cleaning used executable file
clean:
	go clean
# Running the profiler application and cleaning binaries
.PHONY: all
all:
	go build && ./nodeprofiler --project cos-interns-playground && go clean

