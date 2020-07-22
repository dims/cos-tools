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

package changelog

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"go.chromium.org/luci/common/proto/git"
)

const bugLinePrefix string = "BUG="
const releaseNoteLinePrefix string = "RELEASE_NOTE="

// Commit is a simplified struct of git.Commit
// Useful for interfaces
type Commit struct {
	SHA           string
	AuthorName    string
	CommitterName string
	Subject       string
	Bugs          []string
	ReleaseNote   string
	CommitTime    string
}

// All bug patterns need to be added here to recognize whether a bug entry
// should be ignored or not
var bugPatternToReplacement = map[*regexp.Regexp]string{
	regexp.MustCompile("^b/"):          "b/",
	regexp.MustCompile("^b:"):          "b/",
	regexp.MustCompile("^chromium.*:"): "crbug/",
	regexp.MustCompile("^chrome.*:"):   "crbug/",
}

func author(commit *git.Commit) string {
	if commit.Author != nil {
		return commit.Author.Name
	}
	return "None"
}

func committer(commit *git.Commit) string {
	if commit.Committer != nil {
		return commit.Committer.Name
	}
	return "None"
}

func subject(commit *git.Commit) string {
	return strings.Split(commit.Message, "\n")[0]
}

func bugs(commit *git.Commit) []string {
	output := []string{}
	msgSplit := strings.Split(commit.Message, "\n")
	bugLine := ""
	for _, line := range msgSplit {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, bugLinePrefix) {
			bugLine = line
			break
		}
	}
	if len(bugLine) <= len(bugLinePrefix) {
		return output
	}
	bugList := strings.Split(bugLine[len(bugLinePrefix):], ",")
	for _, bug := range bugList {
		bug := strings.TrimSpace(bug)
		for prefix, replacement := range bugPatternToReplacement {
			if match := prefix.FindString(bug); match != "" {
				output = append(output, replacement+bug[len(match):])
			}
		}
	}
	return output
}

func releaseNote(commit *git.Commit) string {
	msgSplit := strings.Split(commit.Message, "\n")
	for _, line := range msgSplit {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, releaseNoteLinePrefix) {
			return line[len(releaseNoteLinePrefix):]
		}
	}
	return ""
}

func commitTime(commit *git.Commit) string {
	if commit.Committer != nil {
		return commit.Committer.Time.AsTime().Format(time.RFC1123)
	}
	return "None"
}

// ParseGitCommit converts a git.Commit object into a
// Commit object with processed fields
func parseGitCommit(commit *git.Commit) (*Commit, error) {
	if commit == nil {
		return nil, errors.New("ParseCommit: Input should not be nil")
	}
	return &Commit{
		SHA:           commit.Id,
		AuthorName:    author(commit),
		CommitterName: committer(commit),
		Subject:       subject(commit),
		Bugs:          bugs(commit),
		ReleaseNote:   releaseNote(commit),
		CommitTime:    commitTime(commit),
	}, nil
}

// ParseGitCommitLog converts a slice of git.Commit objects
// into a slice of Commit objects with processed fields
func ParseGitCommitLog(commits []*git.Commit) ([]*Commit, error) {
	if commits == nil {
		return nil, errors.New("ParseCommitLog: Input should not be nil")
	}
	output := make([]*Commit, len(commits))
	for i, commit := range commits {
		parsedCommit, err := parseGitCommit(commit)
		if err != nil {
			return nil, errors.New("ParseCommitLog: Input slice contains nil pointer")
		}
		output[i] = parsedCommit
	}
	return output, nil
}
