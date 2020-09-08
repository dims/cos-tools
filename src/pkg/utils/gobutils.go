package utils

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"go.chromium.org/luci/common/proto/git"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

const (
	manifestFileName string = "snapshot.xml"

	// These constants are used for exponential increase in Gitiles request size.
	defaultPageSize          = 100
	pageSizeGrowthMultiplier = 5
	maxPageSize              = 10000

	// Maximum time to wait for a response from a Gitiles request
	requestMaxAge = 2 * time.Minute
)

// limitPageSize will restrict a request page size to min of pageSize (which grows exponentially)
// or remaining request size
func limitPageSize(pageSize, querySize int, noLimit bool) int {
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if noLimit || pageSize <= querySize {
		return pageSize
	}
	return querySize
}

// DownloadManifest retrieves a manifest file from Git on Borg for a specific
// build number
func DownloadManifest(client gitilesProto.GitilesClient, manifestRepo, buildNum string) (*gitilesProto.DownloadFileResponse, error) {
	log.Debugf("Downloading manifest file for build %s", buildNum)
	request := gitilesProto.DownloadFileRequest{
		Project:    manifestRepo,
		Committish: "refs/tags/" + buildNum,
		Path:       manifestFileName,
		Format:     1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestMaxAge)
	defer cancel()
	response, err := client.DownloadFile(ctx, &request)
	return response, err
}

func nextCommits(client gitilesProto.GitilesClient, repo string, committish string, ancestor string, nextToken string, pageSize int) (*gitilesProto.LogResponse, error) {
	request := gitilesProto.LogRequest{
		Project:            repo,
		Committish:         committish,
		ExcludeAncestorsOf: ancestor,
		PageToken:          nextToken,
		PageSize:           int32(pageSize),
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestMaxAge)
	defer cancel()
	return client.Log(ctx, &request)
}

// Commits retrieves querySize commits that occur between a committish and an ancestor
// for a given repository. Returns a list of commits and a bool that is set to true
// if there are more than querySize commits between the two provided committishs.
func Commits(client gitilesProto.GitilesClient, repo string, committish string, ancestor string, querySize int) ([]*git.Commit, bool, error) {
	log.Debugf("Fetching changelog for repo: %s on committish %s\n", repo, committish)
	if querySize < -1 {
		return nil, false, fmt.Errorf("commits: %d is not a valid querySize. Please specify a positive querySize, or -1 for all commits", querySize)
	}
	start := time.Now()

	noLimit := querySize == -1
	pageSize := limitPageSize(defaultPageSize, querySize, noLimit)
	querySize -= pageSize
	response, err := nextCommits(client, repo, committish, ancestor, "", pageSize)
	if err != nil {
		return nil, false, fmt.Errorf("commits: Error retrieving commits for repo %s with committish %s and ancestor %s:\n%w", repo, committish, ancestor, err)
	}

	// No nextPageToken means there were less than <defaultPageSize> commits total.
	// We can immediately return.
	if response.NextPageToken == "" {
		log.Debugf("Retrieved %d commits from %s in %s\n", len(response.Log), repo, time.Since(start))
		return response.Log, false, nil
	}
	// Retrieve remaining commits using exponential increase in pageSize.
	allCommits := response.Log
	for (noLimit || querySize > 0) && response.NextPageToken != "" {
		if pageSize < maxPageSize {
			pageSize *= pageSizeGrowthMultiplier
		}
		pageSize = limitPageSize(pageSize, querySize, noLimit)
		querySize -= pageSize
		response, err = nextCommits(client, repo, committish, ancestor, response.NextPageToken, pageSize)
		if err != nil {
			return nil, false, fmt.Errorf("commits: Error retrieving next page commits for repo %s with committish %s and ancestor %s:\n%w", repo, committish, ancestor, err)
		}
		allCommits = append(allCommits, response.Log...)
	}
	log.Debugf("Retrieved %d commits from %s in %s\n", len(allCommits), repo, time.Since(start))
	return allCommits, response.NextPageToken != "", nil
}
