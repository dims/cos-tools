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
// commits in the source build that aren't present in the target build. It
// also finds the sysctl value changes between two builds by fetching artifacts
// from GCS. This package uses concurrency to improve performance.
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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
	"github.com/beevik/etree"

	log "github.com/sirupsen/logrus"
	gitilesApi "go.chromium.org/luci/common/api/gitiles"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

var (
	imageBuildRe = regexp.MustCompile("^cos-(dev-|beta-|stable-|rc-)?\\d+-([\\d-]+)$")
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

type commitsRequest struct {
	Client      gitilesProto.GitilesClient
	InstanceURL string
	Path        string
	Repo        string
	Committish  string
	Ancestor    string
	QuerySize   int
	OutputChan  chan commitsResult
}

type commitsResult struct {
	Commits        []*Commit
	InstanceURL    string
	Repo           string
	Path           string
	HasMoreCommits bool
	Err            utils.ChangelogError
}

type additionsResult struct {
	Additions map[string]*RepoLog
	Err       utils.ChangelogError
}

// RepoLog contains a changelist for a particular repository
type RepoLog struct {
	Commits        []*Commit
	InstanceURL    string
	Repo           string
	SourceSHA      string
	TargetSHA      string
	HasMoreCommits bool
}

// resolveImageName returns the build number associated with an image name.
// If the string is not an image name, it returns the input string.
func resolveImageName(imageName string) string {
	build := imageBuildRe.FindStringSubmatch(imageName)
	if len(build) < 2 {
		return imageName
	}
	buildNum := strings.Replace(build[2], "-", ".", 3)
	log.Debugf("resolveImageName: image name %s was resolved to build number %s", imageName, buildNum)
	return buildNum
}

// limitPageSize will restrict a request page size to min of pageSize (which grows exponentially)
// or remaining request size
func limitPageSize(pageSize, requestedSize int) int {
	if requestedSize == -1 || pageSize <= requestedSize {
		return pageSize
	}
	return requestedSize
}

func gitilesClient(httpClient *http.Client, remoteURL string) (gitilesProto.GitilesClient, utils.ChangelogError) {
	log.Debugf("Creating Gitiles client for remote url %s\n", remoteURL)
	cl, err := gitilesApi.NewRESTClient(httpClient, remoteURL, true)
	if err != nil {
		log.Errorf("gitilesClient: failed to create client for remote url %s", remoteURL)
		return nil, utils.InternalServerError
	}
	return cl, nil
}

func createGitilesClients(clients map[string]gitilesProto.GitilesClient, httpClient *http.Client, repoMap map[string]*repo) utils.ChangelogError {
	log.Debug("Creating additional Gerrit clients for manifest file if not already created")
	for _, repoData := range repoMap {
		remoteURL := repoData.InstanceURL
		if _, ok := clients[remoteURL]; ok {
			continue
		}
		client, err := gitilesClient(httpClient, remoteURL)
		if err != nil {
			return err
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
		log.Error("repoMap: manifest file is empty")
		return nil, errors.New("manifest file is empty")
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromString(manifest); err != nil {
		log.Debug("repoMap: error parsing manifest xml:\n%w", err)
		return nil, errors.New("could not parse XML for manifest file associated with build")
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
		name, path := project.SelectAttr("name").Value, project.SelectAttrValue("path", "")
		repos[path] = &repo{
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
func mappedManifest(client gitilesProto.GitilesClient, repo string, buildInput, buildNum string) (map[string]*repo, utils.ChangelogError) {
	log.Debugf("Retrieving manifest file for build %s\n", buildNum)
	response, err := utils.DownloadManifest(client, repo, buildNum)
	if err != nil {
		log.Errorf("mappedManifest: error downloading manifest file from repo %s for build %s:\n%v", repo, buildNum, err)
		httpCode := utils.GitilesErrCode(err)
		if httpCode == "403" {
			return nil, utils.ForbiddenError
		} else if httpCode == "404" {
			return nil, utils.BuildNotFound(buildInput)
		}
		return nil, utils.InternalServerError
	}
	mappedManifest, err := repoMap(response.Contents)
	if err != nil {
		log.Errorf("mappedManifest: error retrieving mapped manifest file from repo %s for build %s:\n%v", repo, buildNum, err)
		httpCode := utils.GitilesErrCode(err)
		if httpCode == "404" {
			return nil, utils.BuildNotFound(buildInput)
		}
		return nil, utils.InternalServerError
	}
	return mappedManifest, nil
}

// commits get all commits that occur between committish and ancestor for a specific repo.
func commits(req commitsRequest) {
	log.Debugf("Fetching changelog for repo: %s on committish %s\n", req.Repo, req.Committish)
	commits, hasMoreCommits, err := utils.Commits(req.Client, req.Repo, req.Committish, req.Ancestor, req.QuerySize)
	if err != nil {
		if utils.GitilesErrCode(err) == "404" {
			req.OutputChan <- commitsResult{
				InstanceURL: req.InstanceURL,
				Path:        req.Path,
				Repo:        req.Repo,
			}
		} else {
			log.Errorf("commits: error retrieving commit changelog on repo %s from commit %s to commit %s:\n%v", req.Repo, req.Committish, req.Ancestor, err)
			req.OutputChan <- commitsResult{Err: utils.InternalServerError}
		}
		return
	}
	if commits == nil {
		log.Info(req.Repo, req.Committish, req.Ancestor)
	}
	parsedCommits, err := ParseGitCommitLog(commits)
	if err != nil {
		log.Errorf("commits: error parsing Gitiles commits response\n%v", err)
		req.OutputChan <- commitsResult{Err: utils.InternalServerError}
		return
	}
	req.OutputChan <- commitsResult{
		Commits:        parsedCommits,
		InstanceURL:    req.InstanceURL,
		Path:           req.Path,
		Repo:           req.Repo,
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
		commitsReq := commitsRequest{
			Client:      cl,
			Path:        targetRepoInfo.Path,
			InstanceURL: targetRepoInfo.InstanceURL,
			Repo:        targetRepoInfo.Repo,
			Committish:  targetRepoInfo.Committish,
			Ancestor:    ancestorCommittish,
			QuerySize:   querySize,
			OutputChan:  commitsChan,
		}
		go commits(commitsReq)
	}
	for i := 0; i < len(targetRepos); i++ {
		res := <-commitsChan
		if res.Err != nil {
			outputChan <- additionsResult{Err: res.Err}
			return
		}
		var sourceSHA string
		if sourceData, ok := sourceRepos[res.Path]; ok {
			sourceSHA = sourceData.Committish
		}
		if len(res.Commits) > 0 {
			repoCommits[res.Path] = &RepoLog{
				Commits:        res.Commits,
				HasMoreCommits: res.HasMoreCommits,
				InstanceURL:    res.InstanceURL,
				Repo:           res.Repo,
				SourceSHA:      sourceSHA,
				TargetSHA:      targetRepos[res.Path].Committish,
			}
		}
	}
	outputChan <- additionsResult{Additions: repoCommits}
}

// getSysctlDiff finds sysctl difference between the two builds.
// Returns a list of change lists:[[name, old-value, new-value], ...]
func GetSysctlDiff(bucket, sourceBoard, sourceMilestone, source, targetBoard, targetMilestone, target string) (
	[][]string, bool, bool) {
	sourceBuildNum, targetBuildNum := resolveImageName(source), resolveImageName(target)
	sourceChan := make(chan map[string]string)
	targetChan := make(chan map[string]string)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf("failed to create storage client (error: %s)", err)
		return [][]string{}, false, false
	}
	go fetchSysctlToMap(fmt.Sprintf("%s/%s-release/R%s-%s",
		bucket, sourceBoard, sourceMilestone, sourceBuildNum), sourceChan, client, ctx)
	go fetchSysctlToMap(fmt.Sprintf("%s/%s-release/R%s-%s",
		bucket, targetBoard, targetMilestone, targetBuildNum), targetChan, client, ctx)
	sourceSysctl := <-sourceChan
	targetSysctl := <-targetChan
	foundSource := false
	foundTarget := false
	// if either one of the sysctl file doesn't exist,
	// return an empty list.
	if len(sourceSysctl) > 0 {
		foundSource = true
	}
	if len(targetSysctl) > 0 {
		foundTarget = true
	}
	if !foundSource || !foundTarget {
		return [][]string{}, foundSource, foundTarget
	}

	changes := [][]string{}
	for newName, newValue := range targetSysctl {
		if oldValue, found := sourceSysctl[newName]; !found {
			changes = append(changes, []string{newName, "---", newValue})
		} else if oldValue != newValue {
			changes = append(changes, []string{newName, oldValue, newValue})
		}
	}
	for oldName, oldValue := range sourceSysctl {
		if _, found := targetSysctl[oldName]; !found {
			changes = append(changes, []string{oldName, oldValue, "---"})
		}
	}

	sort.SliceStable(changes, func(i, j int) bool {
		return changes[i][0] < changes[j][0]
	})

	return changes, foundSource, foundTarget
}

// fetchSysctlToMap fetches sysctl file from artifacts in GCS created
// by build-executor and map each line to a <parameter_name: value>
// pair.
func fetchSysctlToMap(path string, outputChan chan map[string]string, client *storage.Client, ctx context.Context) {
	// Some sysctl value changes are insignificant and should not be displayed.
	sysctlFilter := map[string]bool{
		"kernel.hostname":                  true,
		"kernel.version":                   true,
		"fs.dentry-state":                  true,
		"fs.file-nr":                       true,
		"fs.inode-nr":                      true,
		"fs.inode-state":                   true,
		"fs.quota.syncs":                   true,
		"kernel.ns_last_pid":               true,
		"kernel.pty.nr":                    true,
		"kernel.random.boot_id":            true,
		"kernel.random.entropy_avail":      true,
		"kernel.random.uuid":               true,
		"net.netfilter.nf_conntrack_count": true,
		"kernel.osrelease":                 true,
		"net.ipv4.tcp_fastopen_key":        true,
	}
	outMap := make(map[string]string)
	defer func() { outputChan <- outMap }()
	rc, err := client.Bucket(path).Object("sysctl_a.txt").NewReader(ctx)
	if err != nil {
		log.Errorf("failed to open %s at %s (error:%s)", "sysctl_a.txt", path, err)
		return
	}

	byteBuf, err := ioutil.ReadAll(rc)
	rc.Close()
	if err != nil {
		log.Errorf("failed to read sysctl file (error:%s)", err)
		return
	}
	separator := " = "
	for _, line := range strings.Split(string(byteBuf), "\n") {
		parts := strings.Split(line, separator)
		// Insignificant sysctl parameters are excluded.
		if _, found := sysctlFilter[parts[0]]; found {
			continue
		}
		// no value for this parameter
		if len(parts) == 2 && parts[1] == "" {
			outMap[parts[0]] = "---"
		} else {
			// assume the parameter name is before the first separator.
			outMap[parts[0]] = strings.Join(parts[1:], separator)
		}
	}
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
func Changelog(httpClient *http.Client, source, target, host, repo, croslandURL string, querySize int) (map[string]*RepoLog, map[string]*RepoLog, utils.ChangelogError) {
	if httpClient == nil {
		log.Error("httpClient is nil")
		return nil, nil, utils.InternalServerError
	}
	sourceBuildNum, targetBuildNum := resolveImageName(source), resolveImageName(target)
	log.Infof("Retrieving changelog between %s and %s\n", sourceBuildNum, targetBuildNum)
	clients := make(map[string]gitilesProto.GitilesClient)

	// Since the manifest file is always in the cos instance, add cos client
	// so that client knows what URL to use
	manifestClient, err := gitilesClient(httpClient, host)
	if err != nil {
		return nil, nil, err
	}
	sourceRepos, sourceErr := mappedManifest(manifestClient, repo, source, sourceBuildNum)
	targetRepos, targetErr := mappedManifest(manifestClient, repo, target, targetBuildNum)
	if sourceErr != nil && sourceErr.HTTPCode() == "404" && targetErr != nil && targetErr.HTTPCode() == "404" {
		return nil, nil, utils.BothBuildsNotFound(croslandURL, source, target, sourceBuildNum, targetBuildNum)
	} else if sourceErr != nil {
		return nil, nil, sourceErr
	} else if targetErr != nil {
		return nil, nil, targetErr
	}

	clients[host] = manifestClient
	err = createGitilesClients(clients, httpClient, sourceRepos)
	if err != nil {
		return nil, nil, err
	}
	err = createGitilesClients(clients, httpClient, targetRepos)
	if err != nil {
		return nil, nil, err
	}

	addChan := make(chan additionsResult, 1)
	missChan := make(chan additionsResult, 1)
	go additions(clients, sourceRepos, targetRepos, querySize, addChan)
	go additions(clients, targetRepos, sourceRepos, querySize, missChan)
	missRes := <-missChan
	if missRes.Err != nil {
		return nil, nil, missRes.Err
	}
	addRes := <-addChan
	if addRes.Err != nil {
		return nil, nil, addRes.Err
	}

	return addRes.Additions, missRes.Additions, nil
}
