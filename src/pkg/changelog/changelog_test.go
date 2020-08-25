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
	"context"
	"fmt"
	"net/http"
	"testing"

	"go.chromium.org/luci/common/api/gerrit"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const cosInstance = "cos.googlesource.com"
const defaultManifestRepo = "cos/manifest-snapshots"

func getHTTPClient() (*http.Client, error) {
	creds, err := google.FindDefaultCredentials(context.Background(), gerrit.OAuthScope)
	if err != nil || len(creds.JSON) == 0 {
		return nil, fmt.Errorf("no application default credentials found - run `gcloud auth application-default login` and try again")
	}
	return oauth2.NewClient(oauth2.NoContext, creds.TokenSource), nil
}

func commitsMatch(commits []*Commit, expectedCommits []string) bool {
	if len(commits) != len(expectedCommits) {
		return false
	}
	for i, commit := range commits {
		if commit == nil {
			return false
		}
		if commit.SHA != expectedCommits[i] {
			return false
		}
	}
	return true
}

func repoListInLog(log map[string]*RepoLog, check []string) error {
	for _, check := range check {
		if log, ok := log[check]; !ok || len(log.Commits) == 0 {
			return fmt.Errorf("Repo path %s not in log", check)
		}
	}
	return nil
}

func TestChangelog(t *testing.T) {
	httpClient, err := getHTTPClient()

	// Test invalid source
	additions, removals, err := Changelog(httpClient, "15", "15043.0.0", cosInstance, defaultManifestRepo, -1)
	if additions != nil {
		t.Errorf("changelog failed, expected nil additions, got %v", additions)
	} else if removals != nil {
		t.Errorf("changelog failed, expected nil removals, got %v", removals)
	} else if err == nil {
		t.Errorf("changelog failed, expected error, got nil")
	}

	// Test invalid target
	additions, removals, err = Changelog(httpClient, "15043.0.0", "abx", cosInstance, defaultManifestRepo, -1)
	if additions != nil {
		t.Errorf("changelog failed, expected nil additions, got %v", additions)
	} else if removals != nil {
		t.Errorf("changelog failed, expected nil removals, got %v", removals)
	} else if err == nil {
		t.Errorf("changelog failed, expected error, got nil")
	}

	// Test invalid instance
	additions, removals, err = Changelog(httpClient, "15036.0.0", "15041.0.0", "com", defaultManifestRepo, -1)
	if additions != nil {
		t.Errorf("changelog failed, expected nil additions, got %v", additions)
	} else if removals != nil {
		t.Errorf("changelog failed, expected nil removals, got %v", removals)
	} else if err == nil {
		t.Errorf("changelog failed, expected error, got nil")
	}

	// Test invalid manifest repo
	additions, removals, err = Changelog(httpClient, "15036.0.0", "15041.0.0", cosInstance, "cos/not-a-repo", -1)
	if additions != nil {
		t.Errorf("changelog failed, expected nil additions, got %v", additions)
	} else if removals != nil {
		t.Errorf("changelog failed, expected nil removals, got %v", removals)
	} else if err == nil {
		t.Errorf("changelog failed, expected error, got nil")
	}

	// Test build number higher than latest release
	additions, removals, err = Changelog(httpClient, "15036.0.0", "99999.0.0", cosInstance, defaultManifestRepo, -1)
	if additions != nil {
		t.Errorf("changelog failed, expected nil additions, got %v", additions)
	} else if removals != nil {
		t.Errorf("changelog failed, expected nil removals, got %v", removals)
	} else if err == nil {
		t.Errorf("changelog failed, expected error, got nil")
	}

	// Test manifest with remote urls specified and no default URL
	additions, removals, err = Changelog(httpClient, "1.0.0", "2.0.0", cosInstance, defaultManifestRepo, -1)
	if additions == nil {
		t.Errorf("changelog failed, expected additions, got nil")
	} else if removals == nil {
		t.Errorf("changelog failed, expected removals, got nil")
	} else if err != nil {
		t.Errorf("changelog failed, expected no error, got %v", err)
	}

	// Test 1 build number difference with only 1 repo change between them
	// Ensure that commits are correctly inserted in proper order
	// Check that changelog metadata correctly populated
	source := "15050.0.0"
	target := "15051.0.0"
	expectedCommits := []string{
		"6201c49afe667c8fa7796608a4d7162bb3f7f4f4",
		"a8bcf0feaa0e3c0131a888fcd9d0dcbbe8c3850c",
		"5e3ef32e062fb227aaa6b47138950557ec91d23e",
		"654ed08e8a349e7199eb3a80b6d7704a20ff8ec4",
		"d5c0e74fbb2a50517a1249cbbec4dcee3d049883",
		"cd226061776dad6c0e35323f407eaa138795f4cc",
		"4351d0dc5480e941fac96cb0ec898a87171eadda",
		"cdbcf507749a86acad3e8787ffb3c3356ed76b3a",
		"4fdd7f397bc09924e91f475d3ed55bb5a302bdaf",
		"3adae69de78875a8d33061205357388a513ea51d",
		"5fd85ec937d362984e5108762e8b5e20105a4219",
		"03b6099c920c1b3cb4cbda2172089e80b4d4be6e",
		"1febb203aaf99f00e5d9d80d965726458ba8348f",
		"2de610687308b6ea00d9ac6190d83f0edb2a46b4",
		"db3083c438442ea6ab34e84404b4602618d2e07b",
		"13eb9486f2bf43d56ce58695df8461099fd7c314",
		"12b8a449ef93289674d93f437c19a06530c2c966",
		"6d9752b0abeeaf7438ab08ea7ff5b0f76c2dacca",
		"8555ba160a5eee0be464b25a07abc6031dc9159c",
		"a8c1c3c2971acc03f4246c20b1ddd5bb5376ded3",
		"762495e014eaa74e3aa4d83caaaa778fcfb968a2",
		"784782cd8c1d846c17541a3e527ad56857fe2e91",
		"7c6916858860715db25eaadc2b3ec81865304095",
		"76cc8bf290a133ee821a8a2b14207150de9a7803",
		"dc07ec7806f249fdb0b7bda68c687a87b311c952",
		"f18ad3b35466354d5a0e166008070f54a06759a6",
		"34f008f664e11b6df2f06735b6db6d6a42804d25",
		"a24eee7a6b6caed0448365e548e92724069a8448",
		"64ddf2924656f07bd63269524ed1731a2357b82f",
		"e40d4ce60313cd28ebf1c376860402f9b3d373cd",
		"3018e2531a1f0f22c4d053ed0b8a5cc86ad81319",
		"668cd418350d03e1535c7862ebe93801ace0b1c6",
		"fabf26e3eab2af24371c48e19062d7c8df34bd9f",
		"7b38982caecbeb16520b4dd84422ecad0edaf772",
		"658380877ca2eedc3cd80d3b6daafa24ab96a261",
		"63dee6c8cd318dfa20cfddf2e72243873e816046",
		"bc194a3ce16407015da5bc8d46df55231cf4d625",
		"ff75e90067c7c535116cb5566ebc14451785b36a",
		"c64b1cc6b930024e77425fd105716ade26d0524c",
		"d5123111900fd70d85b7acf5809df701da24f1ea",
		"c617b261c68b52b0abefc0635c1ea03c4cb0cb11",
		"a2619465e4eca49692d832b593cb205118042bc9",
		"6f6451dd56a7fad25b2e8b31a053275adb2008a4",
		"68d5d3901d5c3df44e3be8c3fac0c6b1e90d780c",
		"308882e4e837f231e3ad0f37fd143cee419d816f",
		"1cb20f5aa5a82a412d97fad7b9c13c87c9381f14",
		"f6c0c6f1618676519efd74c8f946e191472b6a4e",
		"dea6ca48a629e80cc2ffbf203c9cc1855a28a47e",
		"fa0115b220b3471a1542b3b66463f9ec80c8c7f0",
		"b815d624f7715ab51379e8a913c280cac1eafde4",
		"39fe5d201b87e02baedf4da8b02523571c4ccbcc",
		"58aff81e0829100cc9d3239791573300e2d2398d",
		"cd570b8e278aca36f166eb84b5003eaee3c03ecc",
		"50f9936fe8ab106d2716e007a342860c695f7822",
		"2a1e98d6c3dca9b52bcb7b02c7a242c10c0a0de9",
		"b9fe6cc174f215d576954e6b2c93bc4de8ba2c34",
		"f78d275ec9d0c4061f75ae2f97f958657a71ebd1",
		"315ea4a344e3f8b300e8c3e48fafc21eaee767fe",
		"1c9392eb35c68ca38a1f0178cd191f07d387f52d",
		"9cd44834d383b5414bd9bac873e9c620a67eff1d",
		"e0f3f79316591affedeaf2702a350d3512bd6a69",
		"148bba54f3762b23a79057825a763c1132bd1d55",
		"48ce30dd18de40852cea15dccaaa833b4017ae10",
		"474e61f82f79d9779b0e2c3bc63d920d9f75b5b6",
		"b93f0e4f3edbe3e64b0128db38ee231a737f06c9",
		"714065afa108556b6ff43ff312b731c239d6e551",
		"45a780a84daa27307addd836df94afa2c70dccb6",
		"0df346778d142f9c6bf221d67bdac96d9d636408",
		"6ad098080fb6437da98511e56026476fa71cce87",
		"3f2915159ab1e42b258ee78d2a71f2dc59d51d35",
		"1d5a9ebc23d1455966963a042bd610fdb38cd705",
		"e31b072bbc2d83db107d913a3f32d907de119ca2",
		"6da63745bd4318577ab8937100871e654df04cb3",
		"d5a54c19f7bf1f8250bc5ac779f80450764e836e",
		"54c59bdcf9965dbb77a6dd9682f255e21e4821a1",
		"67b538de711500bfb1ed5d322e916e8cd3f74700",
		"2814ccbb44a3d19cb4d696705794ced3beb31ef3",
		"deb92542c03e9096fe37d8833532a50a6bb1df3c",
		"d2b9b62c2ad5440005b72826bb55a36dfc115ac2",
		"da9cd84436f716c3c7a6d90e820afb87a9a218b9",
		"d0937f57cd2904df1af7449f32c75aaadaeac2a2",
		"65441913baef06967e59158f3848e41dce18b43a",
		"7cc03e836eba4d13526969b84aaa8dd61d8b6216",
		"dff08d118cab7f8416b8f171aac91b8ca3f6b44d",
		"aedb933f853499a0c736deb2d2ab899b607aacee",
		"aa592bf7b0b7b13eee2b20fa54fd81e11e96cf56",
		"f495c107eefc879b10fdf2e3a2a0155259210dba",
		"7e4e0964a1426d46cdbcbccd861cee7a106a9430",
		"d0ca437a1ed89e2adbd6b2d1bd572b475cd1d8ec",
		"0dce9e5070718b7ba950f0b6575bb3bbd0e362bc",
		"ddd73889c36e93c6128a4d791b6d673cd655447e",
		"04e70ee7abbb702e4939fef98d50b5e6cc018ccd",
		"86da591dd3d8515ebf4d1eebc68a61092ad13e95",
		"8676fbad9fa41e0d0f69dafb2b4f8bd4b5a3b3cc",
		"b8b3a8cc67fcdf58d495489c19e5d3aa23d22563",
		"7441c2cf859b84f7cedff8946dbd0c3dc7ef956b",
		"7f3e0778e212c8a22f8262e2819a6aebfca8b879",
		"a82b808965dbe304e0a95cb9534b09b3b5c0486a",
		"0388f30783e2454ea9f0c3978f92c797fc0bdf20",
		"67f6e97cee8a5b33f8e27b4d2426fb009c0ae435",
		"094bef7b6bd0c034ea19aa3cb9744ca35998ecc8",
		"ec07a4f7eb15d867e453c8c8991656b361a29882",
		"0a304d6481d01d774fe97f31c9574c970fdb532f",
		"3f77b91ad1abb2d2074286635927fa6472eb0a2e",
		"ca721a37ec8edc8f1b8aeb4c143aa936dc032ac1",
		"c0b7d2df81ae29869f9d7a1874b741eeec0d5d18",
		"9bc12bb411f357188d008864f80dfba43210b9d8",
		"bf0dd3757826b9bc9d7082f5f749ff7615d4bcb3",
	}
	additions, removals, err = Changelog(httpClient, source, target, cosInstance, defaultManifestRepo, -1)
	if err != nil {
		t.Errorf("changelog failed, expected no error, got %v", err)
	} else if len(removals) != 0 {
		t.Errorf("changelog failed, expected empty removals list, got %v", removals)
	} else if len(additions) != 1 {
		t.Errorf("changelog failed, expected only 1 repo in additions, got %v", additions)
	}
	boardOverlayLog := additions["src/overlays"]
	if boardOverlayLog == nil {
		t.Errorf("Changelog failed, expected src/overlays in changelog, got nil")
	} else if changes := boardOverlayLog.Commits; len(changes) != 108 {
		t.Errorf("Changelog failed, expected 108 changes for \"src/overlays\", got %d", len(changes))
	} else if !commitsMatch(boardOverlayLog.Commits, expectedCommits) {
		t.Errorf("changelog failed, Changelog output does not match expected commits or is not sorted")
	} else if boardOverlayLog.SourceSHA != "612ca5ef5455534127d008e08c65aa29a2fd97a5" {
		t.Errorf("changelog failed, expected SourceSHA \"612ca5ef5455534127d008e08c65aa29a2fd97a5\", got %s", boardOverlayLog.SourceSHA)
	} else if boardOverlayLog.TargetSHA != "6201c49afe667c8fa7796608a4d7162bb3f7f4f4" {
		t.Errorf("changelog failed, expected SourceSHA \"6201c49afe667c8fa7796608a4d7162bb3f7f4f4\", got %s", boardOverlayLog.TargetSHA)
	}

	// Test build numbers further apart from each other with multiple repo differences
	// Also ensures that removals are correctly populated
	source = "15030.0.0"
	target = "15056.0.0"
	additionRepos := []string{
		"src/scripts",
		"src/platform/vboot_reference",
		"src/platform/dev",
		"chromite",
		"src/third_party/autotest/files",
		"src/third_party/eclass-overlay",
		"src/third_party/toolchain-utils",
		"src/platform/crostestutils",
		"src/third_party/coreboot",
		"src/third_party/kernel/v5.4",
		"src/overlays",
		"src/chromium/depot_tools",
		"src/third_party/portage-stable",
		"chromite/infra/proto",
		"manifest",
		"src/platform2",
		"src/third_party/chromiumos-overlay",
	}
	additions, removals, err = Changelog(httpClient, source, target, cosInstance, defaultManifestRepo, -1)
	if err != nil {
		t.Errorf("changelog failed, expected no error, got %v", err)
	}
	if len(removals) != 0 {
		t.Errorf("Changelog failed, expected empty removals, got %v", removals)
	} else if err := repoListInLog(additions, additionRepos); err != nil {
		t.Errorf("Changelog failed, additions repo output does not match expected repos: %v", err)
	}

	// Test changelog returns correct output when given a querySize instead of -1
	source = "15030.0.0"
	target = "15050.0.0"
	querySize := 50
	additions, removals, err = Changelog(httpClient, source, target, cosInstance, defaultManifestRepo, querySize)
	if err != nil {
		t.Errorf("changelog failed, expected no error, got %v", err)
	} else if additions == nil {
		t.Errorf("changelog failed, non-empty expected additions, got nil")
	} else if removals == nil {
		t.Errorf("Changelog failed, non-empty expected removals, got nil")
	} else if _, ok := additions["src/third_party/kernel/v5.4"]; !ok {
		t.Errorf("Changelog failed, expected repo: src/third_party/kernel/v4.19 in additions")
	}
	for repoName, repoLog := range additions {
		if repoLog.Commits == nil || len(repoLog.Commits) == 0 {
			t.Errorf("changelog failed, expected non-empty additions commits, got nil or empty commits")
		}
		if len(repoLog.Commits) > querySize {
			t.Errorf("Changelog failed, expected %d commits for repo: %s, got: %d", querySize, repoName, len(repoLog.Commits))
		} else if repoName == "src/third_party/kernel/v5.4" && !repoLog.HasMoreCommits {
			t.Errorf("Changelog failed, expected HasMoreCommits = True for repo: src/third_party/kernel/v5.4, got False")
		} else if repoLog.HasMoreCommits && len(repoLog.Commits) < querySize {
			t.Errorf("changelog failed, expected HasMoreCommits = False for repo: %s with %d commits, got True", repoName, len(repoLog.Commits))
		}
	}

	// Test changelog handles manifest with non-matching repositories
	source = "12871.1177.0"
	target = "12871.1179.0"
	additions, removals, err = Changelog(httpClient, source, target, cosInstance, defaultManifestRepo, querySize)
	if err != nil {
		t.Errorf("changelog failed, expected no error, got %v", err)
	} else if len(removals) != 0 {
		t.Errorf("Changelog failed, expected empty removals, got %v", removals)
	} else if _, ok := additions["src/platform/cobble"]; !ok {
		t.Errorf("Changelog failed, expected repo: src/third_party/kernel/v4.19 in additions")
	}
	for repoName, repoLog := range additions {
		if repoLog.Commits == nil || len(repoLog.Commits) == 0 {
			t.Errorf("Changelog failed, expected non-empty additions commits, got nil or empty commits")
		} else if repoName == "src/platform/cobble" {
			if repoLog.HasMoreCommits {
				t.Errorf("Changelog failed, expected hasMoreCommits = false for repo: src/platform/cobble, got true")
			} else if repoLog.SourceSHA != "" {
				t.Errorf("Changelog failed, expected empty SourceSHA for src/platform/cobble, got %s", repoLog.SourceSHA)
			} else if repoLog.TargetSHA != "4ab43f1f86b7099b8ad75cf9615ea1fa155bbd7d" {
				t.Errorf("Changelog failed, expected TargetSHA: \"4ab43f1f86b7099b8ad75cf9615ea1fa155bbd7d\" for src/platform/cobble, got %s", repoLog.TargetSHA)
			}
		}
	}

	// Test changelog handles new repository addition
	source = "12871.1179.0"
	target = "12871.1177.0"
	additions, removals, err = Changelog(httpClient, source, target, cosInstance, defaultManifestRepo, querySize)
	if err != nil {
		t.Errorf("changelog failed, expected no error, got %v", err)
	} else if len(additions) != 0 {
		t.Errorf("Changelog failed, expected empty additions, got %v", additions)
	} else if _, ok := removals["src/platform/cobble"]; !ok {
		t.Errorf("Changelog failed, expected repo: src/third_party/kernel/v4.19 in additions")
	}
	for repoName, repoLog := range removals {
		if repoLog.Commits == nil || len(repoLog.Commits) == 0 {
			t.Errorf("Changelog failed, expected non-empty additions commits, got nil or empty commits")
		} else if repoName == "src/platform/cobble" {
			if repoLog.HasMoreCommits {
				t.Errorf("Changelog failed, expected hasMoreCommits = false for repo: src/platform/cobble, got true")
			} else if repoLog.SourceSHA != "" {
				t.Errorf("Changelog failed, expected empty SourceSHA for src/platform/cobble, got %s", repoLog.SourceSHA)
			} else if repoLog.TargetSHA != "4ab43f1f86b7099b8ad75cf9615ea1fa155bbd7d" {
				t.Errorf("Changelog failed, expected TargetSHA: \"4ab43f1f86b7099b8ad75cf9615ea1fa155bbd7d\" for src/platform/cobble, got %s", repoLog.TargetSHA)
			}
		}
	}

	// Test with different release branches
	source = "13310.1035.0"
	target = "15000.0.0"
	additions, removals, err = Changelog(httpClient, source, target, cosInstance, defaultManifestRepo, querySize)
	if err != nil {
		t.Errorf("Changelog failed, expected no error, got %v", err)
	} else if len(additions) == 0 {
		t.Errorf("Changelog failed, expected non-empty additions, got %v", additions)
	} else if len(removals) == 0 {
		t.Errorf("Changelog failed, expected non-empty removals, got %v", removals)
	}
}
