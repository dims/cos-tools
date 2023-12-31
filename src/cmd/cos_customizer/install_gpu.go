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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"cos.googlesource.com/cos/tools.git/src/pkg/config"
	"cos.googlesource.com/cos/tools.git/src/pkg/fs"
	"cos.googlesource.com/cos/tools.git/src/pkg/provisioner"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"

	"cloud.google.com/go/storage"
	"github.com/google/subcommands"
	"google.golang.org/api/iterator"
)

const (
	gpuScript                 = "install_gpu.sh"
	defaultInstallerContainer = "gcr.io/cos-cloud/cos-gpu-installer:v20230815"
)

// InstallGPU implements subcommands.Command for the "install-gpu" command.
// This command configures the current image build process to customize the result image
// with GPU drivers.
type InstallGPU struct {
	NvidiaDriverVersion  string
	NvidiaDriverMd5sum   string
	NvidiaInstallDirHost string
	gpuType              string
	getValidDrivers      bool
	gpuDataDir           string
	installerContainer   string
}

// Name implements subcommands.Command.Name.
func (*InstallGPU) Name() string {
	return "install-gpu"
}

// Synopsis implements subcommands.Command.Synopsis.
func (*InstallGPU) Synopsis() string {
	return "Configure the image build with GPU drivers."
}

// Usage implements subcommands.Command.Usage.
func (*InstallGPU) Usage() string {
	return `install-gpu [flags]
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (i *InstallGPU) SetFlags(f *flag.FlagSet) {
	f.StringVar(&i.NvidiaDriverVersion, "version", "", "Driver version to install. Can also be the name of an nvidia installer present in the "+
		"directory specified by '-deps-dir'; e.g., NVIDIA-Linux-x86_64-450.51.06.run.")
	f.StringVar(&i.NvidiaDriverMd5sum, "md5sum", "", "Md5sum of the driver to install.")
	f.StringVar(&i.NvidiaInstallDirHost, "install-dir", "/var/lib/nvidia",
		"Location to install drivers on the image.")
	f.StringVar(
		&i.gpuType, "gpu-type", "nvidia-tesla-p100", "This flag is deprecated, please use the same flag in step 'finish-image-build'.")
	f.BoolVar(
		&i.getValidDrivers, "get-valid-drivers", false,
		"Print the list of supported GPU driver versions. If this flag is given, no other actions will be taken.")
	f.StringVar(&i.gpuDataDir, "deps-dir", "", "If provided, the local directory to search for cos-gpu-installer data dependencies. "+
		"The exact data dependencies that must be present in this directory depends on the version of cos-gpu-installer "+
		"used by cos-customizer. Do not expect this flag to be stable; it exists for compatibility with pre-release COS images.")
	f.StringVar(&i.installerContainer, "installer-container", defaultInstallerContainer, "Use the provided cos-gpu-installer-v1 container, for e.g. gcr.io/cos-cloud/cos-gpu-installer:latest")
}

func validDriverVersions(ctx context.Context, gcsClient *storage.Client) (map[string]bool, error) {
	// We gather the set of valid drivers from the set of drivers provided by Nvidia in their GCS bucket.
	// Nominally, paths we care about in this bucket look like 'tesla/<version>/<binaries>'. Version 390.46 has
	// a deprecated path structure, and since it's supported by cos-gpu-installer, we special case that here.
	validDrivers := map[string]bool{"390.46": true}
	query := &storage.Query{Prefix: "tesla/"}
	it := gcsClient.Bucket("nvidia-drivers-us-public").Objects(ctx, query)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// Example object: tesla/396.26/NVIDIA-Linux-x86_64-396.26-diagnostic.run
		if splitPath := strings.SplitN(objAttrs.Name, "/", 3); len(splitPath) > 1 {
			validDrivers[splitPath[1]] = true
		}
	}
	return validDrivers, nil
}

func (i *InstallGPU) validate(ctx context.Context, gcsClient *storage.Client, files *fs.Files, provConfig *provisioner.Config) error {
	if i.NvidiaDriverVersion == "" {
		return fmt.Errorf("version must be set")
	}
	var gpuAlreadyConf bool
	for _, s := range provConfig.Steps {
		if s.Type == "InstallGPU" {
			gpuAlreadyConf = true
			break
		}
	}
	if gpuAlreadyConf {
		return fmt.Errorf("install-gpu can only be invoked once in an image build process. Only one driver version can be installed on the image")
	}
	if strings.HasSuffix(i.NvidiaDriverVersion, ".run") {
		log.Printf("driver version is set to %q, which looks like an nvidia installer file", i.NvidiaDriverVersion)
		if i.gpuDataDir == "" {
			return errors.New(`"-deps-dir" must be set when the version is specified as an nvidia installer file`)
		}
		fileName := filepath.Join(i.gpuDataDir, i.NvidiaDriverVersion)
		info, err := os.Stat(fileName)
		if os.IsNotExist(err) {
			return fmt.Errorf("nvidia installer file at %q does not exist", fileName)
		}
		if info.IsDir() {
			return fmt.Errorf("nvidia installer file at %q is a directory", fileName)
		}
	} else {
		validDrivers, err := validDriverVersions(ctx, gcsClient)
		if err != nil {
			return err
		}
		if !validDrivers[i.NvidiaDriverVersion] {
			var drivers []string
			for d := range validDrivers {
				drivers = append(drivers, d)
			}
			return fmt.Errorf("driver version %s is not valid; valid driver versions are: %v", i.NvidiaDriverVersion, drivers)
		}
	}
	return nil
}

func (i *InstallGPU) templateScript(scriptPath string) error {
	setCOSDownloadGCS := ""
	if i.gpuDataDir != "" {
		setCOSDownloadGCS = "true"
	}
	data := struct {
		NvidiaDriverVersion  string
		NvidiaDriverMd5sum   string
		NvidiaInstallDirHost string
		SetCOSDownloadGCS    string
	}{
		NvidiaDriverVersion:  utils.QuoteForShell(i.NvidiaDriverVersion),
		NvidiaDriverMd5sum:   utils.QuoteForShell(i.NvidiaDriverMd5sum),
		NvidiaInstallDirHost: utils.QuoteForShell(i.NvidiaInstallDirHost),
		SetCOSDownloadGCS:    utils.QuoteForShell(setCOSDownloadGCS),
	}
	tmpl, err := template.New(filepath.Base(scriptPath)).ParseFiles(scriptPath)
	if err != nil {
		return err
	}
	w, err := os.Create(scriptPath)
	if err != nil {
		return err
	}
	defer w.Close()
	return tmpl.Execute(w, data)
}

func (i *InstallGPU) updateBuildConfig(configPath string) error {
	buildConfig := &config.Build{}
	configFile, err := os.OpenFile(configPath, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer configFile.Close()
	if err := config.Load(configFile, buildConfig); err != nil {
		return err
	}
	buildConfig.GPUType = i.gpuType
	if i.gpuDataDir != "" {
		files, err := ioutil.ReadDir(i.gpuDataDir)
		if err != nil {
			return fmt.Errorf("error reading dir %q: %v", i.gpuDataDir, err)
		}
		for _, f := range files {
			if f.Mode().IsRegular() {
				buildConfig.GCSFiles = append(buildConfig.GCSFiles, filepath.Join(i.gpuDataDir, f.Name()))
			}
		}
	}
	if _, err := configFile.Seek(0, 0); err != nil {
		return err
	}
	return config.SaveConfigToFile(configFile, buildConfig)
}

func (i *InstallGPU) updateProvConfig(provConfig *provisioner.Config) error {
	buf, err := json.Marshal(&provisioner.InstallGPUStep{
		NvidiaDriverVersion:      i.NvidiaDriverVersion,
		NvidiaDriverMD5Sum:       i.NvidiaDriverMd5sum,
		NvidiaInstallDirHost:     i.NvidiaInstallDirHost,
		NvidiaInstallerContainer: i.installerContainer,
		// GCSDepsPrefix will be converted into a gs:// address by the preloader
		// package.
		GCSDepsPrefix: i.gpuDataDir,
	})
	if err != nil {
		return err
	}
	provConfig.Steps = append(provConfig.Steps, provisioner.StepConfig{
		Type: "InstallGPU",
		Args: json.RawMessage(buf),
	})
	return nil
}

// Execute implements subcommands.Command.Execute. It configures the current image build process to
// customize the result image with GPU drivers.
func (i *InstallGPU) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	if len(args) < 2 {
		log.Panic("InstallGPU expects two arguments; *fs.Files and ServiceClients")
	}
	files, ok := args[0].(*fs.Files)
	if !ok {
		log.Panic("InstallGPU expects two arguments; *fs.Files and ServiceClients")
	}
	serviceClients, ok := args[1].(ServiceClients)
	if !ok {
		log.Panic("InstallGPU expects two arguments; *fs.Files and ServiceClients")
	}
	_, gcsClient, err := serviceClients(ctx, false)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	defer gcsClient.Close()
	if i.getValidDrivers {
		validDrivers, err := validDriverVersions(ctx, gcsClient)
		if err != nil {
			log.Println(err)
			return subcommands.ExitFailure
		}
		var drivers []string
		for d := range validDrivers {
			drivers = append(drivers, d)
		}
		log.Printf("Valid driver versions are: %v\n", drivers)
		return subcommands.ExitSuccess
	}
	var provConfig provisioner.Config
	if err := config.LoadFromFile(files.ProvConfig, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := i.validate(ctx, gcsClient, files, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := i.updateBuildConfig(files.BuildConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := i.updateProvConfig(&provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := config.SaveConfigToPath(files.ProvConfig, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
