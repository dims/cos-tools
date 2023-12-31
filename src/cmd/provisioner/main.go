// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// provisioner is a tool for provisioning COS instances. The tool is intended to
// run on a running COS machine.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"cloud.google.com/go/storage"
	"github.com/google/subcommands"

	"cos.googlesource.com/cos/tools.git/src/pkg/provisioner"
)

var (
	stateDir = flag.String("state-dir", "/var/lib/.cos-customizer", "Absolute path to the directory to use for provisioner state. "+
		"This directory is used for persisting internal state across reboots, unpacking inputs, and running provisioning scripts. "+
		"The size of the directory scales with the size of the inputs.")
	dockerCredentialGCR = flag.String("docker-credential-gcr", "", "Path to the docker-credential-gcr executable to use during provisioning.")
	veritySetupImage    = flag.String("veritysetup-image", "", "Path to the veritysetup file system tarball to use as a Docker container during provisioning.")
	handleDiskLayoutBin = flag.String("handle-disk-layout-bin", "", "Path to the handle_disk_layout executable to use during provisioning.")
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(&Run{}, "")
	subcommands.Register(&Resume{}, "")
	flag.Parse()
	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Println(err)
		os.Exit(int(subcommands.ExitFailure))
	}
	deps := provisioner.Deps{
		GCSClient:           gcsClient,
		TarCmd:              "tar",
		SystemctlCmd:        "systemctl",
		RootdevCmd:          "rootdev",
		CgptCmd:             "cgpt",
		Resize2fsCmd:        "resize2fs",
		E2fsckCmd:           "e2fsck",
		RootDir:             "/",
		DockerCredentialGCR: *dockerCredentialGCR,
		VeritySetupImage:    *veritySetupImage,
		HandleDiskLayoutBin: *handleDiskLayoutBin,
	}
	var exitCode int
	ret := subcommands.Execute(ctx, deps, &exitCode)
	if ret != subcommands.ExitSuccess {
		os.Exit(int(ret))
	}
	os.Exit(exitCode)
}
