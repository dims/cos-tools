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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cos.googlesource.com/cos/tools/src/pkg/changelog"
	"cos.googlesource.com/cos/tools/src/pkg/findbuild"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
)

const (
	subjectLen int = 100
)

var (
	internalGerritInstance         string
	internalFallbackGerritInstance string
	internalGoBInstance            string
	internalManifestRepo           string
	externalGerritInstance         string
	externalFallbackGerritInstance string
	externalGoBInstance            string
	externalManifestRepo           string
	fallbackRepoPrefix             string
	envQuerySize                   string

	staticBasePath          string
	indexTemplate           *template.Template
	changelogTemplate       *template.Template
	promptLoginTemplate     *template.Template
	locateBuildTemplate     *template.Template
	statusForbiddenTemplate *template.Template
	basicTextTemplate       *template.Template

	grpcCodeToHTTP = map[string]string{
		codes.Unknown.String():            "500",
		codes.InvalidArgument.String():    "400",
		codes.NotFound.String():           "404",
		codes.PermissionDenied.String():   "403",
		codes.Unauthenticated.String():    "401",
		codes.ResourceExhausted.String():  "429",
		codes.FailedPrecondition.String(): "400",
		codes.OutOfRange.String():         "400",
		codes.Internal.String():           "500",
		codes.Unavailable.String():        "503",
		codes.DataLoss.String():           "500",
	}
	httpCodeToHeader = map[string]string{
		"400": "400 Bad Request",
		"401": "401 Unauthorized",
		"403": "403 Forbidden",
		"404": "404 Not Found",
		"429": "429 Too Many Requests",
		"500": "500 Internal Server Error",
		"503": "503 Service Unavailable",
	}
	httpCodeToDesc = map[string]string{
		"400": "The request could not be understood.",
		"401": "You are currently unauthenticated. Please login to access this resource.",
		"403": "This account does not have access to internal repositories. Please retry with an authorized account, or select the external button to query from publically accessible builds.",
		"404": "The requested resource was not found. Please verify that you are using a valid input.",
		"429": "Our servers are currently experiencing heavy load. Please retry in a couple minutes.",
		"500": "Something went wrong on our end. Please try again later.",
		"503": "This service is temporarily offline. Please try again later.",
	}
	gitiles403Desc  = "unexpected HTTP 403 from Gitiles"
	gerritErrCodeRe = regexp.MustCompile("status code\\s*(\\d+)")
)

func init() {
	var err error
	client, err := secretmanager.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to setup client: %v", err)
	}
	internalGerritInstance, err = getSecret(client, os.Getenv("COS_INTERNAL_GERRIT_INSTANCE_NAME"))
	if err != nil {
		log.Fatalf("Failed to retrieve secret for COS_INTERNAL_GERRIT_INSTANCE_NAME with key name %s\n%v", os.Getenv("COS_INTERNAL_GERRIT_INSTANCE_NAME"), err)
	}
	internalFallbackGerritInstance, err = getSecret(client, os.Getenv("COS_INTERNAL_FALLBACK_GERRIT_INSTANCE_NAME"))
	if err != nil {
		log.Fatalf("Failed to retrieve secret for COS_INTERNAL_FALLBACK_GERRIT_INSTANCE_NAME with key name %s\n%v", os.Getenv("COS_INTERNAL_FALLBACK_GERRIT_INSTANCE_NAME"), err)
	}
	internalGoBInstance, err = getSecret(client, os.Getenv("COS_INTERNAL_GOB_INSTANCE_NAME"))
	if err != nil {
		log.Fatalf("Failed to retrieve secret for COS_INTERNAL_GOB_INSTANCE_NAME with key name %s\n%v", os.Getenv("COS_INTERNAL_GOB_INSTANCE_NAME"), err)
	}
	internalManifestRepo, err = getSecret(client, os.Getenv("COS_INTERNAL_MANIFEST_REPO_NAME"))
	if err != nil {
		log.Fatalf("Failed to retrieve secret for COS_INTERNAL_MANIFEST_REPO_NAME with key name %s\n%v", os.Getenv("COS_INTERNAL_MANIFEST_REPO_NAME"), err)
	}
	externalGerritInstance = os.Getenv("COS_EXTERNAL_GERRIT_INSTANCE")
	externalFallbackGerritInstance = os.Getenv("COS_EXTERNAL_FALLBACK_GERRIT_INSTANCE")
	externalGoBInstance = os.Getenv("COS_EXTERNAL_GOB_INSTANCE")
	externalManifestRepo = os.Getenv("COS_EXTERNAL_MANIFEST_REPO")
	fallbackRepoPrefix = os.Getenv("COS_FALLBACK_REPO_PREFIX")
	envQuerySize = getIntVerifiedEnv("CHANGELOG_QUERY_SIZE")
	staticBasePath = os.Getenv("STATIC_BASE_PATH")
	indexTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/index.html"))
	changelogTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/changelog.html"))
	locateBuildTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/locateBuild.html"))
	promptLoginTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/promptLogin.html"))
	basicTextTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/error.html"))
}

type changelogData struct {
	Instance  string
	Source    string
	Target    string
	Additions map[string]*changelog.RepoLog
	Removals  map[string]*changelog.RepoLog
	Internal  bool
}

type changelogPage struct {
	Source     string
	Target     string
	QuerySize  string
	RepoTables []*repoTable
	Internal   bool
}

type locateBuildPage struct {
	CL            string
	CLNum         string
	BuildNum      string
	GerritLink    string
	Internal      bool
}

type statusPage struct {
	ActivePage string
}

type basicTextPage struct {
	Header     string
	Body       string
	ActivePage string
}

type repoTable struct {
	Name          string
	Additions     []*repoTableEntry
	Removals      []*repoTableEntry
	AdditionsLink string
	RemovalsLink  string
}

type repoTableEntry struct {
	IsAddition    bool
	SHA           *shaAttr
	Subject       string
	Bugs          []*bugAttr
	AuthorName    string
	CommitterName string
	CommitTime    string
	ReleaseNote   string
}

type shaAttr struct {
	Name string
	URL  string
}

type bugAttr struct {
	Name string
	URL  string
}

// getIntVerifiedEnv retrieves an environment variable but checks that it can be
// converted to int first
func getIntVerifiedEnv(envName string) string {
	output := os.Getenv(envName)
	if _, err := strconv.Atoi(output); err != nil {
		log.Errorf("getEnvAsInt: Failed to parse env variable %s with value %s: %v",
			envName, os.Getenv(output), err)
	}
	return output
}

func unwrappedError(err error) error {
	innerErr := err
	for errors.Unwrap(innerErr) != nil {
		innerErr = errors.Unwrap(innerErr)
	}
	return innerErr
}

func gobCommitLink(instance, repo, SHA string) string {
	return fmt.Sprintf("https://%s/%s/+/%s", instance, repo, SHA)
}

func gobDiffLink(instance, repo, sourceSHA, targetSHA string) string {
	return fmt.Sprintf("https://%s/%s/+log/%s..%s?n=10000", instance, repo, sourceSHA, targetSHA)
}

func createRepoTableEntry(instance string, repo string, commit *changelog.Commit, isAddition bool) *repoTableEntry {
	entry := new(repoTableEntry)
	entry.IsAddition = isAddition
	entry.SHA = &shaAttr{Name: commit.SHA[:8], URL: gobCommitLink(instance, repo, commit.SHA)}
	entry.Subject = commit.Subject
	if len(entry.Subject) > subjectLen {
		entry.Subject = entry.Subject[:subjectLen]
	}
	entry.Bugs = make([]*bugAttr, len(commit.Bugs))
	for i, bugURL := range commit.Bugs {
		name := bugURL[strings.Index(bugURL, "/")+1:]
		entry.Bugs[i] = &bugAttr{Name: name, URL: "http://" + bugURL}
	}
	entry.AuthorName = commit.AuthorName
	entry.CommitterName = commit.CommitterName
	entry.CommitTime = commit.CommitTime
	entry.ReleaseNote = commit.ReleaseNote
	return entry
}

func createChangelogPage(data changelogData) *changelogPage {
	page := &changelogPage{Source: data.Source, Target: data.Target, QuerySize: envQuerySize, Internal: data.Internal}
	for repoName, repoLog := range data.Additions {
		table := &repoTable{Name: repoName}
		for _, commit := range repoLog.Commits {
			tableEntry := createRepoTableEntry(data.Instance, repoName, commit, true)
			table.Additions = append(table.Additions, tableEntry)
		}
		if _, ok := data.Removals[repoName]; ok {
			for _, commit := range data.Removals[repoName].Commits {
				tableEntry := createRepoTableEntry(data.Instance, repoName, commit, false)
				table.Removals = append(table.Removals, tableEntry)
			}
			if data.Removals[repoName].HasMoreCommits {
				table.RemovalsLink = gobDiffLink(data.Instance, repoName, repoLog.TargetSHA, repoLog.SourceSHA)
			}
		}
		if repoLog.HasMoreCommits {
			table.AdditionsLink = gobDiffLink(data.Instance, repoName, repoLog.SourceSHA, repoLog.TargetSHA)
		}
		page.RepoTables = append(page.RepoTables, table)
	}
	// Add remaining repos that had removals but no additions
	for repoName, repoLog := range data.Removals {
		if _, ok := data.Additions[repoName]; ok {
			continue
		}
		table := &repoTable{Name: repoName}
		for _, commit := range repoLog.Commits {
			tableEntry := createRepoTableEntry(data.Instance, repoName, commit, false)
			table.Removals = append(table.Removals, tableEntry)
		}
		page.RepoTables = append(page.RepoTables, table)
		if repoLog.HasMoreCommits {
			table.RemovalsLink = gobDiffLink(data.Instance, repoName, repoLog.TargetSHA, repoLog.SourceSHA)
		}
	}
	return page
}

func findBuildWithFallback(httpClient *http.Client, gerrit, fallbackGerrit, gob, repo, cl string, internal bool) (*findbuild.BuildResponse, bool, error) {
	didFallback := false
	request := &findbuild.BuildRequest{
		HTTPClient:   httpClient,
		GerritHost:   gerrit,
		GitilesHost:  gob,
		ManifestRepo: repo,
		RepoPrefix:   "",
		CL:           cl,
	}
	buildData, err := findbuild.FindBuild(request)
	innerErr := unwrappedError(err)
	if innerErr == findbuild.ErrorCLNotFound {
		log.Debugf("Cl %s not found in Gerrit instance, using fallback", cl)
		fallbackRequest := &findbuild.BuildRequest{
			HTTPClient:   httpClient,
			GerritHost:   fallbackGerrit,
			GitilesHost:  gob,
			ManifestRepo: repo,
			RepoPrefix:   fallbackRepoPrefix,
			CL:           cl,
		}
		buildData, err = findbuild.FindBuild(fallbackRequest)
		didFallback = true
	}
	return buildData, didFallback, err
}

// parseGRPCError retrieves the header and description to display for a gRPC error
func parseGRPCError(inputErr error) (string, string, error) {
	rpcStatus, ok := status.FromError(inputErr)
	if !ok {
		return "", "", fmt.Errorf("parseGRPCError: Error %v is not a gRPC error", inputErr)
	}
	code, text := rpcStatus.Code(), rpcStatus.Message()
	// RPC status code misclassifies 403 error as 500 error for Gitiles requests
	if text == gitiles403Desc {
		code = codes.PermissionDenied
	}
	if httpCode, ok := grpcCodeToHTTP[code.String()]; ok {
		if _, ok := httpCodeToHeader[httpCode]; !ok {
			log.Errorf("parseGRPCError: No error header mapping found for HTTP code %s", httpCode)
			return "", "", fmt.Errorf("parseGRPCError: No error header mapping found for HTTP code %s", httpCode)
		}
		if _, ok := httpCodeToDesc[httpCode]; !ok {
			log.Errorf("parseGRPCError: No error description mapping found for HTTP code %s", httpCode)
			return "", "", fmt.Errorf("parseGRPCError: No error description mapping found for HTTP code %s", httpCode)
		}
		return httpCodeToHeader[httpCode], httpCodeToDesc[httpCode], nil
	}
	return "", "", fmt.Errorf("parseGRPCError: gRPC error code %s not supported", code.String())
}

// parseGitilesError retrieves the header and description to display for a Gerrit error
func parseGerritError(inputErr error) (string, string, error) {
	matches := gerritErrCodeRe.FindStringSubmatch(inputErr.Error())
	if len(matches) != 2 {
		return "", "", fmt.Errorf("parseGerritError: error %v is not a Gerrit error", inputErr)
	}
	httpCode := matches[1]
	if _, ok := httpCodeToHeader[httpCode]; !ok {
		log.Errorf("parseGerritError: No error header mapping found for HTTP code %s", httpCode)
		return "", "", fmt.Errorf("parseGerritError: No error header mapping found for HTTP code %s", httpCode)
	}
	if _, ok := httpCodeToDesc[httpCode]; !ok {
		log.Errorf("parseGerritError: No error description mapping found for HTTP code %s", httpCode)
		return "", "", fmt.Errorf("parseGerritError: No error description mapping found for HTTP code %s", httpCode)
	}
	return httpCodeToHeader[httpCode], httpCodeToDesc[httpCode], nil
}

// handleError creates the error page for a given error
func handleError(w http.ResponseWriter, inputErr error, currPage string) {
	innerErr := unwrappedError(inputErr)
	header := "An error occurred while handling this request"
	body := innerErr.Error()
	if tmpHeader, tmpBody, err := parseGRPCError(innerErr); err == nil {
		header = tmpHeader
		body = tmpBody
	} else if tmpHeader, tmpBody, err := parseGerritError(innerErr); err == nil {
		header = tmpHeader
		body = tmpBody
	}
	basicTextTemplate.Execute(w, &basicTextPage{
		Header:     header,
		Body:       body,
		ActivePage: currPage,
	})
}

// HandleIndex serves the home page
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	indexTemplate.Execute(w, nil)
}

// HandleChangelog serves the changelog page
func HandleChangelog(w http.ResponseWriter, r *http.Request) {
	httpClient, err := HTTPClient(w, r, "/changelog/")
	if err != nil {
		log.Debug(err)
		err = promptLoginTemplate.Execute(w, &statusPage{ActivePage: "changelog"})
		if err != nil {
			log.Errorf("HandleChangelog: error executing promptLogin template: %v", err)
		}
		return
	}
	if err := r.ParseForm(); err != nil {
		err = changelogTemplate.Execute(w, &changelogPage{QuerySize: envQuerySize})
		if err != nil {
			log.Errorf("HandleChangelog: error executing locatebuild template: %v", err)
		}
		return
	}
	source := r.FormValue("source")
	target := r.FormValue("target")
	// If no source/target values specified in request, display empty changelog page
	if source == "" || target == "" {
		err = changelogTemplate.Execute(w, &changelogPage{QuerySize: envQuerySize, Internal: true})
		if err != nil {
			log.Errorf("HandleChangelog: error executing locatebuild template: %v", err)
		}
		return
	}
	querySize, err := strconv.Atoi(r.FormValue("n"))
	if err != nil {
		querySize, _ = strconv.Atoi(envQuerySize)
	}
	internal, instance, manifestRepo := false, externalGoBInstance, externalManifestRepo
	if r.FormValue("internal") == "true" {
		internal, instance, manifestRepo = true, internalGoBInstance, internalManifestRepo
	}
	added, removed, err := changelog.Changelog(httpClient, source, target, instance, manifestRepo, querySize)
	if err != nil {
		log.Errorf("HandleChangelog: error retrieving changelog between builds %s and %s on GoB instance: %s with manifest repository: %s\n%v\n",
			source, target, externalGoBInstance, externalManifestRepo, err)
		handleError(w, err, "/changelog/")
		return
	}
	page := createChangelogPage(changelogData{
		Instance:  instance,
		Source:    source,
		Target:    target,
		Additions: added,
		Removals:  removed,
		Internal:  internal,
	})
	err = changelogTemplate.Execute(w, page)
	if err != nil {
		log.Errorf("HandleChangelog: error executing changelog template: %v", err)
	}
}

// HandleLocateBuild serves the Locate CL page
func HandleLocateBuild(w http.ResponseWriter, r *http.Request) {
	httpClient, err := HTTPClient(w, r, "/locatebuild/")
	// Require login to access if no session found
	if err != nil {
		log.Debug(err)
		err = promptLoginTemplate.Execute(w, &statusPage{ActivePage: "locatebuild"})
		if err != nil {
			log.Errorf("HandleLocateBuild: error executing promptLogin template: %v", err)
		}
		return
	}
	if err := r.ParseForm(); err != nil {
		err = locateBuildTemplate.Execute(w, &locateBuildPage{Internal: true})
		if err != nil {
			log.Errorf("HandleLocateBuild: error executing locatebuild template: %v", err)
		}
		return
	}
	cl := r.FormValue("cl")
	// If no CL value specified in request, display empty CL form
	if cl == "" {
		err = locateBuildTemplate.Execute(w, &locateBuildPage{Internal: true})
		if err != nil {
			log.Errorf("HandleLocateBuild: error executing locatebuild template: %v", err)
		}
		return
	}
	internal, gerrit, fallbackGerrit, gob, repo := false, externalGerritInstance, externalFallbackGerritInstance, externalGoBInstance, externalManifestRepo
	if r.FormValue("internal") == "true" {
		internal, gerrit, fallbackGerrit, gob, repo = true, internalGerritInstance, internalFallbackGerritInstance, internalGoBInstance, internalManifestRepo
	}
	buildData, didFallback, err := findBuildWithFallback(httpClient, gerrit, fallbackGerrit, gob, repo, cl, internal)
	if err != nil {
		log.Errorf("Handlelocatebuild: error retrieving build for CL %s with internal set to %t\n%v", cl, internal, err)
		handleError(w, err, "/locatebuild/")
		return
	}
	var gerritLink string
	if didFallback {
		gerritLink = fallbackGerrit + "/q/" + buildData.CLNum
	} else {
		gerritLink = gerrit + "/q/" + buildData.CLNum
	}
	page := &locateBuildPage{
		CL:         cl,
		CLNum:      buildData.CLNum,
		BuildNum:   buildData.BuildNum,
		Internal:   internal,
		GerritLink: gerritLink,
	}
	err = locateBuildTemplate.Execute(w, page)
	if err != nil {
		log.Errorf("HandleLocateBuild: error executing locatebuild template: %v", err)
	}
}
