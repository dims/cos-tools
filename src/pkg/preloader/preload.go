// Copyright 2018 Google LLC
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

// Package preloader contains functionality for preloading a COS image from
// provided configuration.
package preloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"cos.googlesource.com/cos/tools.git/src/pkg/config"
	"cos.googlesource.com/cos/tools.git/src/pkg/fs"
	"cos.googlesource.com/cos/tools.git/src/pkg/provisioner"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"

	"cloud.google.com/go/storage"
)

// storeInGCS stores the given files in GCS using the given gcsManager.
// Files to store are provided in a map where each key is a file on the local
// file system and each value is the relative path in GCS at which to store the
// corresponding key. The provided relative paths in GCS must be unique.
func storeInGCS(ctx context.Context, gcs *gcsManager, files map[string]string) error {
	gcsRelPaths := make(map[string]bool)
	for _, gcsRelPath := range files {
		if gcsRelPaths[gcsRelPath] {
			return fmt.Errorf("storeInGCS: collision in relative path %q", gcsRelPath)
		}
		gcsRelPaths[gcsRelPath] = true
	}
	for file, gcsRelPath := range files {
		r, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("error opening %q: %v", file, err)
		}
		defer r.Close()
		if err := gcs.store(ctx, r, gcsRelPath); err != nil {
			return err
		}
	}
	return nil
}

func needScratchDisk(provConfig *provisioner.Config) bool {
	for _, step := range provConfig.Steps {
		if step.Type == "InstallGPU" {
			return true
		}
	}
	return false
}

func needDiskResize(provConfig *provisioner.Config, buildSpec *config.Build) bool {
	// We need to resize the disk during provisioning if:
	// 1. The requested disk size is larger than default, and
	// 2. Partitions need to be relocated, i.e. we are enlarging the OEM partition
	// or reclaiming /dev/sda3
	return buildSpec.DiskSize > 10 && (provConfig.BootDisk.OEMSize != "" || provConfig.BootDisk.ReclaimSDA3)
}

// writeDaisyWorkflow templates the given Daisy workflow and writes the result to a temporary file.
// The given workflow should be the one at //data/build_image.wf.json.
func writeDaisyWorkflow(inputWorkflow string, outputImage *config.Image, buildSpec *config.Build, provConfig *provisioner.Config) (string, error) {
	tmplContents, err := ioutil.ReadFile(inputWorkflow)
	if err != nil {
		return "", err
	}
	labelsJSON, err := json.Marshal(outputImage.Labels)
	if err != nil {
		return "", err
	}
	acceleratorsJSON, err := json.Marshal([]map[string]interface{}{})
	if err != nil {
		return "", err
	}
	if buildSpec.GPUType != "" {
		acceleratorType := fmt.Sprintf("projects/%s/zones/%s/acceleratorTypes/%s",
			buildSpec.Project, buildSpec.Zone, buildSpec.GPUType)
		acceleratorsJSON, err = json.Marshal([]map[string]interface{}{
			{"acceleratorType": acceleratorType, "acceleratorCount": 1}})
		if err != nil {
			return "", err
		}
	}
	licensesJSON, err := json.Marshal(outputImage.Licenses)
	if err != nil {
		return "", err
	}

	// template content for the scratch disk.
	// This disk is used for certain tasks that require additional disk space.
	// We use a scratch disk in order to keep from increasing the boot disk size.
	// The thinking here is that steps that require temporary files for their devops
	// process can use this scratch disk instead of writing to the boot disk directly.
	var scratchDiskJson string
	var scratchDiskSource string
	if needScratchDisk(provConfig) {
		scratchDiskJson = `
      {
        "Name": "scratch-disk",
        "SourceImage": "scratch",
        "Type": "${disk_type}",
        "SizeGb": "10"
      },`
		scratchDiskSource = `{"Source": "scratch-disk"},`
	}

	// template content for the step resize-disk.
	// If the oem-size is set, or need to reclaim sda3 (with disk-size-gb set),
	// create the disk with the default size, and then resize the disk.
	// Otherwise, a place holder is used. The disk is created with provided disk-size-gb or
	// the default size. And the disk will not be resized.
	// The place holder is needed because ResizeDisk API requires a larger size than the original disk.
	var resizeDiskJSON string
	var waitResizeJSON string
	if needDiskResize(provConfig, buildSpec) {
		// actual disk size
		resizeDiskJSON = fmt.Sprintf(`"ResizeDisks": [{"Name": "boot-disk","SizeGb": "%d"}]`, buildSpec.DiskSize)
		waitResizeJSON = `
      "WaitForInstancesSignal": [
        {
          "Name": "preload-vm",
          "Interval": "10s",
          "SerialOutput": {
            "Port": 3,
            "SuccessMatch": "waiting for the boot disk size to change",
            "FailureMatch": "BuildFailed:"
          }
        }
      ]`
	} else {
		// placeholder
		resizeDiskJSON = `"WaitForInstancesSignal": [{"Name": "preload-vm","Interval": "10s","SerialOutput": {"Port": 3,"SuccessMatch": "BuildStatus:"}}]`
		waitResizeJSON = `"WaitForInstancesSignal": [{"Name": "preload-vm","Interval": "10s","SerialOutput": {"Port": 3,"SuccessMatch": "BuildStatus:"}}]`
	}
	tmpl, err := template.New("workflow").Parse(string(tmplContents))
	if err != nil {
		return "", err
	}
	w, err := ioutil.TempFile(fs.ScratchDir, "daisy-")
	if err != nil {
		return "", err
	}
	if err := tmpl.Execute(w, struct {
		Labels            string
		Accelerators      string
		Licenses          string
		ResizeDisks       string
		WaitResize        string
		ScratchDisks      string
		ScratchDiskSource string
	}{
		string(labelsJSON),
		string(acceleratorsJSON),
		string(licensesJSON),
		resizeDiskJSON,
		waitResizeJSON,
		scratchDiskJson,
		scratchDiskSource,
	}); err != nil {
		w.Close()
		os.Remove(w.Name())
		return "", err
	}
	if err := w.Close(); err != nil {
		os.Remove(w.Name())
		return "", err
	}
	return w.Name(), nil
}

func mcopyFiles(path string, files *fs.Files) (err error) {
	return utils.RunCommand([]string{"mcopy", "-i", path, files.ProvConfig, "::/config.json"}, "", nil)
}

func writeImage(fileName string) (path string, err error) {
	img, err := ioutil.TempFile(fs.ScratchDir, "img-")
	if err != nil {
		return "", err
	}
	if err := img.Close(); err != nil {
		return "", err
	}
	if err := utils.CopyFile(fileName, img.Name()); err != nil {
		return "", err
	}
	return img.Name(), err
}

func tarImage(imageName string) (path string, err error) {
	out, err := ioutil.TempFile(fs.ScratchDir, "img-tar-")
	if err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	// tar with the "z" option requires a shell to be installed in the container.
	// To avoid the shell dependency, gzip the tar ourselves.
	if err := utils.RunCommand([]string{
		"tar",
		"cf", out.Name(),
		"--transform", fmt.Sprintf("s|%s|disk.raw|g", strings.TrimLeft(imageName, "/")),
		imageName,
	}, "", nil); err != nil {
		return "", err
	}
	if err := fs.GzipFile(out.Name(), out.Name()+".gz"); err != nil {
		return "", err
	}
	return out.Name() + ".gz", err
}

func updateProvConfig(provConfig *provisioner.Config, buildSpec *config.Build, buildContexts map[string]string, gcs *gcsManager, files *fs.Files) error {
	if needDiskResize(provConfig, buildSpec) {
		provConfig.BootDisk.WaitForDiskResize = true
	}
	provConfig.BuildContexts = buildContexts
	for idx := range provConfig.Steps {
		if provConfig.Steps[idx].Type == "InstallGPU" {
			var step provisioner.InstallGPUStep
			if err := json.Unmarshal(provConfig.Steps[idx].Args, &step); err != nil {
				return err
			}
			if step.GCSDepsPrefix != "" {
				step.GCSDepsPrefix = gcs.managedDirURL() + "/gcs_files"
			}
			buf, err := json.Marshal(&step)
			if err != nil {
				return err
			}
			provConfig.Steps[idx].Args = json.RawMessage(buf)
		}
	}
	buf, err := json.Marshal(provConfig)
	if err != nil {
		return err
	}
	log.Printf("Using provisioner config: %s", string(buf))
	return config.SaveConfigToPath(files.ProvConfig, provConfig)
}

func sanitize(output *config.Image) {
	var licenses []string
	for _, l := range output.Licenses {
		if l != "" {
			licenses = append(licenses, strings.TrimPrefix(l, "https://www.googleapis.com/compute/v1/"))
		}
	}
	output.Licenses = licenses
}

// daisyArgs computes the parameters to the cos-customizer Daisy workflow (//data/build_image.wf.json)
// and uploads dependencies to GCS.
func daisyArgs(ctx context.Context, gcs *gcsManager, files *fs.Files, input *config.Image, output *config.Image, buildSpec *config.Build, provConfig *provisioner.Config) ([]string, error) {
	sanitize(output)
	buildContexts := map[string]string{
		"user": gcs.managedDirURL() + "/" + filepath.Base(files.UserBuildContextArchive),
	}
	toUpload := map[string]string{
		files.UserBuildContextArchive: filepath.Base(files.UserBuildContextArchive),
	}
	for _, gcsFile := range buildSpec.GCSFiles {
		toUpload[gcsFile] = path.Join("gcs_files", filepath.Base(gcsFile))
	}
	if err := storeInGCS(ctx, gcs, toUpload); err != nil {
		return nil, err
	}
	daisyWorkflow, err := writeDaisyWorkflow(files.DaisyWorkflow, output, buildSpec, provConfig)
	if err != nil {
		return nil, err
	}
	if err := updateProvConfig(provConfig, buildSpec, buildContexts, gcs, files); err != nil {
		return nil, err
	}
	ciDataFile, err := writeImage(files.CIDataImg)
	if err != nil {
		return nil, err
	}
	if err := mcopyFiles(ciDataFile, files); err != nil {
		return nil, err
	}
	ciDataFileTar, err := tarImage(ciDataFile)
	if err != nil {
		return nil, err
	}
	scratchImgFileTar, err := tarImage(files.ScratchImg)
	if err != nil {
		return nil, err
	}
	var args []string
	if buildSpec.GCEEndpoint != "" {
		// Adding the "projects/" suffix is a Daisy-specific quirk.
		u, err := url.Parse(buildSpec.GCEEndpoint)
		if err != nil {
			return nil, fmt.Errorf("error parsing %q: %v", buildSpec.GCEEndpoint, err)
		}
		u.Path = path.Join(u.Path, "projects")
		// We add a trailing "/" here because Daisy needs it, and u.String() trims
		// it.
		args = append(args, "-compute_endpoint_override="+u.String()+"/")
	}
	if provConfig.BootDisk.OEMSize == "" && buildSpec.DiskSize > 10 && !provConfig.BootDisk.ReclaimSDA3 {
		// If the oem-size is set, or need to reclaim sda3,
		// create the disk with default size,
		// and then resize the disk in the template step "resize-disk".
		// Otherwise, create the disk with the provided disk-size-gb.
		args = append(args, "-var:disk_size_gb", strconv.Itoa(buildSpec.DiskSize))
	}
	if output.Family != "" {
		args = append(args, "-var:output_image_family", output.Family)
	}
	hostMaintenance := "MIGRATE"
	if buildSpec.GPUType != "" {
		hostMaintenance = "TERMINATE"
	}
	args = append(
		args,
		"-var:source_image",
		input.URL(),
		"-var:output_image_name",
		output.Name,
		"-var:output_image_project",
		output.Project,
		"-var:cidata_img",
		ciDataFileTar,
		"-var:scratch_img",
		scratchImgFileTar,
		"-var:machine_type",
		buildSpec.MachineType,
		"-var:service_account",
		buildSpec.ServiceAccount,
		"-var:disk_type",
		buildSpec.DiskType,
		"-var:host_maintenance",
		hostMaintenance,
		"-var:network",
		buildSpec.Network,
		"-var:subnet",
		buildSpec.Subnet,
		"-gcs_path",
		gcs.managedDirURL(),
		"-project",
		buildSpec.Project,
		"-zone",
		buildSpec.Zone,
		"-default_timeout",
		buildSpec.Timeout,
		"-disable_gcs_logging",
		daisyWorkflow,
	)
	return args, nil
}

// BuildImage builds a customized image using Daisy.
func BuildImage(ctx context.Context, gcsClient *storage.Client, files *fs.Files, input, output *config.Image,
	buildSpec *config.Build, provConfig *provisioner.Config) error {
	gcs := &gcsManager{gcsClient, buildSpec.GCSBucket, buildSpec.GCSDir}
	defer gcs.cleanup(ctx)
	args, err := daisyArgs(ctx, gcs, files, input, output, buildSpec, provConfig)
	if err != nil {
		return err
	}
	cmd := exec.Command(files.DaisyBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	return cmd.Run()
}
