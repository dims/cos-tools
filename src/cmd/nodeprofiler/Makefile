#
# Makefile to run the COS_Node_Profiler Application that utilizes the profiler
# package to fetch performance data from a GKE Cluster, the cloudlogger package
# to export that data to Google Cloud Logging and the utils package that defines
# the interface between the first two packages.
#

# Building the binary file
build:
	go build
# Cleaning used executable file
clean:
	go clean
# Testing the tool with arbitrary commands.
.PHONY: test
test:
	go build && ./nodeprofiler --project="cos-interns-playground" --profiler-count=1 --profiler-interval=0 --cmd="lscpu" --cmd-count=1 --cmd-interval=0 --cmd-timeout=4 && go clean

# Building container image
.PHONY: image
image:
	cd ../../../ && docker build -t cos_node_profiler -f src/cmd/nodeprofiler/Dockerfile .

# Running docker container
.PHONY: container
container:
	cd ../../../ && docker build -t cos_node_profiler -f src/cmd/nodeprofiler/Dockerfile .
	docker run -it cos_node_profiler
