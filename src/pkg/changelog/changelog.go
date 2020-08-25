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
// This package uses Gitiles to request information from a Git on Borg instance.
// To generate a changelog, the package first retrieves the the manifest files for
// the two requested builds using the provided manifest GoB instance and repository.
// The package then parses the XML files and retrieves the committish and instance
// URL. A request is sent on a seperate thread for each repository, asking for a list
// of commits that occurred between the source committish and the target committish.
// Finally, the resulting git.Commit objects are converted to Commit objects, and
// consolidated into a mapping of repository path -> []*Commit.

package changelog

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cos.googlesource.com/cos/tools/src/pkg/utils"
	"github.com/beevik/etree"
	log "github.com/sirupsen/logrus"
	gitilesApi "go.chromium.org/luci/common/api/gitiles"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

type repo struct {
	Repo string
	Path string
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
	Commits        []*Commit
	Repo           string
	Path           string
	HasMoreCommits bool
	Err            error
}

type additionsResult struct {
	Additions map[string]*RepoLog
	Err       error
}

// RepoLog contains a changelist for a particular repository
type RepoLog struct {
	Commits        []*Commit
	Repo           string
	SourceSHA      string
	TargetSHA      string
	HasMoreCommits bool
}

// Creates a unique identifier for a repo + branch pairing.
// Path is used instead of dest-branch because some manifest files do not
// indicate a dest-branch for a project.
// Path itself is not sufficient to guarantee uniqueness, since some repos
// share the same path.
// ex. mirrors/cros/chromiumos/repohooks vs cos/repohooks
func repoID(name, path string) string {
	return name + path
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
			return fmt.Errorf("createClients: error creating client mapping:\n%w", err)
		}
		clients[remoteURL] = client
	}
	return nil
}

// repoMap generates a mapping of repository ID to instance URL and committish.
// This eliminates the need to track remote names and allows lookup
// of source committish when generating changelog.
func repoMap(manifest string) (map[string]*repo, error) {
	log.Debug("Mapping repository to instance URL and committish")
	if manifest == "" {
		return nil, fmt.Errorf("repoMap: manifest data is empty")
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromString(manifest); err != nil {
		return nil, fmt.Errorf("repoMap: error parsing manifest xml:\n%w", err)
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
		name, path := project.SelectAttr("name").Value, project.SelectAttr("path").Value
		repos[repoID(name, path)] = &repo{
			Repo:        name,
			Path:        path,
			InstanceURL: remoteMap[project.SelectAttrValue("remote", "")],
			Committish:  project.SelectAttr("revision").Value,
		}
	}
	return repos, nil
}

// mappedManifest retrieves a Manifest file from GoB and unmarshals XML.
// Returns a mapping of repository ID to repository data.
func mappedManifest(client gitilesProto.GitilesClient, repo string, buildNum string) (map[string]*repo, error) {
	log.Debugf("Retrieving manifest file for build %s\n", buildNum)
	response, err := utils.DownloadManifest(client, repo, buildNum)
	if err != nil {
		return nil, fmt.Errorf("mappedManifest: error downloading manifest file from repo %s:\n%w",
			repo, err)
	}
	mappedManifest, err := repoMap(response.Contents)
	if err != nil {
		return nil, fmt.Errorf("mappedManifest: error parsing manifest contents from repo %s:\n%w",
			repo, err)
	}
	return mappedManifest, nil
}

// commits get all commits that occur between committish and ancestor for a specific repo.
func commits(client gitilesProto.GitilesClient, path string, repo string, committish string, ancestor string, querySize int, outputChan chan commitsResult) {
	log.Debugf("Fetching changelog for repo: %s on committish %s\n", repo, committish)
	commits, hasMoreCommits, err := utils.Commits(client, repo, committish, ancestor, querySize)
	if err != nil {
		outputChan <- commitsResult{Err: err}
	}
	parsedCommits, err := ParseGitCommitLog(commits)
	if err != nil {
		outputChan <- commitsResult{Err: err}
	}
	outputChan <- commitsResult{
		Commits:        parsedCommits,
		Path:           path,
		Repo:           repo,
		HasMoreCommits: hasMoreCommits,
	}
}

// additions retrieves all commits that occured between 2 parsed manifest files for each repo.
// Returns a map of repo name -> list of commits.
func additions(clients map[string]gitilesProto.GitilesClient, sourceRepos map[string]*repo, targetRepos map[string]*repo, querySize int, outputChan chan additionsResult) {
	log.Debug("Retrieving commit additions")
	repoCommits := make(map[string]*RepoLog)
	commitsChan := make(chan commitsResult, len(targetRepos))
	for repoID, targetRepoInfo := range targetRepos {
		cl := clients[targetRepoInfo.InstanceURL]
		// If the source Manifest file does not contain a target repo,
		// count every commit since target repo creation as an addition
		ancestorCommittish := ""
		if sourceRepoInfo, ok := sourceRepos[repoID]; ok {
			ancestorCommittish = sourceRepoInfo.Committish
		}
		go commits(cl, targetRepoInfo.Path, targetRepoInfo.Repo, targetRepoInfo.Committish, ancestorCommittish, querySize, commitsChan)
	}
	for i := 0; i < len(targetRepos); i++ {
		res := <-commitsChan
		if res.Err != nil {
			outputChan <- additionsResult{Err: res.Err}
			return
		}
		var sourceSHA string
		if sourceData, ok := sourceRepos[repoID(res.Repo, res.Path)]; ok {
			sourceSHA = sourceData.Committish
		}
		if len(res.Commits) > 0 {
			repoCommits[res.Path] = &RepoLog{
				Commits:        res.Commits,
				HasMoreCommits: res.HasMoreCommits,
				Repo:           res.Repo,
				SourceSHA:      sourceSHA,
				TargetSHA:      targetRepos[repoID(res.Repo, res.Path)].Committish,
			}
		}
	}
	outputChan <- additionsResult{Additions: repoCommits}
	return
}

// Changelog generates a changelog between 2 build numbers
//
// httpClient is a authorized http.Client object with Gerrit scope.
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
		return nil, nil, fmt.Errorf("Changelog: error creating client for GoB instance: %s:\n%w", host, err)
	}
	sourceRepos, err := mappedManifest(manifestClient, repo, sourceBuildNum)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error retrieving mapped manifest for source build number: %s using manifest repository: %s:\n%w",
			sourceBuildNum, repo, err)
	}
	targetRepos, err := mappedManifest(manifestClient, repo, targetBuildNum)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error retrieving mapped manifest for target build number: %s using manifest repository: %s:\n%w",
			targetBuildNum, repo, err)
	}

	clients[host] = manifestClient
	err = createGerritClients(clients, httpClient, sourceRepos)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error creating source clients:\n%w", err)
	}
	err = createGerritClients(clients, httpClient, targetRepos)
	if err != nil {
		return nil, nil, fmt.Errorf("Changelog: error creating target clients:\n%w", err)
	}

	addChan := make(chan additionsResult, 1)
	missChan := make(chan additionsResult, 1)
	go additions(clients, sourceRepos, targetRepos, querySize, addChan)
	go additions(clients, targetRepos, sourceRepos, querySize, missChan)
	missRes := <-missChan
	if missRes.Err != nil {
		return nil, nil, fmt.Errorf("Changelog: failure when retrieving missed commits:\n%w", missRes.Err)
	}
	addRes := <-addChan
	if addRes.Err != nil {
		return nil, nil, fmt.Errorf("Changelog: failure when retrieving commit additions:\n%w", addRes.Err)
	}

	return addRes.Additions, missRes.Additions, nil
}
