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
	"math/rand"
	"net/http"
	"os"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/gorilla/sessions"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

const (
	// Session variables
	sessionName      = "changelog"
	sessionKeyLength = 32

	// Oauth state generation variables
	oauthStateCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890"
	oauthStateLength  = 16
)

var config = &oauth2.Config{
	ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
	ClientSecret: "",
	Endpoint:     google.Endpoint,
	RedirectURL:  "https://cos-oss-interns-playground.uc.r.appspot.com/oauth2callback/",
	Scopes:       []string{"https://www.googleapis.com/auth/gerritcodereview"},
}
var store *sessions.CookieStore
var projectID = os.Getenv("COS_CHANGELOG_PROJECT_ID")
var clientSecretName = os.Getenv("COS_CHANGELOG_CLIENT_SECRET_NAME")
var sessionSecretName = os.Getenv("COS_CHANGELOG_SESSION_SECRET_NAME")

func init() {
	var err error
	client, err := secretmanager.NewClient(context.Background())
	if err != nil {
		log.Fatalf("failed to setup client: %v", err)
	}
	config.ClientSecret, err = getSecret(client, clientSecretName)
	if err != nil {
		log.Fatalf("failed to retrieve secret: %s\n%v", clientSecretName, err)
	}

	sessionSecret, err := getSecret(client, sessionSecretName)
	if err != nil {
		log.Fatalf("failed to retrieve secret :%s\n%v", sessionSecretName, err)
	}
	store = sessions.NewCookieStore([]byte(sessionSecret))
}

// Retrieve secrets stored in Gcloud Secret Manager
func getSecret(client *secretmanager.Client, secretName string) (string, error) {
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
	}
	result, err := client.AccessSecretVersion(context.Background(), accessRequest)
	if err != nil {
		return "", fmt.Errorf("failed to access secret at %s: %v", accessRequest.Name, err)
	}
	return string(result.Payload.Data), nil
}

func randomString(stringSize int, suffix string) string {
	randWithSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
	stateArr := make([]byte, stringSize)
	for i := range stateArr {
		stateArr[i] = oauthStateCharset[randWithSeed.Intn(len(oauthStateCharset))]
	}
	return string(stateArr) + suffix
}

func returnURLFromState(state string) string {
	return state[oauthStateLength:]
}

// tokenExpired indicates whether the Oauth token associated with a request is expired
func tokenExpired(r *http.Request) bool {
	var parsedExpiry time.Time
	session, _ := store.Get(r, sessionName)
	parsedExpiry, err := time.Parse(time.RFC3339, session.Values["expiry"].(string))
	return err != nil || parsedExpiry.Before(time.Now())
}

// GetLoginURL returns a login URL to redirect the user to
func GetLoginURL(redirect string, auto bool) string {
	return fmt.Sprintf("/login/?redirect=%s&auto=%v", redirect, auto)
}

// SignedIn returns a bool indicating if the current request is signed in
func SignedIn(r *http.Request) bool {
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return false
	}
	for _, key := range []string{"accessToken", "refreshToken", "tokenType", "expiry"} {
		if val, ok := session.Values[key]; !ok || val == nil {
			return false
		}
	}
	return true
}

// RequireToken will check if the user has a valid, unexpired Oauth token.
// If not, it will initiate the Oauth flow.
// Returns a bool indicating if the user was redirected
func RequireToken(w http.ResponseWriter, r *http.Request, activePage string) bool {
	if !SignedIn(r) {
		err := promptLoginTemplate.Execute(w, &statusPage{ActivePage: activePage})
		if err != nil {
			log.Errorf("error executing promptLogin template: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true
	}
	// If token is expired, auto refresh instead of prompting sign in
	if tokenExpired(r) {
		loginURL := GetLoginURL(activePage, true)
		http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
		return true
	}
	return false
}

// HTTPClient creates an authorized HTTP Client using stored token credentials.
func HTTPClient(w http.ResponseWriter, r *http.Request) (*http.Client, error) {
	session, _ := store.Get(r, sessionName)
	parsedExpiry, err := time.Parse(time.RFC3339, session.Values["expiry"].(string))
	if err != nil {
		return nil, err
	}
	token := &oauth2.Token{
		AccessToken:  session.Values["accessToken"].(string),
		RefreshToken: session.Values["refreshToken"].(string),
		TokenType:    session.Values["tokenType"].(string),
		Expiry:       parsedExpiry,
	}
	return config.Client(context.Background(), token), nil
}

// HandleLogin initiates the Oauth flow.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("could not parse request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	autoAuth := r.FormValue("auto") == "true"
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/"
	}

	state := randomString(oauthStateLength, redirect)
	// Ignore store.Get() errors because an error indicates the
	// old session could not be deciphered. It returns a new session
	// regardless.
	session, _ := store.Get(r, sessionName)
	session.Values["oauthState"] = state
	err := session.Save(r, w)
	if err != nil {
		log.Errorf("Error saving key: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var authURL string
	if autoAuth {
		authURL = config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	} else {
		authURL = config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	}
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles the response from the Oauth callback URL.
// It verifies response state and populates session with callback values.
// Redirects to URL stored in the callback state on completion.
func HandleCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("could not parse query: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	authCode := r.FormValue("code")
	callbackState := r.FormValue("state")

	session, err := store.Get(r, sessionName)
	if err != nil {
		log.Errorf("error retrieving session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if val, ok := session.Values["oauthState"]; !ok || val == nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sessionState := session.Values["oauthState"].(string)
	if callbackState != sessionState {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Errorf("error exchanging token: %v", token)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["accessToken"] = token.AccessToken
	session.Values["refreshToken"] = token.RefreshToken
	session.Values["tokenType"] = token.TokenType
	session.Values["expiry"] = token.Expiry.Format(time.RFC3339)
	err = session.Save(r, w)
	if err != nil {
		log.Errorf("error saving session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, returnURLFromState(sessionState), http.StatusTemporaryRedirect)
}

// HandleSignOut signs out the user by removing token information from the
// session
func HandleSignOut(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("could not parse request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/"
	}
	session, _ := store.Get(r, sessionName)
	session.Values["accessToken"] = nil
	session.Values["refreshToken"] = nil
	session.Values["tokenType"] = nil
	session.Values["expiry"] = nil
	err := session.Save(r, w)
	if err != nil {
		log.Errorf("error saving session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
}
