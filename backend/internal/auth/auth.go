// Package auth signs in with the public Slippi web endpoints.
package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultAPIKey    = "AIzaSyAuQqc_wgqcUu3FqrICEPZ9Av_hPxMR_i4"
	identityEndpoint = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"
	graphqlEndpoint  = "https://internal.slippi.gg/graphql"
)

var (
	APIKey      = defaultAPIKey
	IdentityURL = identityEndpoint
	GraphQLURL  = graphqlEndpoint
	Client      = &http.Client{Timeout: 25 * time.Second}
)

const userQuery = `
query getUserKeyQuery($fbUid: String) {
  getUser(fbUid: $fbUid) {
    fbUid
    displayName
    connectCode { code }
    private { playKey }
  }
}
`

type User struct {
	UID         string `json:"uid"`
	DisplayName string `json:"displayName"`
	ConnectCode string `json:"connectCode"`
	PlayKey     string `json:"playKey"`
}

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string { return e.Message }

func friendly(code string) string {
	switch code {
	case "INVALID_LOGIN_CREDENTIALS", "EMAIL_NOT_FOUND", "INVALID_PASSWORD":
		return "Incorrect email or password."
	case "USER_DISABLED":
		return "This account has been disabled."
	case "TOO_MANY_ATTEMPTS_TRY_LATER":
		return "Too many attempts. Try again later."
	default:
		if code == "" {
			return "Slippi sign-in failed."
		}
		return fmt.Sprintf("Slippi sign-in failed (%s).", code)
	}
}

func post(url string, payload any, token string) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, &AuthError{Message: "Slippi sign-in failed."}
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, &AuthError{Message: "Slippi sign-in failed."}
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := Client.Do(request)
	if err != nil {
		return nil, &AuthError{Message: "Cannot reach Slippi. Check the network connection."}
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	if response.StatusCode >= 400 {
		var decoded struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &decoded)
		return nil, &AuthError{Message: friendly(decoded.Error.Message)}
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, &AuthError{Message: "Slippi sign-in failed (invalid response)."}
	}
	return decoded, nil
}

func Login(email, password string) (User, error) {
	if email == "" || password == "" {
		return User{}, &AuthError{Message: "Email and password are required."}
	}
	session, err := post(
		fmt.Sprintf("%s?key=%s", IdentityURL, APIKey),
		map[string]any{
			"email":             email,
			"password":          password,
			"returnSecureToken": true,
		},
		"",
	)
	if err != nil {
		return User{}, err
	}
	token, _ := session["idToken"].(string)
	uid, _ := session["localId"].(string)
	if token == "" || uid == "" {
		return User{}, &AuthError{Message: "Slippi sign-in failed."}
	}

	result, err := post(GraphQLURL, map[string]any{
		"query":     userQuery,
		"variables": map[string]any{"fbUid": uid},
	}, token)
	if err != nil {
		return User{}, err
	}
	data, _ := result["data"].(map[string]any)
	user, _ := data["getUser"].(map[string]any)
	if user == nil {
		return User{}, &AuthError{Message: "No Slippi account data found for this login."}
	}
	private, _ := user["private"].(map[string]any)
	connect, _ := user["connectCode"].(map[string]any)
	playKey, _ := private["playKey"].(string)
	if playKey == "" {
		return User{}, &AuthError{
			Message: "This account is not activated for online play. " +
				"Activate it once at slippi.gg/online/enable.",
		}
	}
	displayName, _ := user["displayName"].(string)
	connectCode, _ := connect["code"].(string)
	return User{
		UID:         uid,
		DisplayName: displayName,
		ConnectCode: connectCode,
		PlayKey:     playKey,
	}, nil
}
