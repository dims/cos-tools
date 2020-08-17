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
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cos.googlesource.com/cos/tools/src/pkg/changelog"

	log "github.com/sirupsen/logrus"
)

const (
	subjectLen int = 100
)

var (
	internalInstance     string
	internalManifestRepo string
	externalInstance     string
	externalManifestRepo string
	envQuerySize         string
	staticBasePath       string
	indexTemplate        *template.Template
	changelogTemplate    *template.Template
	promptLoginTemplate  *template.Template
)

func init() {
	var err error
	client, err := secretmanager.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to setup client: %v", err)
	}
	internalInstance, err = getSecret(client, os.Getenv("COS_INTERNAL_GOB_INSTANCE_NAME"))
	if err != nil {
		log.Fatalf("Failed to retrieve secret for COS_INTERNAL_GOB_INSTANCE_NAME with key name %s\n%v", os.Getenv("COS_INTERNAL_GOB_INSTANCE_NAME"), err)
	}
	internalManifestRepo, err = getSecret(client, os.Getenv("COS_INTERNAL_MANIFEST_REPO_NAME"))
	if err != nil {
		log.Fatalf("Failed to retrieve secret for COS_INTERNAL_MANIFEST_REPO_NAME with key name %s\n%v", os.Getenv("COS_INTERNAL_MANIFEST_REPO_NAME"), err)
	}
	externalInstance = os.Getenv("COS_EXTERNAL_GOB_INSTANCE")
	externalManifestRepo = os.Getenv("COS_EXTERNAL_MANIFEST_REPO")
	envQuerySize = getIntVerifiedEnv("CHANGELOG_QUERY_SIZE")
	staticBasePath = os.Getenv("STATIC_BASE_PATH")
	indexTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/index.html"))
	changelogTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/changelog.html"))
	promptLoginTemplate = template.Must(template.ParseFiles(staticBasePath + "templates/promptLogin.html"))
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

type promptLoginPage struct {
	ActivePage string
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

// HandleIndex serves the home page
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	indexTemplate.Execute(w, nil)
}

// HandleChangelog serves the changelog page
func HandleChangelog(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		changelogTemplate.Execute(w, &changelogPage{QuerySize: envQuerySize})
		return
	}
	httpClient, err := HTTPClient(w, r, "/changelog/")
	if err != nil {
		log.Debug(err)
		err = promptLoginTemplate.Execute(w, &promptLoginPage{ActivePage: "changelog"})
		if err != nil {
			log.Errorf("HandleChangelog: error executing promptLogin template: %v", err)
		}
		return
	}
	source := r.FormValue("source")
	target := r.FormValue("target")
	// If no source/target values specified in request, display empty changelog page
	if source == "" || target == "" {
		changelogTemplate.Execute(w, &changelogPage{QuerySize: envQuerySize, Internal: true})
		return
	}
	querySize, err := strconv.Atoi(r.FormValue("n"))
	if err != nil {
		querySize, _ = strconv.Atoi(envQuerySize)
	}
	internal, instance, manifestRepo := false, externalInstance, externalManifestRepo
	if r.FormValue("internal") == "true" {
		internal, instance, manifestRepo = true, internalInstance, internalManifestRepo
	}
	added, removed, err := changelog.Changelog(httpClient, source, target, instance, manifestRepo, querySize)
	if err != nil {
		log.Errorf("HandleChangelog: error retrieving changelog between builds %s and %s on GoB instance: %s with manifest repository: %s\n%v\n",
			source, target, externalInstance, externalManifestRepo, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
