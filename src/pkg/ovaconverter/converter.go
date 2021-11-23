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

package ovaconverter

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	"google.golang.org/api/compute/v1"

	"cos.googlesource.com/cos/tools.git/src/pkg/fs"
	"cos.googlesource.com/cos/tools.git/src/pkg/gce"
	"cos.googlesource.com/cos/tools.git/src/pkg/gcs"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

const (
	vmdkFileExtension = ".vmdk"
)

type Converter struct {
	GCSClient      *storage.Client
	ComputeService *compute.Service
}

func NewConverter(ctx context.Context) *Converter {
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil
	}
	svc, err := compute.NewService(ctx)
	if err != nil {
		return nil
	}
	return &Converter{
		GCSClient:      gcsClient,
		ComputeService: svc,
	}
}

// ConvertOVAToGCE converts the OVA file at GCS Location to a GCE image.
func (converter *Converter) ConvertOVAToGCE(ctx context.Context, inputURL, imageName, gcsBucket, imageProject string) error {
	// Create a temporary working directory
	tempWorkDir, err := os.MkdirTemp("", "ova_dir")
	if err != nil {
		return err
	}
	defer utils.RemoveDir(tempWorkDir, "error on removing the temporary working directory", nil)

	glog.Info("Downloading OVA from the input GCS URL")
	inputFile := filepath.Join(tempWorkDir, "input.ova")
	if err = gcs.DownloadGCSObject(ctx, converter.GCSClient,
		inputURL, inputFile); err != nil {
		return err
	}

	extractWorkDir := filepath.Join(tempWorkDir, "extractWorkDir")
	glog.Info("Converting OVA to VMDK...")
	if err = fs.ExtractFile(inputFile, extractWorkDir); err != nil {
		return err
	}

	var vmdkFile string
	files, _ := ioutil.ReadDir(extractWorkDir)
	for _, file := range files {
		if filepath.Ext(file.Name()) == vmdkFileExtension {
			vmdkFile = file.Name()
			break
		}
	}

	glog.Info("Converting VMDK to Raw...")
	tempRawImage := filepath.Join(tempWorkDir, "disk.raw")
	if err = utils.ConvertImageToRaw(filepath.Join(extractWorkDir,
		vmdkFile), tempRawImage); err != nil {
		return err
	}

	glog.Info("Compressing disk.raw to tar.gz...")
	if err = fs.TarFile(tempRawImage, filepath.Join(tempWorkDir,
		"cos_gce.tar.gz")); err != nil {
		return err
	}

	cosGCETar := fmt.Sprintf("%s_gce.tar.gz", imageName)
	cosTarURL := fmt.Sprintf("gs://%s/%s", gcsBucket, cosGCETar)

	glog.Info("Uploading tar.gz file to a remote GCS location...")
	if err = gcs.UploadGCSObject(ctx, converter.GCSClient, filepath.Join(tempWorkDir, cosGCETar), cosTarURL); err != nil {
		return err
	}
	// delete the image staged temporarily before creating a GCE image
	defer func() {
		if gcs.DeleteGCSObject(ctx, converter.GCSClient, cosTarURL); err != nil {
			glog.Warningf("error deleting the GCS temporary Object: %v", err)
		}
	}()

	glog.Info("Creating a GCS Image...")
	return gce.CreateImage(converter.ComputeService, filepath.Join(gcsBucket, cosGCETar),
		imageName, imageProject)

}

// ConvertOVAFromGCE converts GCE Image to OVA Format.
func (converter *Converter) ConvertOVAFromGCE(ctx context.Context, sourceImage, destinationPath, gcsBucket, imageProject, zone string,
	gceToOVAConverterConfig *GCEToOVAConverterConfig) error {
	tempWorkDir, err := ioutil.TempDir("", "ovaDir")
	if err != nil {
		return err
	}
	defer utils.RemoveDir(tempWorkDir, "error on removing the temporary working directory", nil)

	tempExportedImageName := fmt.Sprintf("%s-exported-image.tar.gz", sourceImage)
	tempExportImageURL := fmt.Sprintf("gs://%s/%s", gcsBucket, tempExportedImageName)

	glog.Info("Exporting image in tar.gz format to a temporary GCS location")
	// Export the GCE image to a temporary location
	if err = exportImageFromGCEUsingDaisy(sourceImage, imageProject, tempExportImageURL, zone,
		gceToOVAConverterConfig.DaisyBin, gceToOVAConverterConfig.DaisyWorkflowPath); err != nil {
		return err
	}

	glog.Info("Downloading the image in tar.gz...")
	downloadImagePath := filepath.Join(tempWorkDir, tempExportedImageName)
	if err = gcs.DownloadGCSObject(ctx, converter.GCSClient,
		tempExportImageURL, downloadImagePath); err != nil {
		return err
	}

	defer func() {
		if err = gcs.DeleteGCSObject(ctx, converter.GCSClient, tempExportImageURL); err != nil {
			glog.Warningf("error deleting the GCS temporary Object: %v", err)

		}
	}()

	glog.Info("Extracting the disk.raw image from the tar file...")
	extractWorkDir := filepath.Join(tempWorkDir, "extractDir")
	if err = fs.ExtractFile(downloadImagePath, extractWorkDir); err != nil {
		return err
	}

	glog.Info("Converting the disk.raw image to vmdk format...")
	tempVMDKImageName := filepath.Join(tempWorkDir, "chromiumos_image.vmdk")
	if err = utils.ConvertImageToVMDK(filepath.Join(extractWorkDir, "disk.raw"), tempVMDKImageName); err != nil {
		return err
	}

	// convert to OVA image
	glog.Info("Converting the VMDK to OVA image...")
	tempOVAImage := filepath.Join(tempWorkDir, filepath.Base(destinationPath))
	oVAImageName := strings.ReplaceAll(filepath.Base(destinationPath),
		filepath.Ext(filepath.Base(destinationPath)), "")
	if err = utils.RunCommand([]string{
		gceToOVAConverterConfig.MakeOVAScript,
		"-d", tempVMDKImageName, "-o", tempOVAImage, "-p", "GKE On-Prem", "-n",
		oVAImageName, "-t", gceToOVAConverterConfig.OVATemplate,
	}, "", nil); err != nil {
		return err
	}

	glog.Info("Uploading the OVA file to the GCS URL...")
	if err = gcs.UploadGCSObject(ctx, converter.GCSClient, tempOVAImage,
		destinationPath); err != nil {
		return err
	}
	return nil
}

// exportImageFromGCEUsingDaisy exports an image to the gce.tar.gz file by initiating a
// daisy workflow.
// Input: daisyBin - path to the daisy binary, daisyWorkflowPath - path to the image exporter workflow.
func exportImageFromGCEUsingDaisy(imageName, imageProject, destinationFile, zone, daisyBin, daisyWorkflowPath string) error {
	sourceImage := fmt.Sprintf("-var:source_image=projects/%s/global/images/%s", imageProject, imageName)
	destination := fmt.Sprintf("-var:destination=%s", destinationFile)
	exportImageUsingDaisyCommand := []string{
		daisyBin, "-project", imageProject, "-zone", zone, sourceImage, destination, daisyWorkflowPath,
	}
	return utils.RunCommand(exportImageUsingDaisyCommand, "", nil)
}
