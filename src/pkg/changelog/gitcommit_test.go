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
	"reflect"
	"testing"
	"time"

	"go.chromium.org/luci/common/proto/git"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	id            string = "4f07bfb8463cb54227cc4cdbffc5d295edc05631"
	tree          string = "c005de2ade3f30506e7899b5c95d17f2904a598f"
	parent        string = "7645df3136c5b5e43eb1af182b0c67d78ca2d517"
	authorName    string = "Austin Yuan"
	committerName string = "Boston Yuan"
)

var authorTime time.Time
var committerTime time.Time

func init() {
	authorTime = time.Date(2019, 6, 6, 10, 50, 0, 0, time.UTC)
	committerTime = time.Date(2020, 2, 1, 8, 15, 0, 0, time.UTC)
}

func createCommitWithMessage(message string) *git.Commit {
	return &git.Commit{
		Id:        id,
		Tree:      tree,
		Parents:   []string{parent},
		Author:    &git.Commit_User{Name: authorName, Email: "austinyuan@google.com", Time: timestamppb.New(authorTime)},
		Committer: &git.Commit_User{Name: committerName, Email: "bostonyuan@google.com", Time: timestamppb.New(committerTime)},
		Message:   message,
	}
}

func TestParseGitCommit(t *testing.T) {
	tests := map[string]struct {
		Input         *git.Commit
		SHA           string
		AuthorName    string
		CommitterName string
		Subject       string
		Bugs          []string
		ShouldError   bool
	}{
		"basic": {
			Input: createCommitWithMessage(`provision_AutoUpdate: Do not stage AU payloads unless necessary

Currently provisionning always stages the AU full payloads at the
beginning. But majority of provision runs succeed by quick-provisioning
and they never get to AU provisioning. So this is a waste of time and
space trying to stage large files that are not going to be used. This CL
fixes that problem.

BUG=chromium:1097995
TEST=test_that --args="value='reef-release/R85-13280.0.0'" chromeos6-row4-rack10-host19.cros.corp.google.com provision_AutoUpdate
TEST=same as above, but changed the code to skip the quick-provisioning

Change-Id: I0b6895f7860921f6bed25090d64f8489dbeeb19e
Reviewed-on: https://chromium-review.googlesource.com/c/chromiumos/third_party/autotest/+/2268290
Tested-by: Amin Hassani <ahassani@chromium.org>
Commit-Queue: Amin Hassani <ahassani@chromium.org>
Reviewed-by: Allen Li <ayatane@chromium.org>
Auto-Submit: Amin Hassani <ahassani@chromium.org>`),
			SHA:           id,
			AuthorName:    authorName,
			CommitterName: committerName,
			Subject:       "provision_AutoUpdate: Do not stage AU payloads unless necessary",
			Bugs:          []string{"crbug/1097995"},
			ShouldError:   false,
		},
		"multiple bugs": {
			Input: createCommitWithMessage(`autotest: Move host dependency check inside verifier.

Moved it to minimize fail case if host is not available.

BUG=chromium:1069101, chromium:1059439, b:533302,b/21114011,chromium-os:993221,chrome-os-partner:3341233
TEST=unittests, presubmit, run local`),
			SHA:           id,
			AuthorName:    authorName,
			CommitterName: committerName,
			Subject:       "autotest: Move host dependency check inside verifier.",
			Bugs:          []string{"crbug/1069101", "crbug/1059439", "b/533302", "b/21114011", "crbug/993221", "crbug/3341233"},
			ShouldError:   false,
		},
		"improperly formatted bugs": {
			Input: createCommitWithMessage(`chrome-os-partner:1224444

b/3225555

BUG=54985123, z, c/54811233, notabug, 0, b%21333443, -3, hello b/12321155, 
TEST=b/2222222`),
			SHA:           id,
			AuthorName:    authorName,
			CommitterName: committerName,
			Subject:       "chrome-os-partner:1224444",
			Bugs:          []string{},
			ShouldError:   false,
		},
		"proper and improper bugs": {
			Input: createCommitWithMessage(`autotest: Move host dependency check inside verifier.

Some extra details here

BUG=3, -1, b:2212344, c/54811233, chrome-os-partner:1111111, notabug, b%21333443, test b:6644322
TEST=unittests, presubmit, run local`),
			SHA:           id,
			AuthorName:    authorName,
			CommitterName: committerName,
			Subject:       "autotest: Move host dependency check inside verifier.",
			Bugs:          []string{"b/2212344", "crbug/1111111"},
			ShouldError:   false,
		},
		"empty commit message": {
			Input:         createCommitWithMessage(""),
			SHA:           id,
			AuthorName:    authorName,
			CommitterName: committerName,
			Subject:       "",
			Bugs:          []string{},
			ShouldError:   false,
		},
		"only subject line": {
			Input:         createCommitWithMessage("$()!-1"),
			SHA:           id,
			AuthorName:    authorName,
			CommitterName: committerName,
			Subject:       "$()!-1",
			Bugs:          []string{},
			ShouldError:   false,
		},
		"no bug line": {
			Input: createCommitWithMessage(`autotest: Move host dependency check inside verifier.

Moved it to minimize fail case if host is not available.
AdminAudit is starting with the set of actions and all of them has to
run if possible. By this move we allowed each verifier to check what
required to run.
If dependency not provided we can skip of the action.

TEST=unittests, presubmit, run local`),
			SHA:           id,
			AuthorName:    authorName,
			CommitterName: committerName,
			Subject:       "autotest: Move host dependency check inside verifier.",
			Bugs:          []string{},
			ShouldError:   false,
		},
		"missing fields": {
			Input: &git.Commit{
				Id:        id,
				Tree:      "",
				Parents:   nil,
				Author:    nil,
				Committer: nil,
				Message:   "",
			},
			SHA:           id,
			AuthorName:    "None",
			CommitterName: "None",
			Subject:       "",
			Bugs:          []string{},
			ShouldError:   false,
		},
		"nil input": {
			Input:       nil,
			ShouldError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res, err := ParseGitCommit(test.Input)
			switch {
			case (err != nil) != test.ShouldError:
				ShouldError := "no error"
				if test.ShouldError {
					ShouldError = "some error"
				}
				t.Fatalf("expected %v, got: %v\n", ShouldError, err)
			case test.ShouldError && (err != nil) == test.ShouldError && res == nil:
			case res.SHA != test.SHA:
				t.Errorf("expected commitSHA %s, got %s", test.SHA, res.SHA)
			case res.AuthorName != test.AuthorName:
				t.Errorf("expected authorName: %s, got: %s", test.AuthorName, res.AuthorName)
			case res.CommitterName != test.CommitterName:
				t.Errorf("expected committerName %s, got %s", test.CommitterName, res.CommitterName)
			case res.Subject != test.Subject:
				t.Errorf("expected subject %s, got %s", test.Subject, res.Subject)
			case !reflect.DeepEqual(res.Bugs, test.Bugs):
				t.Errorf("exptected bugs %#v, got %#v", test.Bugs, res.Bugs)
			}
		})
	}
}

func TestParseGitCommitLog(t *testing.T) {
	tests := map[string]struct {
		Input          []*git.Commit
		SHAs           []string
		AuthorNames    []string
		CommitterNames []string
		Subjects       []string
		Bugs           [][]string
		ShouldError    bool
	}{
		"multiple commits": {
			Input: []*git.Commit{
				createCommitWithMessage(`This is a subject

This commit has no bugs

TEST=unittests, presubmit, run local`),
				createCommitWithMessage(`autotest: Some subject

This commit some bugs

BUG=b/4332134, chrome-os-partner:0999212, b:11111
TEST=unittests, presubmit, run local

Change-Id: I0b6895f7860921f6bed25090d64f8489dbeeb19e`),
				createCommitWithMessage(`Third

This commits has some multiple bugs, some not valid b/1221212

BUG=56456651, chromium:777882, -1, b:9999999`),
			},
			SHAs:           []string{id, id, id},
			AuthorNames:    []string{authorName, authorName, authorName},
			CommitterNames: []string{committerName, committerName, committerName},
			Subjects:       []string{"This is a subject", "autotest: Some subject", "Third"},
			Bugs:           [][]string{{}, {"b/4332134", "crbug/0999212", "b/11111"}, {"crbug/777882", "b/9999999"}},
			ShouldError:    false,
		},
		"empty list": {
			Input:          []*git.Commit{},
			SHAs:           []string{},
			AuthorNames:    []string{},
			CommitterNames: []string{},
			Subjects:       []string{},
			Bugs:           [][]string{},
			ShouldError:    false,
		},
		"nil input": {
			Input:       nil,
			ShouldError: true,
		},
		"normal and nil input": {
			Input: []*git.Commit{
				createCommitWithMessage(`This is a subject

This commit has no bugs

TEST=unittests, presubmit, run local`),
				createCommitWithMessage(`autotest: Some subject

This commit some bugs

BUG=b/4332134, chrome-os-partner:0999212, b:11111
TEST=unittests, presubmit, run local

Change-Id: I0b6895f7860921f6bed25090d64f8489dbeeb19e`),
				nil,
				createCommitWithMessage(`Third

This commits has some multiple bugs, some not valid b/1221212

BUG=56456651, chromium:777882, -1, b:9999999`),
			},
			ShouldError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res, err := ParseGitCommitLog(test.Input)
			switch {
			case (err != nil) != test.ShouldError:
				ShouldError := "no error"
				if test.ShouldError {
					ShouldError = "some error"
				}
				t.Fatalf("expected %v, got: %v\n", ShouldError, err)
			case test.ShouldError && (err != nil) == test.ShouldError && res == nil:
			}
			for i, commit := range res {
				switch {
				case commit.SHA != test.SHAs[i]:
					t.Errorf("expected commitSHA %s, got %s", test.SHAs[i], commit.SHA)
				case commit.AuthorName != test.AuthorNames[i]:
					t.Errorf("expected authorName: %s, got: %s", test.AuthorNames[i], commit.AuthorName)
				case commit.CommitterName != test.CommitterNames[i]:
					t.Errorf("expected committerName %s, got %s", test.CommitterNames[i], commit.CommitterName)
				case commit.Subject != test.Subjects[i]:
					t.Errorf("expected subject %s, got %s", test.Subjects[i], commit.Subject)
				case !reflect.DeepEqual(commit.Bugs, test.Bugs[i]):
					t.Errorf("exptected bugs %#v, got %#v", test.Bugs[i], commit.Bugs)
				}
			}
		})
	}
}
