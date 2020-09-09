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

// This package contains the error interface returned by changelog and
// findbuild packages. It includes functions to retrieve HTTP status codes
// from Gerrit and Gitiles errors, and functions to create ChangelogErrors
// relevant to the changelog and findbuild features.

package utils

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/api/gitiles"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	cosGoBURL    = "cos.googlesource.com"
	manifestRepo = "cos/manifest-snapshots"
)

type repo struct {
	// The Git on Borg instance to query from.
	InstanceURL string
	// A value that points to the last commit for a build on a given repo.
	// Acceptable values:
	// - A commit SHA
	// - A ref, ex. "refs/heads/branch"
	// - A ref defined as n-th parent of R in the form "R-n".
	//   ex. "master-2" or "deadbeef-1".
	// Source: https://pkg.go.dev/go.chromium.org/luci/common/proto/gitiles?tab=doc#LogRequest
	Committish string
}

func getHTTPClient() (*http.Client, error) {
	creds, err := google.FindDefaultCredentials(context.Background(), gerrit.OAuthScope)
	if err != nil || len(creds.JSON) == 0 {
		return nil, fmt.Errorf("no application default credentials found - run `gcloud auth application-default login` and try again")
	}
	return oauth2.NewClient(oauth2.NoContext, creds.TokenSource), nil
}

// repoMap generates a mapping of repo name to instance URL and committish.
// This eliminates the need to track remote names and allows lookup
// of source committish when generating changelog.
func repoMap(manifest string) (map[string]*repo, error) {
	if manifest == "" {
		return nil, fmt.Errorf("repoMap: manifest data is empty")
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromString(manifest); err != nil {
		return nil, fmt.Errorf("repoMap: error parsing manifest xml:\n%v", err)
	}
	root := doc.SelectElement("manifest")

	// Parse each <remote fetch=X name=Y> tag in the manifest xml file.
	// Extract the "fetch" and "name" attributes from each remote tag, and map the name to the fetch URL.
	remoteMap := make(map[string]string)
	for _, remote := range root.SelectElements("remote") {
		url := strings.Replace(remote.SelectAttr("fetch").Value, "https://", "", 1)
		remoteMap[remote.SelectAttr("name").Value] = url
	}

	// Parse each <project name=X remote=Y revision=Z> tag in the manifest xml file.
	// Extract the "name", "remote", and "revision" attributes from each project tag.
	// Some projects do not have a "remote" attribute.
	// If this is the case, they should use the default remoteURL.
	if root.SelectElement("default").SelectAttr("remote") != nil {
		remoteMap[""] = remoteMap[root.SelectElement("default").SelectAttr("remote").Value]
	}
	repos := make(map[string]*repo)
	for _, project := range root.SelectElements("project") {
		repos[project.SelectAttr("name").Value] = &repo{
			InstanceURL: remoteMap[project.SelectAttrValue("remote", "")],
			Committish:  project.SelectAttr("revision").Value,
		}
	}
	return repos, nil
}

func TestDownloadManifest(t *testing.T) {
	tests := map[string]struct {
		ManifestRepo string
		BuildNum     string
		ShouldError  bool
	}{
		"master branch": {
			ManifestRepo: manifestRepo,
			BuildNum:     "15000.0.0",
			ShouldError:  false,
		},
		"release branch": {
			ManifestRepo: manifestRepo,
			BuildNum:     "13310.1035.0",
			ShouldError:  false,
		},
		"invalid build": {
			ManifestRepo: manifestRepo,
			BuildNum:     "1.1551226.0",
			ShouldError:  true,
		},
		"invalid manifest repo": {
			ManifestRepo: manifestRepo,
			BuildNum:     "1.1551226.0",
			ShouldError:  true,
		},
	}
	httpClient, _ := getHTTPClient()
	gobClient, _ := gitiles.NewRESTClient(httpClient, cosGoBURL, false)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := DownloadManifest(gobClient, test.ManifestRepo, test.BuildNum)
			if (err != nil) != test.ShouldError {
				ShouldError := "no error"
				if test.ShouldError {
					ShouldError = "some error"
				}
				t.Fatalf("expected %s, got: %v", ShouldError, err)
			}
			if !test.ShouldError {
				_, err = repoMap(resp.Contents)
				if err != nil {
					t.Fatalf("expected parsable manifest file, got %v while attempting to parse response", err)
				}
			}
		})
	}
}

func TestCommits(t *testing.T) {
	tests := map[string]struct {
		Repo        string
		SHA         string
		AncestorSHA string
		QuerySize   int
		ShouldError bool
		MoreCommits bool
	}{
		"no ancestor": {
			Repo:      "cos/cobble",
			SHA:       "a910c096139769e35720174069e81e89bf90fdc6",
			QuerySize: -1,
		},
		"ancestor specified": {
			Repo:        "cos/cobble",
			SHA:         "a910c096139769e35720174069e81e89bf90fdc6",
			AncestorSHA: "30a49d0373138996adcd90f80a5adfba9a342c6d",
			QuerySize:   -1,
		},
		"query size": {
			Repo:        "third_party/kernel",
			SHA:         "f8649a7408c63f53937e33b0e8379679b0434849",
			QuerySize:   15000,
			MoreCommits: true,
		},
		"incorrect repo": {
			Repo:        "not/arepo",
			SHA:         "a910c096139769e35720174069e81e89bf90fdc6",
			QuerySize:   -1,
			ShouldError: true,
		},
		"nonexistant-SHA": {
			Repo:        manifestRepo,
			SHA:         "a910c096139769e35720174069e81e89bddddddd",
			ShouldError: true,
		},
	}
	httpClient, _ := getHTTPClient()
	gobClient, _ := gitiles.NewRESTClient(httpClient, cosGoBURL, false)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			commits, moreCommits, err := Commits(gobClient, test.Repo, test.SHA, test.AncestorSHA, test.QuerySize)
			if (err != nil) != test.ShouldError {
				ShouldError := "no error"
				if test.ShouldError {
					ShouldError = "some error"
				}
				t.Fatalf("expected %s, got: %v", ShouldError, err)
			}
			if !test.ShouldError {
				switch {
				case len(commits) == 0:
					t.Fatalf("expected non-empty commits list, got empty list")
				case test.QuerySize != -1 && len(commits) > test.QuerySize:
					t.Fatalf("expected commits list of at most %d commits, got commits list with %d commits", test.QuerySize, len(commits))
				case moreCommits != test.MoreCommits:
					t.Fatalf("expected moreCommits = %v, got moreCommits = %v", test.MoreCommits, moreCommits)
				case commits[0].Id != test.SHA:
					t.Fatalf("expected commits list to start with commit %s, got %s as starting commit", test.SHA, commits[0].Id)
				}
			}
		})
	}
}
