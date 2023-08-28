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
	"io/ioutil"
	"testing"

	"cos.googlesource.com/cos/tools.git/src/pkg/config"
	"cos.googlesource.com/cos/tools.git/src/pkg/fakes"
	"github.com/google/go-cmp/cmp"
	spdx_common "github.com/spdx/tools-golang/spdx/v2/common"
	spdx2_2 "github.com/spdx/tools-golang/spdx/v2/v2_2"
)

func TestGenerateSBOM(t *testing.T) {
	timeNow = func() string { return "2023-04-27T16:08:32Z" }
	testData := []struct {
		testName  string
		sbomInput *SBOMInput
		want      *spdx2_2.Document
		wantErr   bool
	}{
		{
			testName: "ValidInput",
			sbomInput: &SBOMInput{
				OutputImageName:    "image1",
				OutputImageVersion: "123",
				Creators:           []string{"Organization: G"},
				Supplier:           "Organization: G",
				SPDXPackages: []*spdx2_2.Package{
					{
						PackageName:           "pkg1",
						PackageSPDXIdentifier: "SPDXRef-pkg1",
						PackageVersion:        "111",
						PackageSupplier: &spdx_common.Supplier{
							Supplier:     "K",
							SupplierType: "Organization",
						},
						FilesAnalyzed:           false,
						PackageLicenseDeclared:  "BSD-3-Clause AND Apache-2.0",
						PackageLicenseConcluded: "BSD-3-Clause AND Apache-2.0",
						PackageDownloadLocation: "test.com",
						PackageExternalReferences: []*spdx2_2.PackageExternalReference{
							{
								Category: "SECURITY",
								RefType:  "cpe23Type",
								Locator:  "cpe:/a:vendor:pkg-1:1.2.3",
							},
						},
						PackageChecksums: []spdx_common.Checksum{
							{
								Algorithm: spdx_common.ChecksumAlgorithm("SHA1"),
								Value:     "11d7774ac38f40e009dcee453a760750aea75bbd",
							},
						},
					},
				},
				SBOMPackages: []*SBOMPackage{
					{
						Name:          "pkg2",
						SpdxDocument:  "doc-url",
						Algorithm:     spdx_common.ChecksumAlgorithm("SHA1"),
						ChecksumValue: "11d7774ac38f40e009dcee453a760750aea11111",
					},
				},
				ExtractedLicensingInfos: []*spdx2_2.OtherLicense{
					{
						LicenseIdentifier:      "LicenseRef-license1",
						ExtractedText:          "test license",
						LicenseCrossReferences: []string{"license-url"},
					},
				},
			},
			want: &spdx2_2.Document{
				SPDXVersion:       "SPDX-2.2",
				DataLicense:       "CC0-1.0",
				SPDXIdentifier:    spdx_common.ElementID("SPDXRef-DOCUMENT"),
				DocumentName:      "image1-123_sbom.spdx.json",
				DocumentNamespace: "NOASSERTION",
				ExternalDocumentReferences: []spdx2_2.ExternalDocumentRef{
					{
						DocumentRefID: "DocumentRef-pkg2",
						URI:           "doc-url",
						Checksum: spdx_common.Checksum{
							Algorithm: spdx_common.ChecksumAlgorithm("SHA1"),
							Value:     "11d7774ac38f40e009dcee453a760750aea11111",
						},
					},
				},
				CreationInfo: &spdx2_2.CreationInfo{
					Creators: []spdx_common.Creator{
						{
							Creator:     "gcr.io/cos-cloud/cos-customizer",
							CreatorType: "Tool"},
						{
							Creator:     "G",
							CreatorType: "Organization",
						},
					},
					Created: "2023-04-27T16:08:32Z",
				},
				Packages: []*spdx2_2.Package{
					{
						PackageName:           "image1",
						PackageSPDXIdentifier: "SPDXRef-image1",
						PackageVersion:        "123",
						PackageSupplier: &spdx_common.Supplier{
							Supplier:     "G",
							SupplierType: "Organization",
						},
						FilesAnalyzed:           false,
						PackageLicenseDeclared:  "NOASSERTION",
						PackageLicenseConcluded: "NOASSERTION",
						PackageDownloadLocation: "NOASSERTION",
					},
					{
						PackageName:           "pkg1",
						PackageSPDXIdentifier: "SPDXRef-pkg1",
						PackageVersion:        "111",
						PackageSupplier: &spdx_common.Supplier{
							Supplier:     "K",
							SupplierType: "Organization",
						},
						FilesAnalyzed:           false,
						PackageLicenseDeclared:  "BSD-3-Clause AND Apache-2.0",
						PackageLicenseConcluded: "BSD-3-Clause AND Apache-2.0",
						PackageDownloadLocation: "test.com",
						PackageExternalReferences: []*spdx2_2.PackageExternalReference{
							{
								Category: "SECURITY",
								RefType:  "cpe23Type",
								Locator:  "cpe:/a:vendor:pkg-1:1.2.3",
							},
						},
						PackageChecksums: []spdx_common.Checksum{
							{
								Algorithm: spdx_common.ChecksumAlgorithm("SHA1"),
								Value:     "11d7774ac38f40e009dcee453a760750aea75bbd",
							},
						},
					},
				},
				OtherLicenses: []*spdx2_2.OtherLicense{
					{
						LicenseIdentifier:      "LicenseRef-license1",
						ExtractedText:          "test license",
						LicenseCrossReferences: []string{"license-url"},
					},
				},
				Relationships: []*spdx2_2.Relationship{
					{
						RefA:         spdx_common.DocElementID{ElementRefID: spdx_common.ElementID("SPDXRef-DOCUMENT")},
						RefB:         spdx_common.DocElementID{ElementRefID: spdx_common.ElementID("SPDXRef-image1")},
						Relationship: "DESCRIBES",
					},
					{
						RefA:         spdx_common.DocElementID{ElementRefID: spdx_common.ElementID("SPDXRef-image1")},
						RefB:         spdx_common.DocElementID{ElementRefID: spdx_common.ElementID("SPDXRef-pkg1")},
						Relationship: "CONTAINS",
					},
					{
						RefA:         spdx_common.DocElementID{ElementRefID: spdx_common.ElementID("SPDXRef-image1")},
						RefB:         spdx_common.DocElementID{DocumentRefID: "pkg2", ElementRefID: spdx_common.ElementID("SPDXRef-DOCUMENT")},
						Relationship: "CONTAINS",
					},
				},
			},
		},
		{
			testName: "InvalidCreator",
			sbomInput: &SBOMInput{
				OutputImageName:    "image2",
				OutputImageVersion: "123",
				Creators:           []string{"OrganizationG"},
			},
			wantErr: true,
		},
		{
			testName: "InvalidSupplier",
			sbomInput: &SBOMInput{
				OutputImageName:    "image2",
				OutputImageVersion: "123",
				Creators:           []string{"Organization: G"},
				Supplier:           "OrganizationG",
			},
			wantErr: true,
		},
	}

	for _, test := range testData {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			t.Parallel()
			sbom := NewSBOMCreator(nil, nil, nil)
			sbom.sbomInput = test.sbomInput
			srcImage := &config.Image{}
			outImage := &config.Image{}
			if err := sbom.GenerateSBOM(srcImage, outImage); (err != nil) != test.wantErr {
				t.Fatalf("Unexpected error status, want err: %v, got err: %v", test.wantErr, err)
			}
			if !test.wantErr {
				if diff := cmp.Diff(sbom.sbomOutput, test.want, cmp.AllowUnexported(spdx2_2.Package{})); diff != "" {
					t.Fatalf("Mismatch in output SBOM Document. diff: %v", diff)
				}
			}
		})
	}
}

func TestUploadSBOMToGCS(t *testing.T) {
	ctx := context.Background()
	fakeGCS := fakes.GCSForTest(t)
	gcsClient := fakeGCS.Client
	defer fakeGCS.Close()
	sbom := NewSBOMCreator(ctx, gcsClient, nil)
	sbom.sbomOutput = &spdx2_2.Document{
		SPDXVersion:       "SPDX-2.2",
		DataLicense:       "CC0-1.0",
		SPDXIdentifier:    spdx_common.ElementID("SPDXRef-DOCUMENT"),
		DocumentName:      "image1_sbom.json",
		DocumentNamespace: "NOASSERTION",
		CreationInfo: &spdx2_2.CreationInfo{
			Creators: []spdx_common.Creator{
				{
					Creator:     "gcr.io/cos-cloud/cos-customizer",
					CreatorType: "Tool"},
				{
					Creator:     "G",
					CreatorType: "Organization",
				},
			},
			Created: "2023-04-27T16:08:32Z",
		},
	}
	wantSBOM := `{
  "spdxVersion": "SPDX-2.2",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "image1_sbom.json",
  "documentNamespace": "NOASSERTION",
  "creationInfo": {
    "creators": [
      "Tool: gcr.io/cos-cloud/cos-customizer",
      "Organization: G"
    ],
    "created": "2023-04-27T16:08:32Z"
  }
}`
	sbom.UploadSBOMToGCS("gs://test-bucket/folder")
	r, err := gcsClient.Bucket("test-bucket").Object("folder/image1_sbom.json").NewReader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(got), wantSBOM); diff != "" {
		t.Errorf("unexpected SBOM content, diff: %v", diff)
	}
}
