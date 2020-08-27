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

// This package returns the build number of the first build containing
// a specified CL. It accepts any value that identifies a unique CL,
// such as a CL Number or Commit SHA.
//
// This package uses information from Gerrit and Git on Borg to complete a
// request. To locate the first build containing a CL, the package first
// queries Gerrit for the commit SHA, repository, release branch, and
// submission time of the user-provided CL. It then retrieves a list of
// commits that were submitted in the manifest repository under the same
// release branch. It narrows down the commit list to all commits that were
// made within 5 days of the CL submission, and downloads/parses each manifest
// file created within this time range concurrently. Each thread retrieves
// the commit SHA associated with the CL's repository and branch in the
// manifest file, and maps it to the manifest file's build number. It creates
// a repository changelog between the first and last commit SHA in the window,
// and traverses the changelog until it encounters the target CL. It then
// continues traversing until it encounters a commit SHA that exists in the
// build mapping. This is the first build containing the CL, and is returned.

package findbuild

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cos.googlesource.com/cos/tools/src/pkg/utils"
	"github.com/beevik/etree"
	log "github.com/sirupsen/logrus"
	"go.chromium.org/luci/common/api/gerrit"
	gitilesApi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/proto/git"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

const (
	// Number of days after CL submission to search for CL landing
	endBuildTime = 5
	// Maximum time to wait for a response from a Gerrit or Gitiles request
	requestMaxAge = 30 * time.Second
)

var (
	// clReleaseMapping is used to handle special cases where a CL's branch
	// name does not map to a branch in the manifest repository.
	clReleaseMapping = map[string]struct {
		// releaseRe is the regexp used to retrieve the release branch name
		// from a CL branch name
		releaseRe *regexp.Regexp
		// defaultRelease is the release branch that will be used if there is
		// no regexp match
		defaultRelease string
	}{
		"third_party/kernel": {
			releaseRe:      regexp.MustCompile("(.*)-cos-.*"),
			defaultRelease: "master",
		},
	}
)

// BuildRequest is the input struct for the FindBuild function
type BuildRequest struct {
	// HttpClient is a authorized http.Client object with Gerrit scope.
	HTTPClient *http.Client
	// GerritHost is the Gerrit instance to query from.
	// ex. "https://cos-review.googlesource.com"
	GerritHost string
	// GitilesHost is the GoB instance to query from.
	// It should contain the manifest repository
	// ex. "cos.googlesource.com"  (note the lack of https://)
	GitilesHost string
	// ManifestRepo is the repository the manifest.xml files are located in.
	// ex. "cos/manifest-snapshots"
	ManifestRepo string
	// RepoPrefix is the prefix that will be added to the repository name
	// when querying GoB using the repository listed in a Gerrit CL.
	// For example, if a CL in Gerrit is for a change in the
	// "chromiumos/overlays/chromiumos-overlay" repository, specifying a prefix
	// of "mirrors/cros" will instruct this package to search in the repository
	// "mirrors/cros/chromiumos/overlays/chromiumos-overlay". This is useful if
	// your GerritHost and GitilesHost are not a part of the same instance.
	RepoPrefix string
	// CL can be either the CL number or commit SHA of your target CL
	// ex. 3741 or If9f774179322c413fa0fd5ebb3dd615c5b22cd6c
	CL string
}

// BuildResponse is the output struct for the FindBuild function
type BuildResponse struct {
	BuildNum string
	CLNum    string
}

type clData struct {
	CLNum    string
	Project  string
	Release  string
	Branch   string
	Revision string
	Time     time.Time
}

type repoData struct {
	Candidates map[string]string
	SourceSHA  string
	TargetSHA  string
	RemoteURL  string
}

type manifestResponse struct {
	BuildNum  string
	SHA       string
	RemoteURL string
	Err       error
}

// queryCL retrieves the list of CLs matching a query from Gerrit
func queryCL(client *gerrit.Client, clID string) (*gerrit.Change, utils.ChangelogError) {
	log.Debug("Retrieving CL List from Gerrit")
	queryOptions := gerrit.ChangeQueryParams{
		Query:   clID,
		N:       1,
		Options: []string{"CURRENT_REVISION"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestMaxAge)
	defer cancel()
	clList, _, err := client.ChangeQuery(ctx, queryOptions)
	if err != nil {
		log.Errorf("queryCL: Error retrieving change for input %s:\n%v", clID, err)
		httpCode := utils.GerritErrCode(err)
		return nil, utils.FromFindBuildError(httpCode, clID)
	}
	if len(clList) == 0 {
		log.Errorf("queryCL: CL with identifier %s not found", clID)
		return nil, utils.FromFindBuildError("404", clID)
	}
	change := clList[0]
	if change.ChangeID == clID {
		log.Debugf("Provided CL identifier %s is a Change-ID, should be CL num or commit SHA", clID)
		return nil, utils.FromFindBuildError("400", clID)
	}
	if change.Submitted == "" {
		log.Debugf("Provided CL identifier %s maps to an unsubmitted CL", clID)
		return nil, utils.FromFindBuildError("406", clID)
	}
	return change, nil
}

func getCLData(client *gerrit.Client, clID string, manifestPrefix string) (*clData, utils.ChangelogError) {
	log.Debugf("Retrieving CL data from Gerrit for changeID: %s", clID)
	change, err := queryCL(client, clID)
	if err != nil {
		return nil, err
	}
	log.Debugf("Target CL found with SHA %s on repo %s, branch %s", change.CurrentRevision, change.Project, change.Branch)
	parsedTime, parseErr := time.Parse("2006-01-02 15:04:05.000000000", change.Submitted)
	if parseErr != nil {
		log.Errorf("getTargetData: Error parsing submission time %s for CL %d:\n%v", change.Submitted, change.ChangeNumber, err)
		return nil, utils.InternalError
	}
	// If a repository has non-conventional branch names, need to convert the
	// repository branch name to a release branch name
	release := change.Branch
	if rule, ok := clReleaseMapping[change.Project]; ok {
		release = rule.defaultRelease
		matches := rule.releaseRe.FindStringSubmatch(release)
		if len(matches) == 2 {
			release = matches[1]
		}
	}
	return &clData{
		CLNum:    strconv.Itoa(change.ChangeNumber),
		Project:  manifestPrefix + change.Project,
		Release:  release,
		Branch:   change.Branch,
		Revision: change.CurrentRevision,
		Time:     parsedTime,
	}, nil
}

// candidateManifestCommits returns a list of commits to the manifest-snapshot
// repo that were committed within a time range from the target commit time, in
// reverse chronological order.
func candidateManifestCommits(client gitilesProto.GitilesClient, manifestRepo string, targetData *clData) ([]*git.Commit, utils.ChangelogError) {
	log.Debugf("Retrieving all manifest snapshots committed within %d days of CL submission", endBuildTime)
	allManifests, _, err := utils.Commits(client, manifestRepo, "refs/heads/"+targetData.Release, "", -1)
	if err != nil {
		httpCode := utils.GitilesErrCode(err)
		return nil, utils.FromFindBuildError(httpCode, targetData.CLNum)
	}
	// Find latest commit that occurs before the target commit time.
	// allManifests is in reverse chronological order.
	left, right := 0, len(allManifests)-1
	for left < right {
		mid := (left + right) / 2
		if allManifests[mid].Committer == nil {
			log.Errorf("Manifest %s has no committer", allManifests[mid].Id)
			return nil, utils.InternalError
		}
		currDate := allManifests[mid].Committer.Time.AsTime()
		if currDate.Before(targetData.Time) {
			right = mid
		} else {
			left = mid + 1
		}
	}
	earliestIdx := left
	// Find the earliest commit that occurs one week after the target commit time.
	// Don't have to reset right, since anything to the right is earlier.
	left = 0
	endDate := targetData.Time.AddDate(0, 0, endBuildTime)
	for left < right {
		mid := (left+right)/2 + 1
		if allManifests[mid].Committer == nil {
			log.Errorf("Manifest %s has no committer", allManifests[mid].Id)
			return nil, utils.InternalError
		}
		currDate := allManifests[mid].Committer.Time.AsTime()
		if currDate.After(endDate) {
			left = mid
		} else {
			right = mid - 1
		}
	}
	latestIdx := right
	return allManifests[latestIdx : earliestIdx+1], nil
}

// repoTags retrieves all tags belonging to a repository
func repoTags(client gitilesProto.GitilesClient, repo string) (*gitilesProto.RefsResponse, error) {
	log.Debugf("Retrieving tags for repository %s", repo)
	request := &gitilesProto.RefsRequest{
		Project:  repo,
		RefsPath: "refs/tags",
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestMaxAge)
	defer cancel()
	res, err := client.Refs(ctx, request)
	if err != nil {
		log.Errorf("Error retrieving tags:\n%v", err)
		return nil, err
	}
	return res, nil
}

// candidateBuildNums returns a list of build numbers from a list of possible
// builds that a given CL could have landed in, in reverse chronological order.
// It first finds all possible commits to the manifest-snapshots repository that
// could be a candidate. It then retrieves a mapping of build number -> commit SHA,
// for all commits in the manifest repo, and compares it with the candidate
// list to create a list of build numbers.
func candidateBuildNums(client gitilesProto.GitilesClient, manifestRepo string, targetData *clData) ([]string, utils.ChangelogError) {
	manifestCommits, utilErr := candidateManifestCommits(client, manifestRepo, targetData)
	if utilErr != nil {
		return nil, utilErr
	}
	tags, err := repoTags(client, manifestRepo)
	if err != nil {
		log.Errorf("tagMapping: Failed to retrieve tags for project %s:\n%v", manifestRepo, err)
		httpCode := utils.GerritErrCode(err)
		return nil, utils.FromFindBuildError(httpCode, targetData.CLNum)
	}
	log.Debug("Retrieving associated build number for each manifest commit")
	gitTagsMap := map[string]string{}
	for tagRef, manifestSHA := range tags.Revisions {
		gitTagsMap[manifestSHA] = tagRef
	}
	output := make([]string, len(manifestCommits))
	for i, commit := range manifestCommits {
		tag, ok := gitTagsMap[commit.Id]
		if !ok {
			log.Errorf("candidateBuildNums: No ref tag found for commit sha %s in repository %s", commit.Id, manifestRepo)
			return nil, utils.InternalError
		} else if len(tag) <= 10 {
			log.Errorf("candidateBuildNums: Ref tag: %s for commit sha %s is malformed", tag, commit.Id)
			return nil, utils.InternalError
		}
		// Remove refs/tags/ prefix for each git tag
		output[i] = gitTagsMap[commit.Id][10:]
	}
	return output, nil
}

// manifestData retrieves the commit SHA and remote URL used in a particular build
// for the same repository and branch as the target CL.
func manifestData(client gitilesProto.GitilesClient, manifestRepo string, buildNum string, targetData *clData, out chan manifestResponse, wg *sync.WaitGroup) {
	defer wg.Done()
	response, err := utils.DownloadManifest(client, manifestRepo, buildNum)
	log.Debugf("Parsing manifest for build %s", buildNum)
	if err != nil {
		out <- manifestResponse{Err: err}
		return
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(response.Contents); err != nil {
		out <- manifestResponse{Err: err}
		return
	}
	root := doc.SelectElement("manifest")
	// Parse each <remote fetch=X name=Y> tag in the manifest xml file.
	// Extract the "fetch" and "name" attributes from each remote tag, and map the name to the fetch URL.
	remoteMap := make(map[string]string)
	for _, remote := range root.SelectElements("remote") {
		url := strings.Replace(remote.SelectAttr("fetch").Value, "https://", "", 1)
		remoteMap[remote.SelectAttr("name").Value] = url
	}
	if root.SelectElement("default").SelectAttr("remote") != nil {
		remoteMap[""] = remoteMap[root.SelectElement("default").SelectAttr("remote").Value]
	}
	// Parse each <project> tag in the manifest xml file.
	// Some projects do not have a "remote" attribute.
	// If this is the case, they should use the default remoteURL.
	output := manifestResponse{BuildNum: buildNum}
	for _, project := range root.SelectElements("project") {
		repo := project.SelectAttr("name").Value
		branch := project.SelectAttrValue("dest-branch", "")
		// Remove refs/heads/ prefix for branch if specified
		if len(branch) > 0 {
			branch = branch[11:]
		}
		if repo == targetData.Project && (branch == "" || branch == targetData.Branch) {
			output.SHA = project.SelectAttr("revision").Value
			output.RemoteURL = remoteMap[project.SelectAttrValue("remote", "")]
		}
	}
	if output.SHA == "" || output.RemoteURL == "" {
		out <- manifestResponse{Err: fmt.Errorf("manifestData: Repository associated with CL could not be found in manifest %s", buildNum)}
		return
	}
	out <- output
}

// getRepoData retrieves information about the repository being modified by the
// CL. It retrieves candidate build numbers and their associated SHA, the
// the first and last SHA in the repository changelog, and the remote URL.
func getRepoData(client gitilesProto.GitilesClient, manifestRepo string, targetData *clData) (*repoData, utils.ChangelogError) {
	buildNums, err := candidateBuildNums(client, manifestRepo, targetData)
	if err != nil {
		return nil, err
	}
	log.Debug("Retrieving and parsing manifest file for each build")
	buildOrder := map[string]int{}
	for i, buildNum := range buildNums {
		buildOrder[buildNum] = i * -1
	}

	output := repoData{Candidates: map[string]string{}}
	shaChan := make(chan manifestResponse, len(buildNums))
	var wg sync.WaitGroup
	wg.Add(len(buildNums))
	for _, buildNum := range buildNums {
		go manifestData(client, manifestRepo, buildNum, targetData, shaChan, &wg)
	}
	wg.Wait()

	sourceOrder, targetOrder := len(buildNums), len(buildNums)*-1
	for i := 0; i < len(buildNums); i++ {
		curr := <-shaChan
		if curr.Err != nil {
			log.Debug(curr.Err)
			continue
		}
		if output.RemoteURL != "" && output.RemoteURL != curr.RemoteURL {
			log.Errorf("getRepoData: Remote URL for repository %s changed in build %s", targetData.Project, curr.BuildNum)
			return nil, utils.InternalError
		}
		output.RemoteURL = curr.RemoteURL
		// Since a manifest file may not use the repository/branch used by a
		// CL, need to select the earliest/latest builds that do
		if buildOrder[curr.BuildNum] > targetOrder {
			output.TargetSHA = curr.SHA
			targetOrder = buildOrder[curr.BuildNum]
		}
		if buildOrder[curr.BuildNum] < sourceOrder {
			output.SourceSHA = curr.SHA
			sourceOrder = buildOrder[curr.BuildNum]
		}
		if storedBuild, ok := output.Candidates[curr.SHA]; !ok || buildOrder[curr.BuildNum] < buildOrder[storedBuild] {
			output.Candidates[curr.SHA] = curr.BuildNum
		}
	}
	if len(output.Candidates) == 0 {
		log.Debugf("getRepoData: No builds found for CL %s", targetData.CLNum)
		return nil, utils.FromFindBuildError("404", targetData.CLNum)
	}
	return &output, nil
}

// firstBuild retrieves the earliest build containing the target CL from a map
// of candidate builds.
func firstBuild(changelog []*git.Commit, targetData *clData, candidates map[string]string) (string, utils.ChangelogError) {
	log.Debug("Scanning changelog for first build")
	targetIdx := -1
	for i, commit := range changelog {
		if commit.Id == targetData.Revision {
			targetIdx = i
		}
	}
	if targetIdx == -1 {
		return "", utils.FromFindBuildError("404", targetData.CLNum)
	}
	for i := targetIdx; i >= 0; i-- {
		currSHA := changelog[i].Id
		if buildNum, ok := candidates[currSHA]; ok {
			return buildNum, nil
		}
	}
	return "", utils.FromFindBuildError("404", targetData.CLNum)
}

// FindBuild locates the first build that a CL was introduced to.
func FindBuild(request *BuildRequest) (*BuildResponse, utils.ChangelogError) {
	log.Debugf("Fetching first build for CL: %s", request.CL)
	start := time.Now()
	if request == nil {
		log.Error("expected non-nil request")
		return nil, utils.InternalError
	}
	gerritClient, err := gerrit.NewClient(request.HTTPClient, request.GerritHost)
	if err != nil {
		log.Errorf("Failed to establish Gerrit client for host %s:\n%v", request.GerritHost, err)
		return nil, utils.InternalError
	}
	gitilesClient, err := gitilesApi.NewRESTClient(request.HTTPClient, request.GitilesHost, true)
	if err != nil {
		log.Errorf("Failed to establish Gitiles client for host %s:\n%v", request.GitilesHost, err)
		return nil, utils.InternalError
	}
	clData, clErr := getCLData(gerritClient, request.CL, request.RepoPrefix)
	if clErr != nil {
		return nil, clErr
	}
	repoData, clErr := getRepoData(gitilesClient, request.ManifestRepo, clData)
	if clErr != nil {
		return nil, clErr
	}
	// The remote URL for a repo may not be the same as the manifest remote URL
	if request.GitilesHost != repoData.RemoteURL {
		log.Debugf("Repository is located in different GoB host, setting gitiles client to URL: %s", repoData.RemoteURL)
		gitilesClient, err = gitilesApi.NewRESTClient(request.HTTPClient, repoData.RemoteURL, true)
		if err != nil {
			log.Errorf("failed to establish Gitiles client for host %s:\n%v", repoData.RemoteURL, err)
			return nil, utils.InternalError
		}
	}
	changelog, _, err := utils.Commits(gitilesClient, clData.Project, repoData.TargetSHA, repoData.SourceSHA, -1)
	if err != nil {
		httpCode := utils.GerritErrCode(err)
		return nil, utils.FromFindBuildError(httpCode, clData.CLNum)
	}
	buildNum, clErr := firstBuild(changelog, clData, repoData.Candidates)
	if clErr != nil {
		return nil, clErr
	}
	log.Debugf("Retrieved first build for CL: %s in %s\n", request.CL, time.Since(start))
	return &BuildResponse{
		BuildNum: buildNum,
		CLNum:    clData.CLNum,
	}, nil
}
