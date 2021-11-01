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

type ToGCECmd struct {
	InputURL     string
	ImageProject string
	ImageName    string
	GcsBucket    string
}

// Name returns the name of the command.
func (tgc *ToGCECmd) Name() string {
	return "to-gce"
}

// Synopsis returns short description of the command.
func (tgc *ToGCECmd) Synopsis() string {
	return "Converts the Input OVA image " +
		"to raw format and uploads to GCE Project"
}

// Usage returns instructions on how to use the command.
func (tgc *ToGCECmd) Usage() string {
	return "Converts the Input OVA image " +
		"to raw format and uploads to GCE Project"
}

// SetFlags adds the flags to the specified set.
func (tgc *ToGCECmd) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&tgc.GcsBucket, "gcs-bucket", "",
		"GCS bucket for the working dir. It is mandatory. Example: sample-bucket")
	fs.StringVar(&tgc.InputURL, "input-url", "",
		"URL to the Input OVA Image. It is mandatory. Example: gs://sample-bucket/input.ova")
	fs.StringVar(&tgc.ImageProject, "image-project", "",
		"Project in which the image is to be created. It is mandatory. Example: test-project")
	fs.StringVar(&tgc.ImageName, "image-name", "",
		"Name of the Image to be create. It is mandatory. Example: input-gce")
}

// Execute executes the command (converts OVA to GCE) and returns the CommandStatus
func (otgc *ToGCECmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if err := otgc.validateInput(); err != nil {
		glog.Errorf("failed to parse flags: %v", err)
		return subcommands.ExitFailure
	}
	exitStatus := subcommands.ExitSuccess

	converter := ovaconverter.NewConverter(ctx)

	if err := converter.ConvertOVAToGCE(ctx, otgc.InputURL, otgc.ImageName,
		otgc.GcsBucket, otgc.ImageProject); err != nil {
		glog.Errorf("failed to convert to the gce image: %v", err)
		exitStatus = subcommands.ExitFailure
	}
	return exitStatus
}

// validateInput validates the input from flags parsed and returns error when the
// mandatory input values are not present
func (tgc *ToGCECmd) validateInput() error {
	var errMsgs []string
	if tgc.GcsBucket == "" {
		errMsgs = append(errMsgs, "gcs-bucket is mandatory")
	}
	if tgc.InputURL == "" {
		errMsgs = append(errMsgs, "input-url is mandatory")
	}
	if tgc.ImageName == "" {
		errMsgs = append(errMsgs, "image-name is mandatory")
	}
	if tgc.ImageProject == "" {
		errMsgs = append(errMsgs, "image-project is mandatory")
	}
	if len(errMsgs) > 0 {
		return errors.New(strings.Join(errMsgs, ";"))
	}
	return nil
}
