// Copyright 2021 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"flag"
	"strings"

	"github.com/golang/glog"
	"github.com/google/subcommands"

	"cos.googlesource.com/cos/tools.git/src/pkg/ovaconverter"
)

type FromGCECmd struct {
	DestinationPath string
	ImageProject    string
	SourceImage     string
	GcsBucket       string
	Zone            string
}

// Name returns the name of the command.
func (fgc *FromGCECmd) Name() string {
	return "from-gce"
}

// Synopsis returns short description of the command.
func (fgc *FromGCECmd) Synopsis() string {
	return "Converts the Input GCE image " +
		"to OVA format and uploads to destination path"
}

// Usage returns instructions on how to use the command.
func (fgc *FromGCECmd) Usage() string {
	return "Converts the Input GCE image " +
		"to OVA format and uploads to destination path"
}

// SetFlags adds the flags to the specified set.
func (fgc *FromGCECmd) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&fgc.GcsBucket, "gcs-bucket", "",
		"GCS bucket for the working dir. It is mandatory. Example: sample-bucket")
	fs.StringVar(&fgc.Zone, "zone", "us-west1-b",
		"Zone is required when exporting a GCE image. It is optional, by default it is us-west1-b. Example: us-west1-b")
	fs.StringVar(&fgc.DestinationPath, "destination-path", "",
		"URL to the save the OVA Image. It is mandatory. Example: gs://output-bucket/output.ova")
	fs.StringVar(&fgc.ImageProject, "image-project", "",
		"Project in which the source image is present. It is mandatory. Example: input-project")
	fs.StringVar(&fgc.SourceImage, "source-image", "",
		"Name of the Source Image. It is mandatory. Example: input-gce")
}

// Execute executes the command (converts GCE to OVA) and returns the CommandStatus
func (fgc *FromGCECmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if err := fgc.validateInput(); err != nil {
		glog.Errorf("failed to parse flags: %v", err)
		return subcommands.ExitFailure
	}
	exitStatus := subcommands.ExitSuccess

	gceToOVAConverterConfig := args[0].(*ovaconverter.GCEToOVAConverterConfig)
	converter := ovaconverter.NewConverter(ctx)

	if err := converter.ConvertOVAFromGCE(ctx, fgc.SourceImage, fgc.DestinationPath,
		fgc.GcsBucket, fgc.ImageProject, fgc.Zone, gceToOVAConverterConfig); err != nil {
		glog.Errorf("failed to create the OVA image: %v", err)
		exitStatus = subcommands.ExitFailure
	}
	return exitStatus
}

// validateInput validates the input from flags parsed and returns error when the
// mandatory input values are not present
func (tgc *FromGCECmd) validateInput() error {
	var errMsgs []string
	if tgc.GcsBucket == "" {
		errMsgs = append(errMsgs, "gcs-bucket is mandatory")
	}
	if tgc.DestinationPath == "" {
		errMsgs = append(errMsgs, "destination-path is mandatory")
	}
	if tgc.SourceImage == "" {
		errMsgs = append(errMsgs, "source-image is mandatory")
	}
	if tgc.ImageProject == "" {
		errMsgs = append(errMsgs, "image-project is mandatory")
	}
	if len(errMsgs) > 0 {
		return errors.New(strings.Join(errMsgs, ";"))
	}
	return nil
}
