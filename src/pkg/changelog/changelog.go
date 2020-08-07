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

// This package generates a changelog based on the commit history between
// two build numbers. The changelog consists of two outputs - the commits
// added to the target build that aren't present in the source build, and the
// commits in the source build that aren't present in the target build. This
// package uses concurrency to improve performance.
//
// This packages uses Gitiles to request information from a Git on Borg instance.
// To generate a changelog, the package first retrieves the the manifest files for
// the two requested builds using the provided manifest GoB instance and repository.
// The package then parses the XML files and retrieves the committish and instance
// URL. A request is sent on a seperate thread for each repository, asking for a list
// of commits that occurred between the source committish and the target committish.
// Finally, the resulting git.Commit objects are converted to Commit objects, and
// consolidated into a mapping of repositoryName -> []*Commit.

package changelog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/beevik/etree"
	log "github.com/sirupsen/logrus"
	gitilesApi "go.chromium.org/luci/common/api/gitiles"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

const (
	manifestFileName string = "snapshot.xml"

	// These constants are used for exponential increase in Gitiles request size.
	defaultPageSize          int = 1000
	pageSizeGrowthMultiplier int = 5
	maxPageSize              int = 10000
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

type commitsResult struct {
	RepoURL        string
	Commits        []*Commit
	HasMoreCommits bool
	Err            error
}

type additionsResult struct {
	Additions map[string]*RepoLog
	Err       error
}

// limitPageSize will restrict a request page size to min of pageSize (which grows exponentially)
// or remaining request size
func limitPageSize(pageSize, requestedSize int) int {
	if requestedSize == -1 || pageSize <= requestedSize {
		return pageSize
	}
	return requestedSize
}

func gerritClient(httpClient *http.Client, remoteURL string) (gitilesProto.GitilesClient, error) {
	log.Debugf("Creating Gerrit client for remote url %s\n", remoteURL)
	cl, err := gitilesApi.NewRESTClient(httpClient, remoteURL, true)
	if err != nil {
		return nil, errors.New("changelog: Failed to establish client to remote url: " + remoteURL)
	}
	return cl, nil
}

func createGerritClients(clients map[string]gitilesProto.GitilesClient, httpClient *http.Client, repoMap map[string]*repo) error {
	log.Debug("Creating additional Gerrit clients for manifest file if not already created")
	for _, repoData := range repoMap {
		remoteURL := repoData.InstanceURL
		if _, ok := clients[remoteURL]; ok {
			continue
		}
		client, err := gerritClient(httpClient, remoteURL)
		if err != nil {
			return fmt.Errorf("createClients: error creating client mapping:\n%v", err)
		}
		clients[remoteURL] = client
	}
	return nil
}

// repoMap generates a mapping of repo name to instance URL and committish.
// This eliminates the need to track remote names and allows lookup
// of source committish when generating changelog.
func repoMap(manifest string) (map[string]*repo, error) {
	log.Debug("Mapping repository to instance URL and committish")
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

// mappedManifest retrieves a Manifest file from GoB and unmarshals XML.
func mappedManifest(client gitilesProto.GitilesClient, repo string, buildNum string) (map[string]*repo, error) {
	log.Debugf("Retrieving manifest file for build %s\n", buildNum)
	request := gitilesProto.DownloadFileRequest{
		Project:    repo,
		Committish: "refs/tags/" + buildNum,
		Path:       manifestFileName,
		Format:     1,
	}
	response, err := client.DownloadFile(context.Background(), &request)
	if err != nil {
		return nil, fmt.Errorf("mappedManifest: error downloading manifest file from repo %s:\n%v",
			repo, err)
	}
	mappedManifest, err := repoMap(response.Contents)
	if err != nil {
		return nil, fmt.Errorf("mappedManifest: error parsing manifest contents from repo %s:\n%v",
			repo, err)
	}
	return mappedManifest, nil
}

// commits get all commits that occur between committish and ancestor for a specific repo.
func commits(client gitilesProto.GitilesClient, repo string, committish string, ancestor string, querySize int, outputChan chan commitsResult) {
	log.Debugf("Fetching changelog for repo: %s on committish %s\n", repo, committish)
	start := time.Now()

	pageSize := limitPageSize(defaultPageSize, querySize)
	querySize -= pageSize
	request := gitilesProto.LogRequest{
		Project:            repo,
		Committish:         committish,
		ExcludeAncestorsOf: ancestor,
		PageSize:           int32(pageSize),
	}
	response, err := client.Log(context.Background(), &request)
	if err != nil {
		outputChan <- commitsResult{Err: fmt.Errorf("commits: Error retrieving log for repo: %s with committish: %s and ancestor %s:\n%v",
			repo, committish, ancestor, err)}
		return
	}

	// No nextPageToken means there were less than <defaultPageSize> commits total.
	// We can immediately return.
	if response.NextPageToken == "" {
		log.Debugf("Retrieved %d commits from %s in %s\n", len(response.Log), repo, time.Since(start))
		parsedCommits, err := ParseGitCommitLog(response.Log)
		if err != nil {
			outputChan <- commitsResult{Err: fmt.Errorf("commits: Error parsing log response for repo: %s with committish: %s and ancestor %s:\n%v",
				repo, committish, ancestor, err)}
			return
		}
		outputChan <- commitsResult{RepoURL: repo, Commits: parsedCommits, HasMoreCommits: (response.NextPageToken != "")}
		return
	}
	// Retrieve remaining commits using exponential increase in pageSize.
	allCommits := response.Log
	for querySize > 0 && response.NextPageToken != "" {
		if pageSize < maxPageSize {
			pageSize *= pageSizeGrowthMultiplier
		}
		pageSize = limitPageSize(pageSize, querySize)
		querySize -= pageSize
		request := gitilesProto.LogRequest{
			Project:            repo,
			Committish:         committish,
			ExcludeAncestorsOf: ancestor,
			PageToken:          response.NextPageToken,
			PageSize:           int32(pageSize),
		}
		response, err = client.Log(context.Background(), &request)
		if err != nil {
			outputChan <- commitsResult{Err: fmt.Errorf("commits: Error retrieving log for repo: %s with committish: %s and ancestor %s:\n%v",
				repo, committish, ancestor, err)}
			return
		}
		allCommits = append(allCommits, response.Log...)
	}
	log.Debugf("Retrieved %d commits from %s in %s\n", len(allCommits), repo, time.Since(start))
	parsedCommits, err := ParseGitCommitLog(allCommits)
	if err != nil {
		outputChan <- commitsResult{Err: fmt.Errorf("commits: Error parsing log response for repo: %s with committish: %s and ancestor %s:\n%v",
			repo, committish, ancestor, err)}
		return
	}
	outputChan <- commitsResult{RepoURL: repo, Commits: parsedCommits, HasMoreCommits: (response.NextPageToken != "")}
}

// additions retrieves all commits that occured between 2 parsed manifest files for each repo.
// Returns a map of repo name -> list of commits.
func additions(clients map[string]gitilesProto.GitilesClient, sourceRepos map[string]*repo, targetRepos map[string]*repo, querySize int, outputChan chan additionsResult) {
	log.Debug("Retrieving commit additions")
	repoCommits := make(map[string]*RepoLog)
	commitsChan := make(chan commitsResult, len(targetRepos))
	for repoURL, targetRepoInfo := range targetRepos {
		cl := clients[targetRepoInfo.InstanceURL]
		// If the source Manifest file does not contain a target repo,
		// count every commit since target repo creation as an addition
		ancestorCommittish := ""
		if sourceRepoInfo, ok := sourceRepos[repoURL]; ok {
			ancestorCommittish = sourceRepoInfo.Committish
		}
		go commits(cl, repoURL, targetRepoInfo.Committish, ancestorCommittish, querySize, commitsChan)
	}
	for i := 0; i < len(targetRepos); i++ {
		res := <-commitsChan
		if res.Err != nil {
			outputChan <- additionsResult{Err: res.Err}
			return
		}
		sourceSHA := ""
		if sha, ok := sourceRepos[res.RepoURL]; ok {
			sourceSHA = sha.Committish
		}
		if len(res.Commits) > 0 {
			repoCommits[res.RepoURL] = &RepoLog{
				Commits:        res.Commits,
				HasMoreCommits: res.HasMoreCommits,
				SourceSHA:      sourceSHA,
				TargetSHA:      targetRepos[res.RepoURL].Committish,
			}
		}
	}
	outputChan <- additionsResult{Additions: repoCommits}
	return
}

// Changelog generates a changelog between 2 build numbers
//
// authenticator is an auth.Authenticator object that is used to build authenticated
// Gitiles clients
//
// sourceBuildNum and targetBuildNum should be build numbers. It should match
// a tag that links directly to snapshot.xml
// Ex. For /refs/tags/15049.0.0, the argument should be 15049.0.0
//
// host should be the GoB instance that Manifest files are hosted in
// ex. "cos.googlesource.com"
//
// repo should be the repository that build manifest files
// are located, ex. "cos/manifest-snapshots"
//
// querySize should be the number of commits that should be included in each
// repository changelog. Specify as -1 to get all commits
//
// Outputs two changelogs
// The first changelog contains new commits that were added to the target
// build starting from the source build number
//
// The second changelog contains all commits that are present in the source build
// but not present in the target build
func Changelog(httpClient *http.Client, sourceBuildNum string, targetBuildNum string, host string, repo string, querySize int) (map[string]*RepoLog, map[string]*RepoLog, error) {
	if httpClient == nil {
		return nil, nil, errors.New("Changelog: httpClient should not be nil")
	}

	log.Infof("Retrieving changelog between %s and %s\n", sourceBuildNum, targetBuildNum)
	clients := make(map[string]gitilesProto.GitilesClient)

	// Since the manifest file is always in the cos instance, add cos client
	// so that client knows what URL to use
	manifestClient, err := gerritClient(httpClient, host)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error creating client for GoB instance: %s:\n%v", host, err)
	}
	sourceRepos, err := mappedManifest(manifestClient, repo, sourceBuildNum)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error retrieving mapped manifest for source build number: %s using manifest repository: %s:\n%v",
			sourceBuildNum, repo, err)
	}
	targetRepos, err := mappedManifest(manifestClient, repo, targetBuildNum)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error retrieving mapped manifest for target build number: %s using manifest repository: %s:\n%v",
			targetBuildNum, repo, err)
	}

	clients[host] = manifestClient
	err = createGerritClients(clients, httpClient, sourceRepos)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error creating source clients:\n%v", err)
	}
	err = createGerritClients(clients, httpClient, targetRepos)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error creating target clients:\n%v", err)
	}

	addChan := make(chan additionsResult, 1)
	missChan := make(chan additionsResult, 1)
	go additions(clients, sourceRepos, targetRepos, querySize, addChan)
	go additions(clients, targetRepos, sourceRepos, querySize, missChan)
	missRes := <-missChan
	if missRes.Err != nil {
		return nil, nil, fmt.Errorf("Changelog: failure when retrieving missed commits:\n%v", err)
	}
	addRes := <-addChan
	if addRes.Err != nil {
		return nil, nil, fmt.Errorf("Changelog: failure when retrieving commit additions:\n%v", err)
	}

	return addRes.Additions, missRes.Additions, nil
}
