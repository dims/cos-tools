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

package ovaconverter

const (
	// makeOVAScriptPath is the location of make_ova.sh script
	makeOVAScriptPath = "/crosutils/cos/make_ova.sh"
	// ovaTemplate is the location of template used for OVA
	ovaTemplatePath = "/crosutils/cos/template.ovf"
	// daisyBin is the location of the Daisy binary.
	daisyBin = "/daisy"
	// daisyWorkflowPath is the location to the image_export.wf.json workflow.
	daisyWorkflowPath = "/workflows/export/image_export.wf.json"
)

// GCEToOVAConverterConfig holds the required configuration about the
// daisy bin path, make OVA script path and template path
type GCEToOVAConverterConfig struct {
	DaisyBin          string
	MakeOVAScript     string
	OVATemplate       string
	DaisyWorkflowPath string
}

func GetDefaultOVAConverterConfig() *GCEToOVAConverterConfig {
	return &GCEToOVAConverterConfig{
		DaisyBin:          daisyBin,
		MakeOVAScript:     makeOVAScriptPath,
		OVATemplate:       ovaTemplatePath,
		DaisyWorkflowPath: daisyWorkflowPath,
	}
}
