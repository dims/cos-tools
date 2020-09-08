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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

const (
	gerritURL      = "https://cos-review.googlesource.com"
	fallbackURLs   = "https://chromium-review.googlesource.com"
	gitilesURL     = "cos.googlesource.com"
	manifestRepo   = "cos/manifest-snapshots"
	fallbackPrefix = "mirrors/cros/"
)

var (
	repoLogFields = []string{"Commits", "InstanceURL", "Repo", "SourceSHA", "TargetSHA", "HasMoreCommits"}
	commitFields  = []string{"SHA", "AuthorName", "CommitterName", "Subject", "Bugs", "ReleaseNote", "CommitTime"}
)

type Commit struct {
	SHA           string
	AuthorName    string
	CommitterName string
	Subject       string
	Bugs          []string
	CommitTime    string
	ReleaseNote   string
}

func setup() error {
	cmd := exec.Command("go", "build", "-o", "changelog-cli", "main.go")
	return cmd.Run()
}

func cleanOutputFiles(source, target string) {
	additions := filename(source, target)
	removals := filename(target, source)
	cmd := exec.Command("rm", additions, removals)
	cmd.Run()
}

func filename(source, target string) string {
	return fmt.Sprintf("%s -> %s.json", source, target)
}

func fileExists(source, target string) bool {
	fname := filename(source, target)
	info, err := os.Stat(fname)
	if os.IsNotExist(err) || info.IsDir() {
		return false
	}
	return true
}

func fileContents(source, target string) []byte {
	fname := filename(source, target)
	contents, _ := ioutil.ReadFile(fname)
	return contents
}

func validateEmptyChangelog(source, target string) bool {
	contents := fileContents(source, target)
	return string(contents) == "{}"
}

// validateCommit verifies if a given interface matches the commit format
func validateCommit(input interface{}) bool {
	commit, ok := input.(map[string]interface{})
	if !ok {
		return false
	}
	for _, field := range commitFields {
		if _, ok := commit[field]; !ok {
			return false
		}
	}
	return true
}

func validateRepoLog(input interface{}) bool {
	repoLog, ok := input.(map[string]interface{})
	if !ok {
		return false
	}
	for _, field := range repoLogFields {
		if _, ok := repoLog[field]; !ok {
			return false
		}
	}
	commits, ok := repoLog["Commits"]
	if !ok {
		return false
	}
	if _, ok := commits.([]interface{}); !ok {
		return false
	}
	for _, commit := range commits.([]interface{}) {
		if !validateCommit(commit) {
			return false
		}
	}
	return true
}

func validateChangelogSchema(source, target string) bool {
	if validateEmptyChangelog(source, target) {
		return false
	}
	contents := fileContents(source, target)
	var data interface{}
	err := json.Unmarshal(contents, &data)
	if err != nil {
		return false
	}
	if _, ok := data.(map[string]interface{}); !ok {
		return false
	}
	for _, val := range data.(map[string]interface{}) {
		if !validateRepoLog(val) {
			return false
		}
	}
	return true
}

func TestChangelog(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatalf("Error compiling main.go:\n%v", err)
	}

	tests := map[string]struct {
		Source    string
		Target    string
		Args      []string
		ShouldErr bool
		EmptyAdds bool
		EmptyRms  bool
	}{
		"basic run": {
			Source:    "15050.0.0",
			Target:    "15056.0.0",
			ShouldErr: false,
			EmptyAdds: false,
			EmptyRms:  true,
		},
		"with instance and repo": {
			Source:    "15048.0.0",
			Target:    "15049.0.0",
			Args:      []string{"-gob", gitilesURL, "-r", manifestRepo},
			ShouldErr: false,
			EmptyAdds: false,
			EmptyRms:  true,
		},
		"image name": {
			Source:    "cos-rc-85-13310-1034-0",
			Target:    "cos-rc-85-13310-1030-0",
			ShouldErr: false,
			EmptyAdds: true,
			EmptyRms:  false,
		},
		"invalid source": {
			Source:    "999999.0.0",
			Target:    "15056.0.0",
			ShouldErr: true,
		},
		"invalid target": {
			Source:    "15056.0.0",
			Target:    "99999.0.0",
			ShouldErr: true,
		},
		"invalid instance": {
			Source:    "999999.0.0",
			Target:    "15056.0.0",
			Args:      []string{"-gob", "cos.gg.com", "-repo", manifestRepo},
			ShouldErr: true,
		},
		"invalid repo": {
			Source:    "999999.0.0",
			Target:    "15056.0.0",
			Args:      []string{"-gob", gitilesURL, "-repo", "not/arepo"},
			ShouldErr: true,
		},
		"same source and target": {
			Source:    "15049.0.0",
			Target:    "15049.0.0",
			Args:      []string{"-gob", gitilesURL, "-r", manifestRepo},
			ShouldErr: false,
			EmptyAdds: true,
			EmptyRms:  true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"-mode", "changelog"}, test.Args...)
			args = append(args, []string{test.Source, test.Target}...)
			cmd := exec.Command("./changelog-cli", args...)
			err := cmd.Run()
			if test.ShouldErr {
				switch {
				case err == nil:
					t.Fatalf("expected error, got nil")
				case fileExists(test.Source, test.Target):
					t.Fatalf("expected no files to be created, got additions file")
				case fileExists(test.Target, test.Source):
					t.Fatalf("expected no files to be created, got removals file")
				}
			} else {
				switch {
				case err != nil:
					t.Fatalf("expected no error, got %v", err)
				case !fileExists(test.Source, test.Target):
					t.Fatalf("expected additions file to be created, got no file")
				case !fileExists(test.Target, test.Source):
					t.Fatalf("expected removals file to be created, got no file")
				case test.EmptyAdds && !validateEmptyChangelog(test.Source, test.Target):
					t.Fatalf("expected empty additions file, got non-empty file")
				case !test.EmptyAdds && !validateChangelogSchema(test.Source, test.Target):
					t.Fatalf("expected valid, nonempty additions file, got invalid/empty file")
				case test.EmptyRms && !validateEmptyChangelog(test.Target, test.Source):
					t.Fatalf("expected empty removals file, got non-empty file")
				case !test.EmptyRms && !validateChangelogSchema(test.Target, test.Source):
					t.Fatalf("expected valid, nonempty removals file, got invalid/empty file")
				}
				cleanOutputFiles(test.Source, test.Target)
			}
		})
	}
}

func TestFindBuild(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatalf("Error compiling main.go:\n%v", err)
	}

	tests := map[string]struct {
		CL        string
		Args      []string
		Output    string
		ShouldErr bool
	}{
		"test basic": {
			CL:     "3781",
			Output: "Build: 12371.1072.0\n",
		},
		"test commit SHA": {
			CL:     "80809c436f1cae4cde117fce34b82f38bdc2fd36",
			Output: "Build: 12871.1183.0\n",
		},
		"test gerrit fallback": {
			CL:     "2288114",
			Output: "Build: 15049.0.0\n",
		},
		"test string flags": {
			CL:     "3781",
			Args:   []string{"-gerrit", gerritURL, "-gob", gitilesURL, "-repo", manifestRepo},
			Output: "Build: 12371.1072.0\n",
		},
		"test fallback string flags": {
			CL:     "2288114",
			Args:   []string{"-gerrit", gerritURL, "-fallback", fallbackURLs, "-gob", gitilesURL, "-repo", manifestRepo, "-prefix", fallbackPrefix},
			Output: "Build: 15049.0.0\n",
		},
		"invalid gob": {
			CL:        "2288114",
			Args:      []string{"-gerrit", gerritURL, "-fallback", fallbackURLs, "-gob", "zop.googlesource.com", "-repo", manifestRepo},
			ShouldErr: true,
		},
		"invalid gerrit": {
			CL:        "3781",
			Args:      []string{"-gerrit", "https://zop-review.googlesource.com", "-fallback", fallbackURLs, "-gob", gitilesURL, "-repo", manifestRepo},
			ShouldErr: true,
		},
		"invalid fallback": {
			CL:        "2288114",
			Args:      []string{"-gerrit", gerritURL, "-fallback", "https://zop-review.googlesource.com", "-gob", gitilesURL, "-repo", manifestRepo},
			ShouldErr: true,
		},
		"invalid prefix": {
			CL:        "2288114",
			Args:      []string{"-gerrit", gerritURL, "-fallback", fallbackURLs, "-gob", gitilesURL, "-repo", manifestRepo, "-prefix", "mirrors/zop"},
			ShouldErr: true,
		},
		"non-existant cl": {
			CL:        "9999999999999999999999",
			ShouldErr: true,
		},
		"unsubmitted cl": {
			CL:        "1540",
			ShouldErr: true,
		},
		"invalid cl identifier": {
			CL:        "I6cc721e6e61b3863e549045e68c1a2bd363efa0a",
			ShouldErr: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			args := append([]string{"-mode", "findbuild"}, test.Args...)
			args = append(args, test.CL)
			cmd := exec.Command("./changelog-cli", args...)
			cmd.Stdout = &out
			err := cmd.Run()
			if test.ShouldErr && err == nil {
				t.Fatalf("expected error, got nil")
			} else if !test.ShouldErr {
				switch {
				case err != nil:
					t.Fatalf("expected no error, got %v", err)
				case out.String() != test.Output:
					t.Fatalf("expected output %s, got %s", test.Output, out.String())
				}
			}
		})
	}
}
