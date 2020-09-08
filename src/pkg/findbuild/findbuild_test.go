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
	"errors"
	"fmt"
	"net/http"
	"testing"

	"go.chromium.org/luci/common/api/gerrit"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	externalGerritURL         string = "https://cos-review.googlesource.com"
	externalGitilesURL        string = "cos.googlesource.com"
	externalFallbackGerritURL string = "https://chromium-review.googlesource.com"
	externalManifestRepo      string = "cos/manifest-snapshots"
	fallbackRepoPrefix        string = "mirrors/cros/"
)

func unwrappedError(err error) error {
	innerErr := err
	for errors.Unwrap(innerErr) != nil {
		innerErr = errors.Unwrap(innerErr)
	}
	return innerErr
}

func getHTTPClient() (*http.Client, error) {
	creds, err := google.FindDefaultCredentials(context.Background(), gerrit.OAuthScope)
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
		FallbackPrefix     string
		ManifestRepo       string
		OutputBuildNum     string
		ShouldFallback     bool
		ShouldError        bool
	}{
		"invalid host": {
			Change:         "3781",
			GerritHost:     "https://zop-review.googlesource.com",
			GitilesHost:    "zop.googlesource.com",
			ManifestRepo:   externalManifestRepo,
			ShouldFallback: false,
			ShouldError:    true,
		},
		"incorrect manifest repo": {
			Change:         "3781",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   "cos/manifest",
			ShouldFallback: false,
			ShouldError:    true,
		},
		"master branch release version": {
			Change:         "3280",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "15085.0.0",
			ShouldFallback: false,
			ShouldError:    false,
		},
		"R85-13310.B branch release version": {
			Change:         "3206",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "13310.1025.0",
			ShouldFallback: false,
			ShouldError:    false,
		},
		"only CL in build diff": {
			Change:         "3781",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "12371.1072.0",
			ShouldFallback: false,
			ShouldError:    false,
		},
		"non-existant CL": {
			Change:             "9999999999",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			ShouldFallback:     true,
			ShouldError:        true,
		},
		"abandoned CL": {
			Change:         "3743",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			ShouldFallback: false,
			ShouldError:    true,
		},
		"under review CL": {
			Change:         "1540",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			ShouldFallback: false,
			ShouldError:    true,
		},
		"chromium CL": {
			Change:             "2288114",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			FallbackPrefix:     fallbackRepoPrefix,
			OutputBuildNum:     "15049.0.0",
			ShouldFallback:     true,
			ShouldError:        false,
		},
		"use commit SHA": {
			Change:             "80809c436f1cae4cde117fce34b82f38bdc2fd36",
			GerritHost:         externalGerritURL,
			GitilesHost:        externalGitilesURL,
			FallbackGerritHost: externalFallbackGerritURL,
			ManifestRepo:       externalManifestRepo,
			OutputBuildNum:     "12871.1183.0",
			ShouldFallback:     false,
			ShouldError:        false,
		},
		"reject cherry-picked change-id": {
			Change:         "I6cc721e6e61b3863e549045e68c1a2bd363efa0a",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			ShouldFallback: false,
			ShouldError:    true,
		},
		"third_party/kernel special branch case": {
			Change:         "3302",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			OutputBuildNum: "15088.0.0",
			ShouldFallback: false,
			ShouldError:    false,
		},
		"branch not in manifest": {
			Change:         "1592",
			GerritHost:     externalGerritURL,
			GitilesHost:    externalGitilesURL,
			ManifestRepo:   externalManifestRepo,
			ShouldFallback: false,
			ShouldError:    true,
		},
	}

	httpClient, _ := getHTTPClient()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := &BuildRequest{
				HTTPClient:   httpClient,
				GerritHost:   test.GerritHost,
				GitilesHost:  test.GitilesHost,
				ManifestRepo: test.ManifestRepo,
				RepoPrefix:   "",
				CL:           test.Change,
			}
			res, err := FindBuild(req)
			innerErr := unwrappedError(err)
			if innerErr != ErrorCLNotFound && test.ShouldFallback {
				t.Fatalf("expected not found error, got %v", err)
			}
			if innerErr == ErrorCLNotFound {
				fallbackReq := &BuildRequest{
					HTTPClient:   httpClient,
					GerritHost:   test.FallbackGerritHost,
					GitilesHost:  test.GitilesHost,
					ManifestRepo: test.ManifestRepo,
					RepoPrefix:   test.FallbackPrefix,
					CL:           test.Change,
				}
				res, err = FindBuild(fallbackReq)
			}
			switch {
			case (err != nil) != test.ShouldError:
				ShouldError := "no error"
				if test.ShouldError {
					ShouldError = "some error"
				}
				t.Fatalf("expected %s, got: %v", ShouldError, err)
			case !test.ShouldError && res.BuildNum != test.OutputBuildNum:
				t.Fatalf("expected output %s, got %s", test.OutputBuildNum, res)
			}
		})
	}
}
