// Copyright 2020 Google Inc. All Rights Reserved.
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

package findbuild

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	externalGerritURL         string = "https://cos-review.googlesource.com"
	externalGitilesURL        string = "cos.googlesource.com"
	externalFallbackGerritURL string = "https://chromium-review.googlesource.com"
	externalManifestRepo      string = "cos/manifest-snapshots"
)

func getHTTPClient() (*http.Client, error) {
	creds, err := google.FindDefaultCredentials(context.Background(), "https://www.googleapis.com/auth/gerritcodereview")
	if err != nil || len(creds.JSON) == 0 {
		return nil, fmt.Errorf("no application default credentials found - run `gcloud auth application-default login` and try again")
	}
	return oauth2.NewClient(oauth2.NoContext, creds.TokenSource), nil
}

func TestFindCL(t *testing.T) {
	tests := map[string]struct {
		Change             string
		GerritHost         string
		GitilesHost        string
		FallbackGerritHost string
		ManifestRepo       string
		OutputBuildNum     string
		ShouldFallback     bool
		ExpectedError      string
	}{
		"exponential search range required": {
			Change:             "1740206",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			OutputBuildNum:     "12371.1001.0",
			ShouldFallback:     true,
		},
		"invalid host": {
			Change:         "3781",
			GerritHost:     "https://zop-review.googlesource.com",
			GitilesHost:    "zop.googlesource.com",
			ManifestRepo:   externalManifestRepo,
			ShouldFallback: false,
			ExpectedError:  "500",
		},
		"no release match in repo": {
			Change:         "3781",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   "not/arepo",
			ShouldFallback: false,
			ExpectedError:  "406",
		},
		"master branch release version": {
			Change:         "3280",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "15085.0.0",
			ShouldFallback: false,
		},
		"R85-13310.B branch release version": {
			Change:         "3206",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "13310.1025.0",
			ShouldFallback: false,
		},
		"only CL in build diff": {
			Change:         "3781",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "12371.1072.0",
			ShouldFallback: false,
		},
		"non-existant CL": {
			Change:             "9",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			ShouldFallback:     true,
			ExpectedError:      "404",
		},
		"abandoned CL": {
			Change:         "3743",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			ShouldFallback: false,
			ExpectedError:  "406",
		},
		"chromium CL": {
			Change:             "2288114",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			OutputBuildNum:     "15049.0.0",
			ShouldFallback:     true,
		},
		"use commit SHA": {
			Change:             "80809c436f1cae4cde117fce34b82f38bdc2fd36",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			OutputBuildNum:     "12871.1183.0",
			ShouldFallback:     false,
		},
		"third_party/kernel special branch case": {
			Change:         "3302",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "15088.0.0",
			ShouldFallback: false,
		},
		"third_party/kernel release branch": {
			Change:         "4983",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "12871.1192.0",
			ShouldFallback: false,
		},
		"branch not in manifest": {
			Change:             "1592",
			GerritHost:         externalGerritURL,
			FallbackGerritHost: externalFallbackGerritURL,
			GitilesHost:        externalGitilesURL,
			ManifestRepo:       externalManifestRepo,
			ExpectedError:      "406",
		},
		"chromeos branch": {
			Change:             "2036225",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			OutputBuildNum:     "12371.1001.0",
			ShouldFallback:     true,
		},
		"main branch": {
			Change:             "808c2270202fbd79367c4c46b6223e6dfc2d1d01",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			OutputBuildNum:     "16101.0.0",
		},
	}

	httpClient, _ := getHTTPClient()
	for name, test := range tests {
		req := &BuildRequest{
			HTTPClient:   httpClient,
			GerritHost:   test.GerritHost,
			GitilesHost:  test.GitilesHost,
			ManifestRepo: test.ManifestRepo,
			CL:           test.Change,
		}
		res, err := FindBuild(req)
		if err != nil && err.HTTPCode() != "404" && test.ShouldFallback {
			t.Fatalf("test \"%s\" failed:\nexpected not found error, got %v", name, err)
		}
		if err != nil && err.HTTPCode() == "404" {
			fallbackReq := &BuildRequest{
				HTTPClient:   httpClient,
				GerritHost:   test.FallbackGerritHost,
				GitilesHost:  test.GitilesHost,
				ManifestRepo: test.ManifestRepo,
				CL:           test.Change,
			}
			res, err = FindBuild(fallbackReq)
		}
		switch {
		case test.ExpectedError == "" && err != nil:
			t.Fatalf("test \"%s\" failed:\nexpected no error, got %v", name, err)
		case test.ExpectedError != "" && err == nil:
			t.Fatalf("test \"%s\" failed:\nexpected error code %s, got nil err", name, test.ExpectedError)
		case test.ExpectedError != "" && err != nil && test.ExpectedError != err.HTTPCode():
			t.Fatalf("test \"%s\" failed:\nexpected error code %s, got error code %s", name, test.ExpectedError, err.HTTPCode())
		case test.ExpectedError == "" && res.BuildNum != test.OutputBuildNum:
			t.Fatalf("test \"%s\" failed:\nexpected output %s, got %s", name, test.OutputBuildNum, res)
		}
		time.Sleep(time.Second * 5)
	}
}
