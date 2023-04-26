// Copyright 2023 Google LLC
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

package sbomutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/fs"
	"cos.googlesource.com/cos/tools.git/src/pkg/gcs"
	spdx_common "github.com/spdx/tools-golang/spdx/v2/common"
	spdx2_2 "github.com/spdx/tools-golang/spdx/v2/v2_2"
)

const (
	spdxDocID               = "SPDXRef-DOCUMENT"
	spdxDocRef              = "DocumentRef-%s"
	spdxRef                 = "SPDXRef-%s"
	spdxNoAssert            = "NOASSERTION"
	docNameSuffix           = "sbom.json"
	creatorToolName         = "gcr.io/cos-cloud/cos-customizer"
	spdxJsonFileNotAnalyzed = `      "filesAnalyzed": false,`
	defaultRootPkgVersion   = "0"
)

type SBOMCreator struct {
	sbomInput               *SBOMInput
	sbomOutput              *spdx2_2.Document
	ctx                     context.Context
	gcsClient               *storage.Client
	files                   *fs.Files
	filesNotAnalyzedPkgRefs []string
}

// NewSBOMCreator creates a new SBOMCreator.
func NewSBOMCreator(ctx context.Context, gcsClient *storage.Client, files *fs.Files) *SBOMCreator {
	return &SBOMCreator{
		sbomInput: &SBOMInput{},
		sbomOutput: &spdx2_2.Document{
			SPDXVersion:       spdx2_2.Version,
			DataLicense:       spdx2_2.DataLicense,
			DocumentNamespace: spdxNoAssert,
			SPDXIdentifier:    spdx_common.ElementID(spdxDocID),
		},
		ctx:       ctx,
		gcsClient: gcsClient,
		files:     files,
	}
}

type SBOMInput struct {
	OutputImageName         string                  `json:"outputImageName,omitempty"`
	OutputImageVersion      string                  `json:"outputImageVersion,omitempty"`
	Creators                []string                `json:"creators"`
	Supplier                string                  `json:"supplier,omitempty"`
	SPDXPackages            []*spdx2_2.Package      `json:"SPDXPackages,omitempty"`
	SBOMPackages            []*SBOMPackage          `json:"SBOMPackages,omitempty"`
	ExtractedLicensingInfos []*spdx2_2.OtherLicense `json:"hasExtractedLicensingInfos,omitempty"`
}

type SBOMPackage struct {
	Name          string                        `json:"name"`
	SpdxDocument  string                        `json:"spdxDocument"`
	Algorithm     spdx_common.ChecksumAlgorithm `json:"algorithm"`
	ChecksumValue string                        `json:"checksumValue"`
}

func (pkg *SBOMPackage) toExternalRef() spdx2_2.ExternalDocumentRef {
	return spdx2_2.ExternalDocumentRef{
		DocumentRefID: fmt.Sprintf(spdxDocRef, pkg.Name),
		URI:           pkg.SpdxDocument,
		Checksum: spdx_common.Checksum{
			Algorithm: pkg.Algorithm,
			Value:     pkg.ChecksumValue,
		},
	}
}

// ParseSBOMInput parses the user input and saves the result in the SBOMCreator.
func (s *SBOMCreator) ParseSBOMInput(sbomInputPath string) error {
	inputBytes, err := fs.ReadObjectFromArchive(s.files.UserBuildContextArchive, sbomInputPath)
	if err != nil {
		return fmt.Errorf("failed to read SBOM input %q, err: %v", sbomInputPath, err)
	}
	if err := json.Unmarshal(inputBytes, s.sbomInput); err != nil {
		return fmt.Errorf("failed to unmarshal %q, err: %v, input content: %q", sbomInputPath, err, string(inputBytes))
	}
	return nil
}

// Use NOASSERTION to fill required but empty fields.
func (s *SBOMCreator) fillNoAssertion() {
	for _, pkg := range s.sbomOutput.Packages {
		if pkg.PackageDownloadLocation == "" {
			pkg.PackageDownloadLocation = spdxNoAssert
		}
		if pkg.PackageSupplier.Supplier == "" {
			pkg.PackageSupplier.Supplier = spdxNoAssert
			pkg.PackageSupplier.SupplierType = spdxNoAssert
		}
		if pkg.PackageLicenseConcluded == "" {
			pkg.PackageLicenseConcluded = spdxNoAssert
		}
		if pkg.PackageLicenseDeclared == "" {
			pkg.PackageLicenseDeclared = spdxNoAssert
		}
	}
}

var timeNow = func() string {
	return fmt.Sprintf("%v", time.Now().UTC().Format(time.RFC3339))
}

// There is a bug in unmarshaling field "filesAnalyzed"
// https://github.com/spdx/tools-golang/issues/209
// Solve the issue manually until it is fixed upstream.
func (s *SBOMCreator) addFilesNotAnalyzed(content string) string {
	lines := strings.Split(content, "\n")
	for _, ref := range s.filesNotAnalyzedPkgRefs {
		for idx, line := range lines {
			if line != ref {
				continue
			}
			lines = append(lines[:idx+1], lines[idx:]...)
			lines[idx] = spdxJsonFileNotAnalyzed
			break
		}
	}
	return strings.Join(lines, "\n")
}

// GenerateSBOM uses the parsed input to generate an SPDX SBOM.
func (s *SBOMCreator) GenerateSBOM(actualOutputImageName string) error {
	// Add SBOM creation info.
	s.sbomOutput.CreationInfo = &spdx2_2.CreationInfo{
		Created: timeNow(),
		Creators: []spdx_common.Creator{
			{
				CreatorType: "Tool",
				Creator:     creatorToolName,
			},
		},
	}
	for _, creator := range s.sbomInput.Creators {
		c := strings.Split(creator, ": ")
		if len(c) != 2 {
			return fmt.Errorf("invalid creator format %q, it should be \"Person/Organization/Tool: name\"", c)
		}
		s.sbomOutput.CreationInfo.Creators = append(s.sbomOutput.CreationInfo.Creators, spdx_common.Creator{
			CreatorType: c[0],
			Creator:     c[1],
		})
	}

	// Use the actual output image name as SBOM output image name.
	if s.sbomInput.OutputImageName == "" {
		s.sbomInput.OutputImageName = actualOutputImageName
		s.sbomInput.OutputImageVersion = ""
	}

	rootPkg := &spdx2_2.Package{
		PackageName:           s.sbomInput.OutputImageName,
		PackageVersion:        s.sbomInput.OutputImageVersion,
		PackageSPDXIdentifier: spdx_common.ElementID(fmt.Sprintf(spdxRef, s.sbomInput.OutputImageName)),
		FilesAnalyzed:         false,
	}
	if rootPkg.PackageVersion == "" {
		rootPkg.PackageVersion = defaultRootPkgVersion
	}
	if s.sbomInput.Supplier != "" {
		c := strings.Split(s.sbomInput.Supplier, ": ")
		if len(c) != 2 {
			return fmt.Errorf("invalid supplier format %q, it should be \"Person/Organization: name\"", c)
		}
		rootPkg.PackageSupplier = &spdx_common.Supplier{
			SupplierType: c[0],
			Supplier:     c[1],
		}
	}
	if s.sbomInput.OutputImageVersion == "" {
		s.sbomOutput.DocumentName = fmt.Sprintf("%s_%s", s.sbomInput.OutputImageName, docNameSuffix)
	} else {
		s.sbomOutput.DocumentName = fmt.Sprintf("%s-%s_%s", s.sbomInput.OutputImageName, s.sbomInput.OutputImageVersion, docNameSuffix)
	}

	s.filesNotAnalyzedPkgRefs = append(s.filesNotAnalyzedPkgRefs, fmt.Sprintf("      \"SPDXID\": \"%s\",", rootPkg.PackageSPDXIdentifier))

	// Add root package and relationship for doc describing root package.
	s.sbomOutput.Packages = append(s.sbomOutput.Packages, rootPkg)
	s.sbomOutput.Relationships = append(s.sbomOutput.Relationships, &spdx2_2.Relationship{
		RefA:         spdx_common.DocElementID{ElementRefID: spdxDocID},
		RefB:         spdx_common.DocElementID{ElementRefID: rootPkg.PackageSPDXIdentifier},
		Relationship: spdx_common.TypeRelationshipDescribe,
	})

	// Add SPDX packages and relationship for root package containing all those pacakges.
	for _, pkg := range s.sbomInput.SPDXPackages {
		s.sbomOutput.Packages = append(s.sbomOutput.Packages, pkg)
		s.sbomOutput.Relationships = append(s.sbomOutput.Relationships, &spdx2_2.Relationship{
			RefA:         spdx_common.DocElementID{ElementRefID: rootPkg.PackageSPDXIdentifier},
			RefB:         spdx_common.DocElementID{ElementRefID: pkg.PackageSPDXIdentifier},
			Relationship: spdx_common.TypeRelationshipContains,
		})
		if !pkg.FilesAnalyzed {
			s.filesNotAnalyzedPkgRefs = append(s.filesNotAnalyzedPkgRefs, fmt.Sprintf("      \"SPDXID\": \"%s\",", fmt.Sprintf(spdxRef, pkg.PackageSPDXIdentifier)))
		}
	}

	// Add SBOM packages and relationship for root package containing all those pacakges.
	for _, pkg := range s.sbomInput.SBOMPackages {
		extRef := pkg.toExternalRef()
		s.sbomOutput.ExternalDocumentReferences = append(s.sbomOutput.ExternalDocumentReferences, extRef)
		s.sbomOutput.Relationships = append(s.sbomOutput.Relationships, &spdx2_2.Relationship{
			RefA:         spdx_common.DocElementID{ElementRefID: rootPkg.PackageSPDXIdentifier},
			RefB:         spdx_common.DocElementID{ElementRefID: spdx_common.ElementID(extRef.DocumentRefID)},
			Relationship: spdx_common.TypeRelationshipContains,
		})
	}

	// Add extracted license info.
	s.sbomOutput.OtherLicenses = s.sbomInput.ExtractedLicensingInfos

	s.fillNoAssertion()
	return nil
}

// UploadSBOMToGCS uploads the generated SBOM to GCS in JSON format.
func (s *SBOMCreator) UploadSBOMToGCS(outputGCSPath string) error {
	sbomOutputBytes, err := json.MarshalIndent(s.sbomOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to convert SBOM document into json: %v", err)
	}
	sbomOutputString := s.addFilesNotAnalyzed(string(sbomOutputBytes))

	sbomOutputURL := fmt.Sprintf("%s/%s", outputGCSPath, s.sbomOutput.DocumentName)
	if err := gcs.UploadGCSObjectString(s.ctx, s.gcsClient, sbomOutputString, sbomOutputURL); err != nil {
		return fmt.Errorf("Failed to upload SBOM to GCS %q, err: %v", outputGCSPath, err)
	}
	return nil
}
