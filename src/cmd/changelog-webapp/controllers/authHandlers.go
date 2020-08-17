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
	sessionAge       = 84600

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

// ErrorSessionRetrieval indicates that a request has no session, or the
// session was malformed.
var ErrorSessionRetrieval = errors.New("No session found")

func init() {
	var err error
	client, err := secretmanager.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to setup client: %v", err)
	}
	config.ClientSecret, err = getSecret(client, clientSecretName)
	if err != nil {
		log.Fatalf("Failed to retrieve secret: %s\n%v", clientSecretName, err)
	}

	sessionSecret, err := getSecret(client, sessionSecretName)
	if err != nil {
		log.Fatalf("Failed to retrieve secret :%s\n%v", sessionSecretName, err)
	}
	store = sessions.NewCookieStore([]byte(sessionSecret))
	store.MaxAge(sessionAge)
}

// Retrieve secrets stored in Gcloud Secret Manager
func getSecret(client *secretmanager.Client, secretName string) (string, error) {
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
	}
	result, err := client.AccessSecretVersion(context.Background(), accessRequest)
	if err != nil {
		return "", fmt.Errorf("Failed to access secret at %s: %v", accessRequest.Name, err)
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

// HTTPClient creates an authorized HTTP Client using stored token credentials.
// Returns error if no session or a malformed session is detected.
// Otherwise returns an HTTP Client with the stored Oauth access token.
// If the access token is expired, automatically refresh the token
func HTTPClient(w http.ResponseWriter, r *http.Request, returnURL string) (*http.Client, error) {
	var parsedExpiry time.Time
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return nil, ErrorSessionRetrieval
	}
	for _, key := range []string{"accessToken", "refreshToken", "tokenType", "expiry"} {
		if val, ok := session.Values[key]; !ok || val == nil {
			return nil, ErrorSessionRetrieval
		}
	}
	if parsedExpiry, err = time.Parse(time.RFC3339, session.Values["expiry"].(string)); err != nil {
		return nil, ErrorSessionRetrieval
	}
	if parsedExpiry.Before(time.Now()) {
		log.Debug("HTTPClient: Token expired, calling Oauth flow")
		HandleLogin(w, r, returnURL)
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
func HandleLogin(w http.ResponseWriter, r *http.Request, returnURL string) {
	state := randomString(oauthStateLength, returnURL)
	// Ignore store.Get() errors in HandleLogin because an error indicates the
	// old session could not be deciphered. It returns a new session
	// regardless.
	session, _ := store.Get(r, sessionName)
	session.Values["oauthState"] = state
	err := session.Save(r, w)
	if err != nil {
		log.Errorf("HandleLogin: Error saving key: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles the response from the Oauth callback URL.
// It verifies response state and populates session with callback values.
// Redirects to URL stored in the callback state on completion.
func HandleCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("Could not parse query: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	authCode := r.FormValue("code")
	callbackState := r.FormValue("state")

	session, err := store.Get(r, sessionName)
	if err != nil {
		log.Errorf("HandleCallback: Error retrieving session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if val, ok := session.Values["oauthState"]; !ok || val == nil {
		http.Redirect(w, r, "/", http.StatusPermanentRedirect)
		return
	}
	sessionState := session.Values["oauthState"].(string)
	if callbackState != sessionState {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Errorf("HandleCallback: Error exchanging token: %v", token)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["accessToken"] = token.AccessToken
	session.Values["refreshToken"] = token.RefreshToken
	session.Values["tokenType"] = token.TokenType
	session.Values["expiry"] = token.Expiry.Format(time.RFC3339)
	err = session.Save(r, w)
	if err != nil {
		log.Errorf("HandleCallback: Error saving session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, returnURLFromState(sessionState), http.StatusPermanentRedirect)
}
