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

	"github.com/gorilla/sessions"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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
	ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
	Endpoint:     google.Endpoint,
	RedirectURL:  "https://cos-oss-interns-playground.uc.r.appspot.com/oauth2callback/",
	Scopes:       []string{"https://www.googleapis.com/auth/gerritcodereview"},
}

var store = sessions.NewCookieStore([]byte(randomString(sessionKeyLength)))

func randomString(stringSize int) string {
	randWithSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
	stateArr := make([]byte, stringSize)
	for i := range stateArr {
		stateArr[i] = oauthStateCharset[randWithSeed.Intn(len(oauthStateCharset))]
	}
	return string(stateArr)
}

// HTTPClient creates an authorized HTTP Client
func HTTPClient(r *http.Request) (*http.Client, error) {
	var parsedExpiry time.Time
	session, err := store.Get(r, sessionName)
	if err != nil {
		return nil, fmt.Errorf("HTTPClient: No session found with sessionName %s", sessionName)
	}
	for _, key := range []string{"accessToken", "refreshToken", "tokenType", "expiry"} {
		if val, ok := session.Values[key]; !ok || val == nil {
			return nil, fmt.Errorf("HTTPClient: Session missing key %s", key)
		}
	}
	if parsedExpiry, err = time.Parse(time.RFC3339, session.Values["expiry"].(string)); err != nil {
		return nil, fmt.Errorf("HTTPClient: Token expiry is in an incorrect format")
	}
	token := &oauth2.Token{
		AccessToken:  session.Values["accessToken"].(string),
		RefreshToken: session.Values["refreshToken"].(string),
		TokenType:    session.Values["tokenType"].(string),
		Expiry:       parsedExpiry,
	}
	return config.Client(context.Background(), token), nil
}

// HandleLogin handles login
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := randomString(oauthStateLength)
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

// HandleCallback handles callback
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
	if callbackState != session.Values["oauthState"].(string) {
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

	http.Redirect(w, r, "/", http.StatusPermanentRedirect)
}
