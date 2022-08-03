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
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	"github.com/beevik/etree"
	_ "github.com/go-sql-driver/mysql"
	"go.chromium.org/luci/common/proto/git"
	"go.chromium.org/luci/common/proto/gitiles"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	gerrit "github.com/andygrunwald/go-gerrit"
	log "github.com/sirupsen/logrus"
	gitilesApi "go.chromium.org/luci/common/api/gitiles"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

const (
	// Exponential search range variables
	defaultSearchRange    = 5 // Search range in days
	searchRangeMultiplier = 5
	// Maximum time to wait for a response from a Gerrit or Gitiles request
	requestMaxAge = 30 * time.Second
	// Max size of changelog if no changelog source is specified
	noSourceChangelogSize = 10000

	shortSHALength = 7
	fullSHALength  = 40

	// Definitions of column names in table.
	commitSha       = "commit_sha"
	cLNumber        = "cl_number"
	landedInBuild   = "landed_build_number"
	releasedInBuild = "released_build_number"
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
		"chromiumos/third_party/kernel": {
			releaseRe:      regexp.MustCompile("(.*)-chromeos-.*"),
			defaultRelease: "master",
		},
		"chromiumos/third_party/lakitu-kernel": {
			releaseRe:      regexp.MustCompile("(.*)-lakitu-.*"),
			defaultRelease: "master",
		},
	}
	// crosRepoRe is used to strip chromium prefixes from the repo name.
	crosRepoRe = regexp.MustCompile("^(?:chromeos|chrome|chromiumos|chromium)?/(.*)")
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
	// CL can be either the CL number or commit SHA of your target CL
	// ex. 3741 or If9f774179322c413fa0fd5ebb3dd615c5b22cd6c
	CL string
}

// iterCache contains information to perform an iteration of the
// findBuildInRange search on a specific time range. It is used to pass information
// that does not change between iterations, such as manifest tags
type iterCache struct {
	GitilesClient   gitilesProto.GitilesClient
	ManifestCommits []*git.Commit
	Tags            map[string]string
}

// BuildResponse is the output struct for the FindBuild function
type BuildResponse struct {
	BuildNum string
	CLNum    string
}

type clData struct {
	CLNum            string
	InstanceURL      string
	Project          string
	Release          string
	Branch           string
	Revision         string
	SearchStartRange time.Time
	SearchEndRange   time.Time
}

type repoData struct {
	Candidates map[string]string
	SourceSHA  string
	TargetSHA  string
	RemoteURL  string
}

type manifestResponse struct {
	BuildNum  string
	Repo      string
	SHA       string
	RemoteURL string
	Err       error
}

func queryString(clID string) string {
	if len(clID) == fullSHALength {
		return fmt.Sprintf("commit:%s", clID)
	}
	return fmt.Sprintf("change:%s", clID)
}

// queryCL retrieves the list of CLs matching a query from Gerrit
func queryCL(client *gerrit.Client, clID, instanceURL string) (gerrit.ChangeInfo, utils.ChangelogError) {
	log.Debugf("Retrieving CL List from Gerrit for clID: %q", clID)
	query := queryString(clID)
	queryOptions := &gerrit.QueryChangeOptions{}
	queryOptions.Query = []string{query}
	queryOptions.AdditionalFields = []string{"CURRENT_REVISION"}
	queryOptions.Limit = 1

	clList, _, err := client.Changes.QueryChanges(queryOptions)
	if err != nil {
		log.Errorf("queryCL: Error retrieving change for input %s:\n%v", clID, err)
		httpCode := utils.GerritErrCode(err)
		if httpCode == "403" {
			return gerrit.ChangeInfo{}, utils.ForbiddenError
		} else if httpCode == "400" || httpCode == "404" {
			return gerrit.ChangeInfo{}, utils.CLNotFound(clID)
		}
		return gerrit.ChangeInfo{}, utils.InternalServerError
	}
	if len(*clList) == 0 {
		log.Errorf("queryCL: CL with identifier %s not found", clID)
		return gerrit.ChangeInfo{}, utils.CLNotFound(clID)
	}
	change := (*clList)[0]
	log.Debugf("Found CL: %+v", change)
	if change.Submitted == nil {
		log.Debugf("Provided CL identifier %s maps to an unsubmitted CL", clID)
		return gerrit.ChangeInfo{}, utils.CLNotSubmitted(strconv.Itoa(change.Number), instanceURL)
	}
	return change, nil
}

func getCLData(clID, instanceURL string, httpClient *http.Client) (*clData, utils.ChangelogError) {
	log.Debugf("Retrieving CL data from Gerrit for changeID: %s", clID)
	gerritClient, clientErr := gerrit.NewClient(instanceURL, httpClient)
	if clientErr != nil {
		log.Errorf("failed to establish Gerrit client for host %s:\n%v", instanceURL, clientErr)
		return nil, utils.InternalServerError
	}
	change, err := queryCL(gerritClient, clID, instanceURL)
	if err != nil {
		return nil, err
	}
	log.Debugf("Target CL found with SHA %s on repo %s, branch %s", change.CurrentRevision, change.Project, change.Branch)
	// If a repository has non-conventional branch names, need to convert the
	// repository branch name to a release branch name
	release := change.Branch
	if rule, ok := clReleaseMapping[change.Project]; ok {
		if matches := rule.releaseRe.FindStringSubmatch(release); matches != nil {
			release = matches[1]
		} else {
			release = rule.defaultRelease
		}
	}
	// In case the branch associated with the change is "main", branch
	// on the manifest-snapshot will be master.
	if change.Branch == "main" {
		release = "master"
	}
	// Strip chromium prefixes
	project := change.Project
	if matches := crosRepoRe.FindStringSubmatch(project); matches != nil {
		project = matches[1]
	}
	submittedTime := *change.Submitted
	return &clData{
		CLNum:            strconv.Itoa(change.Number),
		InstanceURL:      instanceURL,
		Project:          project,
		Release:          release,
		Branch:           change.Branch,
		Revision:         change.CurrentRevision,
		SearchStartRange: submittedTime.Time,
		SearchEndRange:   submittedTime.Time.AddDate(0, 0, defaultSearchRange),
	}, nil
}

// candidateManifestCommits returns a list of commits to the manifest-snapshot
// repo that were committed within a time range from the target commit time, in
// reverse chronological order.
//
// Returns a list of candidate manifest commits, a bool indicating whether the
// search range can be expanded, and an error
func candidateManifestCommits(manifestCommits []*git.Commit, clData *clData) ([]*git.Commit, bool, utils.ChangelogError) {
	log.Debugf("Retrieving all manifest snapshots committed within %v to %v", clData.SearchStartRange, clData.SearchEndRange)
	if manifestCommits[0].Committer.Time.AsTime().Before(clData.SearchStartRange) {
		return nil, false, utils.CLTooRecent(clData.CLNum, clData.InstanceURL)
	}
	// Find latest commit that occurs before the target commit time.
	// allManifests is in reverse chronological order.
	left, right := 0, len(manifestCommits)-1
	for left < right {
		mid := (left + right) / 2
		if manifestCommits[mid].Committer == nil {
			log.Errorf("manifest %s has no committer", manifestCommits[mid].Id)
			return nil, false, utils.InternalServerError
		}
		currDate := manifestCommits[mid].Committer.Time.AsTime()
		if currDate.Before(clData.SearchStartRange) {
			right = mid
		} else {
			left = mid + 1
		}
	}
	earliestIdx := left
	// Find the earliest commit that occurs one week after the target commit time.
	// Don't have to reset right, since anything to the right is earlier.
	left = 0
	for left < right {
		mid := (left+right)/2 + 1
		if manifestCommits[mid].Committer == nil {
			log.Errorf("manifest %s has no committer", manifestCommits[mid].Id)
			return nil, false, utils.InternalServerError
		}
		currDate := manifestCommits[mid].Committer.Time.AsTime()
		if currDate.After(clData.SearchEndRange) {
			left = mid
		} else {
			right = mid - 1
		}
	}
	latestIdx := right
	return manifestCommits[latestIdx : earliestIdx+1], latestIdx != 0, nil
}

// repoTags retrieves all tags belonging to a repository
func repoTags(client *gerrit.Client, repo string) (map[string]string, error) {
	log.Debugf("Retrieving tags for repository %s", repo)
	tagInfos, _, err := client.Projects.ListTags(repo, &gerrit.ProjectBaseOptions{})
	if err != nil {
		log.Errorf("error retrieving tags:\n%v", err)
		return nil, err
	}
	tags := make(map[string]string)
	for _, tagInfo := range *tagInfos {
		log.Debugf("Tag found: %+v", tagInfo)
		commitSHA := tagInfo.Revision
		if tagInfo.Object != "" {
			commitSHA = tagInfo.Object
		}
		tags[tagInfo.Ref] = commitSHA
	}
	return tags, nil
}

// candidateBuildNums returns a list of build numbers from a list of possible
// builds that a given CL could have landed in, in reverse chronological order.
// It first finds all possible commits to the manifest-snapshots repository that
// could be a candidate. It then retrieves a mapping of build number -> commit SHA,
// for all commits in the manifest repo, and compares it with the candidate
// list to create a list of build numbers.
func candidateBuildNums(manifestCommits []*git.Commit, tags map[string]string) ([]string, utils.ChangelogError) {
	log.Debug("Retrieving associated build number for each manifest commit")
	gitTagsMap := map[string]string{}
	for tagRef, manifestSHA := range tags {
		gitTagsMap[manifestSHA] = tagRef
	}
	output := make([]string, len(manifestCommits))
	for i, commit := range manifestCommits {
		tag, ok := gitTagsMap[commit.Id]
		if !ok {
			log.Errorf("no ref tag found for commit sha %s", commit.Id)
			return nil, utils.InternalServerError
		} else if len(tag) <= 10 {
			log.Errorf("ref tag: %s for commit sha %s is malformed", tag, commit.Id)
			return nil, utils.InternalServerError
		}
		// Remove refs/tags/ prefix for each git tag
		output[i] = gitTagsMap[commit.Id][10:]
	}
	return output, nil
}

// manifestData retrieves the commit SHA and remote URL used in a particular build
// for the same repository and branch as the target CL.
func manifestData(client gitilesProto.GitilesClient, manifestRepo string, buildNum string, clData *clData, out chan manifestResponse, wg *sync.WaitGroup) {
	defer wg.Done()
	response, err := utils.DownloadManifest(client, manifestRepo, buildNum)
	log.Debugf("Parsing manifest for build %s", buildNum)
	if err != nil {
		out <- manifestResponse{Err: err}
		return
	}
	if response.Contents == "" {
		// If an empty manifest file is encountered, an empty string SHA is
		// inserted to instruct findBuild to retrieve a complete repo changelog
		out <- manifestResponse{BuildNum: buildNum, SHA: ""}
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
		branch := project.SelectAttrValue("upstream", "")
		if branch == "" {
			branch = project.SelectAttrValue("dest-branch", "")
		}
		// Remove refs/heads/ prefix for branch if specified
		if len(branch) > 0 {
			branch = branch[11:]
		}
		if strings.Contains(repo, clData.Project) && (branch == "" || branch == clData.Branch) {
			clData.Project = repo
			output.SHA = project.SelectAttr("revision").Value
			output.Repo = repo
			output.RemoteURL = remoteMap[project.SelectAttrValue("remote", "")]
		}
	}
	if output.SHA == "" || output.RemoteURL == "" {
		out <- manifestResponse{BuildNum: buildNum, SHA: ""}
		return
	}
	out <- output
}

// getRepoData retrieves information about the repository being modified by the
// CL. It retrieves candidate build numbers and their associated SHA, the
// the first and last SHA in the repository changelog, and the remote URL.
func getRepoData(client gitilesProto.GitilesClient, manifestRepo string, clData *clData, buildNums []string) (*repoData, utils.ChangelogError) {
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
		go manifestData(client, manifestRepo, buildNum, clData, shaChan, &wg)
	}
	wg.Wait()

	sourceOrder, targetOrder := len(buildNums), len(buildNums)*-1
	for i := 0; i < len(buildNums); i++ {
		curr := <-shaChan
		if curr.Err != nil {
			log.Debug(curr.Err)
			continue
		}
		// Since a manifest file may not use the repository/branch used by a
		// CL, need to select the earliest/latest builds that do
		if buildOrder[curr.BuildNum] > targetOrder {
			output.TargetSHA = curr.SHA
			output.RemoteURL = curr.RemoteURL
			if curr.Repo != "" {
				clData.Project = curr.Repo
			}
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
		log.Debugf("getRepoData: No builds found for CL %s", clData.CLNum)
		return nil, utils.CLNotUsed(clData.CLNum, clData.Project, clData.Release, clData.InstanceURL)
	}
	return &output, nil
}

// firstBuild retrieves the earliest build containing the target CL from a map
// of candidate builds.
func firstBuild(changelog []*git.Commit, clData *clData, candidates map[string]string) (string, utils.ChangelogError) {
	log.Debug("Scanning changelog for first build")
	targetIdx := -1
	for i, commit := range changelog {
		if commit.Id == clData.Revision {
			targetIdx = i
		}
	}
	if targetIdx == -1 {
		return "", utils.CLLandingNotFound(clData.CLNum, clData.InstanceURL)
	}
	for i := targetIdx; i >= 0; i-- {
		currSHA := changelog[i].Id
		if buildNum, ok := candidates[currSHA]; ok {
			return buildNum, nil
		}
	}
	return "", utils.CLLandingNotFound(clData.CLNum, clData.InstanceURL)
}

// findBuildInRange searches for the first build containing a given CL in
// Git on Borg within the specified start and end time range.
//
// Returns the build number if found, a bool indicating if the search range
// can be further expanded, and an error.
func findBuildInRange(request *BuildRequest, cache *iterCache, clData *clData) (string, bool, utils.ChangelogError) {
	log.Debugf("Searching for first build containing CL from time %v to time %v", clData.SearchStartRange, clData.SearchEndRange)
	var err error
	manifestCommits, canExpand, utilErr := candidateManifestCommits(cache.ManifestCommits, clData)
	if utilErr != nil {
		return "", canExpand, utilErr
	}
	buildNums, utilErr := candidateBuildNums(manifestCommits, cache.Tags)
	if err != nil {
		return "", canExpand, utilErr
	}
	repoData, utilErr := getRepoData(cache.GitilesClient, request.ManifestRepo, clData, buildNums)
	if utilErr != nil {
		return "", canExpand, utilErr
	}
	if repoData.TargetSHA == "" {
		return "", canExpand, utils.CLLandingNotFound(clData.CLNum, request.GerritHost)
	}
	changelogClient := cache.GitilesClient
	if repoData.RemoteURL != request.GitilesHost {
		log.Debugf("Different remote URL used in build, setting remote URL to %s", repoData.RemoteURL)
		changelogClient, err = gitilesApi.NewRESTClient(request.HTTPClient, repoData.RemoteURL, true)
		if err != nil {
			log.Errorf("failed to establish Gitiles client for remote URL %s", repoData.RemoteURL)
			return "", false, utils.InternalServerError
		}
	}
	querySize := -1
	if repoData.SourceSHA == "" {
		querySize = noSourceChangelogSize
	}
	changelog, _, err := utils.Commits(changelogClient, clData.Project, repoData.TargetSHA, repoData.SourceSHA, querySize)
	if err != nil {
		log.Errorf("failed to retrieve changelog: %v", err)
		if utils.GitilesErrCode(err) == "404" {
			return "", canExpand, utils.CLNotUsed(clData.CLNum, clData.Project, clData.Release, clData.InstanceURL)
		}
		return "", canExpand, utils.InternalServerError
	}
	buildNum, utilErr := firstBuild(changelog, clData, repoData.Candidates)
	if utilErr != nil {
		return "", canExpand, utilErr
	}
	return buildNum, canExpand, nil
}

// findBuildExponential searches for the first build containing a CL in an
// exponentially increasing time range.
func findBuildExponential(gitilesClient gitiles.GitilesClient, request *BuildRequest, clData *clData) (string, utils.ChangelogError) {
	log.Debug("Searching for first build in exponentially increasing time range")
	timeRange := defaultSearchRange

	// Manifest commits and tags only need to be retrieved once and can be
	// reused for each iteration.
	manifestCommits, _, err := utils.Commits(gitilesClient, request.ManifestRepo, "refs/heads/"+clData.Release, "", -1)
	if err != nil {
		log.Errorf("error retrieving manifest commits within CL submission range: %v", err)
		httpCode := utils.GitilesErrCode(err)
		if httpCode == "404" {
			return "", utils.CLInvalidRelease(clData.CLNum, clData.Release, clData.InstanceURL)
		}
		return "", utils.InternalServerError
	}
	if manifestCommits[len(manifestCommits)-1].Committer.Time.AsTime().After(clData.SearchEndRange) {
		clData.SearchStartRange = manifestCommits[len(manifestCommits)-1].Committer.Time.AsTime().Add(-time.Second)
		clData.SearchEndRange = clData.SearchStartRange.AddDate(0, 0, defaultSearchRange)
		log.Debugf("CL submitted earlier than first build, set search range to starting time from %v to %v", clData.SearchStartRange, clData.SearchEndRange)
	}
	// Creating a Gerrit client based on manifest-snapshot repository.
	// The client will be used for finding information associated with
	// an annotated git tag.
	instanceURL, err := utils.CreateGerritURL(request.GitilesHost)
	if err != nil {
		log.Errorf("failed to create Gerrit URL from Gitiles Host %q: %v", request.GitilesHost, err)
		return "", utils.InternalServerError
	}
	gerritClient, err := gerrit.NewClient(instanceURL, request.HTTPClient)
	if err != nil {
		log.Errorf("failed to establish Gerrit client for host %s:\n%v", instanceURL, err)
		return "", utils.InternalServerError
	}
	tagResp, err := repoTags(gerritClient, request.ManifestRepo)
	if err != nil {
		log.Errorf("failed to retrieve tags for project %s:\n%v", request.ManifestRepo, err)
		return "", utils.InternalServerError
	}
	cache := &iterCache{
		GitilesClient:   gitilesClient,
		Tags:            tagResp,
		ManifestCommits: manifestCommits,
	}

	res, canExpand, utilErr := findBuildInRange(request, cache, clData)
	for utilErr != nil && utilErr.Retryable() && canExpand {
		timeRange *= searchRangeMultiplier
		clData.SearchStartRange = clData.SearchEndRange.AddDate(0, 0, -defaultSearchRange)
		clData.SearchEndRange = clData.SearchEndRange.AddDate(0, 0, timeRange)
		log.Debugf("Could not locate CL in current time range, retrying with range %v to %v", clData.SearchStartRange, clData.SearchEndRange)
		res, canExpand, utilErr = findBuildInRange(request, cache, clData)
	}
	return res, utilErr
}

// FindBuild locates the first build that a CL was introduced to.
func FindBuild(request *BuildRequest) (*BuildResponse, utils.ChangelogError) {
	log.Debugf("Fetching first build for CL: %s", request.CL)
	start := time.Now()
	if request == nil {
		log.Error("expected non-nil request")
		return nil, utils.InternalServerError
	}
	gitilesClient, err := gitilesApi.NewRESTClient(request.HTTPClient, request.GitilesHost, true)
	if err != nil {
		log.Errorf("failed to establish Gitiles client for host %s:\n%v", request.GitilesHost, err)
		return nil, utils.InternalServerError
	}
	clData, clErr := getCLData(request.CL, request.GerritHost, request.HTTPClient)
	if clErr != nil {
		return nil, clErr
	}
	buildNum, clErr := findBuildExponential(gitilesClient, request, clData)
	if clErr != nil {
		return nil, clErr
	}
	log.Debugf("Retrieved first build for CL: %s in %s\n", request.CL, time.Since(start))
	return &BuildResponse{
		BuildNum: buildNum,
		CLNum:    clData.CLNum,
	}, nil
}

type secretBundle struct {
	name  string
	value *string
}

// findReleasedBuild locates the first build that a CL was introduced in using the builds-info database
func FindReleasedBuild(request *BuildRequest) (*BuildResponse, utils.ChangelogError) {
	log.Debugf("Fetching first build for CL: %s", request.CL)
	// access secretmanager
	client, err := secretmanager.NewClient(context.Background())
	if err != nil {
		log.Error("failed to create secretmanager client: %v", err)
		return nil, utils.InternalServerError
	}
	var (
		projectID            = os.Getenv("COS_CHANGELOG_PROJECT_ID")
		dbProjectSecretName  = os.Getenv("COS_FINDBUILD_DB_PROJECT")
		findBuildDbName      = os.Getenv("COS_FINDBUILD_DB_NAME")
		findBuildTableName   = os.Getenv("COS_FINDBUILD_TABLE_NAME")
		dbPasswordSecretName = os.Getenv("COS_FINDBUILD_PASSWORD_NAME")
		instanceSecretName   = os.Getenv("COS_FINDBUILD_INSTANCE_NAME")
		user                 = "readonly"
		zone                 = "us-west2"
	)
	var dbProjectID, dbName, tableName, password, instanceName string
	if err := retrieveSecrets(client, projectID, []secretBundle{
		{findBuildDbName, &dbName},
		{findBuildTableName, &tableName},
		{dbPasswordSecretName, &password},
		{instanceSecretName, &instanceName},
		{dbProjectSecretName, &dbProjectID},
	}); err != nil {
		log.Error("failed to retrieve secrets from secretmanager: %v", err)
		return nil, utils.InternalServerError
	}
	connectionName := dbProjectID + ":" + zone + ":" + instanceName
	// connect to database
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@cloudsql(%s)/%s", user, password, connectionName, dbName))
	if err != nil {
		log.Fatalf("Could not open db: %v", err)
	}
	// query database
	// SELECT release_build_number FROM DBName WHERE cLNumber = request.CL;
	queryStmt := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", releasedInBuild, tableName, cLNumber)
	rows, err := db.Query(queryStmt, request.CL)
	if err != nil {
		log.Fatalf("Could not query db: %v", err)
	}
	// change rows to BuildResponse type
	defer rows.Close()
	var releasedBuildNumber string
	if rows.Next() {
		if err := rows.Scan(&releasedBuildNumber); err != nil {
			log.Errorf("Could not scan result: %v", err)
			return nil, utils.InternalServerError
		}
	} else {
		log.Errorf("No build number found")
		return nil, nil
	}
	if rows.Next() {
		log.Errorf("More than one build number found")
	}
	releasedBuild := BuildResponse{}
	releasedBuild.BuildNum = releasedBuildNumber
	releasedBuild.CLNum = request.CL
	return &releasedBuild, nil
}

func retrieveSecret(client *secretmanager.Client, projectID string, secretName string) (string, error) {
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
	}
	result, err := client.AccessSecretVersion(context.Background(), accessRequest)
	if err != nil {
		return "", fmt.Errorf("failed to access secret at %s: %v", accessRequest.Name, err)
	}
	return string(result.Payload.Data), nil
}

func retrieveSecrets(client *secretmanager.Client, projectID string, secrets []secretBundle) error {
	for _, secret := range secrets {
		fetchedSecret, err := retrieveSecret(client, projectID, secret.name)
		if err != nil {
			return err
		}
		*secret.value = fetchedSecret
	}
	return nil
}
