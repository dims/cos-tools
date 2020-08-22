# COS Image Analyzer

COS Image Analyzer is a Linux based command line tool written in Go that can analyze a single COS image or compare two COS images for its relevant binary and package information. The differences include the filesystem difference, OS configuration difference, package version upgrades, etc. The tool has customizable flags below for different use cases. Output is default to the terminal. 

## Usage
Run ./cos_image_analyzer -h/-help
```
NAME
	cos_image_analyzer - finds all meaningful differences of two COS Images (binary and package differences).
		If only one image is passed in, its binary info and package info will be returned.

SYNOPSIS
	%s [-local] FILE-1 [FILE-2] (default true)
		FILE - the local file path to the DOS/MBR boot sector file of your image (Ex: disk.raw)
		Ex: %s image-cos-77-12371-273-0/disk.raw image-cos-81-12871-119-0/disk.raw

	%s -local -binary=Sysctl-settings,OS-config -package=false image-cos-77-12371-273-0/disk.raw

	%s -gcs GCS-PATH-1 [GCS-PATH-2]
		GCS-PATH - the GCS "gs://bucket/object" path for the COS Image ("object" is type .tar.gz)
		Ex: %s -gcs gs://my-bucket/cos-images/cos-77-12371-273-0.tar.gz gs://my-bucket/cos-images/cos-81-12871-119-0.tar.gz


DESCRIPTION
	Input Flags:
	-local (default true, flag is optional)
		input is one or two DOS/MBR disk file on the local filesystem. If the images are downloaded from
		Google Cloud as a tarball, decompress the tarball first then pass the disk.raw file to the program.
	-gcs
		input is one or two objects stored on Google Cloud Storage of type (.tar.gz). This flag temporarily downloads,
		unzips, and loop device mounts the images into this tool's directory.
		To download images from Google Cloud Storage, you need to pass a service account credential to the program.
		Folllow https://cloud.google.com/docs/authentication/production#create_service_account to create a service account and
		download the service account key. Then point environment variable GOOGLE_APPLICATION_CREDENTIALS to the key file then
		run the program.

	Difference Flags:
	-binary (string)
		specify which type of binary difference to show. Types "Version", "BuildID", "Kernel-command-line",
		"Partition-structure", "Sysctl-settings", and "Kernel-configs" are supported for one and two image. "Rootfs",
		"Stateful-partition", and "OS-config" are only supported for two images. To list multiple types separate by
		comma. To NOT list any binary difference, set flag to "false". (default all types)
	-package
		specify whether to show package difference. Shows addition/removal of packages and package version updates.
		To NOT list any package difference, set flag to false. (default false)

	Attribute Flags
	-verbose
		include flag to increase verbosity of Rootfs, Stateful-partition, and OS-config differences. See -compress-rootfs and
		-compress-stateful flags descriptions for the directories that are compressed by default.
	-compress-rootfs (string)
		to customize which directories are compressed in a non-verbose Rootfs and OS-config difference output, provide a local
		file path to a .txt file. Format of the file must be one root file path per line with an ending back slash and no commas.
		By default the directory(s) that are compressed during a diff are /bin/, /lib/modules/, /lib64/, /usr/libexec/, /usr/bin/,
		/usr/sbin/, /usr/lib64/, /usr/share/zoneinfo/, /usr/share/git/, /usr/lib/, /sbin/, /etc/ssh/, /etc/os-release/ and
		/etc/package_list/.
	-compress-stateful (string)
		to customize which directories are compressed in a non-verbose Stateful-partition difference output, provide a local
		file path to a .txt file. Format of file must be one root file path per line with no commas. By default the directory(s)
		that are compressed during a diff are /var_overlay/db/.

	Output Flags:
	-output (string)
		Specify format of output. Only "terminal" stdout or "json" object is supported. (default "terminal")

OUTPUT
	Based on the "-output" flag. Either "terminal" stdout or machine readable "json" format.

NOTE
	The root permission is needed for this program because it needs to mount images into your local filesystem to calculate difference.
```

## Code Layout 

main.go - The controller of execution: Parse images, find binary and package difference, output to the user, and then clean up. 

internal/input/ - Package dedicated to parsing input flags and arguments to setup for execution. Temporary directory is create and all necessary partitions are mounted. 

internal/binary/ - Collects, determines, and formats all binary differences.

internal/packagediff/ - Collects, determines, and formats all package differences.

internal/output/ - Final formatting of output at the end of execution.

internal/utilities/ -  Helper functions used throughout the project (GCS_download, logical helpers, etc).

internal/testdata/ - Testing data for all packages. 


## Documentation

go/cos-image-analyzer
