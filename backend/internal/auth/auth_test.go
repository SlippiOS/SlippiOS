package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	savedKey, savedIdentity, savedGraphQL, savedClient := APIKey, IdentityURL, GraphQLURL, Client
	APIKey = "test-key"
	IdentityURL = server.URL + "/identity"
	GraphQLURL = server.URL + "/graphql"
	Client = server.Client()
	t.Cleanup(func() {
		server.Close()
		APIKey, IdentityURL, GraphQLURL, Client = savedKey, savedIdentity, savedGraphQL, savedClient
	})
	return server
}

func TestLoginSuccess(t *testing.T) {
	testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/identity":
			if r.URL.Query().Get("key") != "test-key" {
				t.Errorf("api key = %q", r.URL.Query().Get("key"))
			}
			var body struct {
				Email    string `json:"email"`
				Password string `json:"password"`
				Secure   bool   `json:"returnSecureToken"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode identity: %v", err)
			}
			if body.Email != "player@example.com" || body.Password != "secret" || !body.Secure {
				t.Errorf("identity body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"idToken":"id-token","localId":"fb-uid"}`))
		case "/graphql":
			if r.Header.Get("Authorization") != "Bearer id-token" {
				t.Errorf("authorization = %q", r.Header.Get("Authorization"))
			}
			var request struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode graphql: %v", err)
			}
			if request.Variables["fbUid"] != "fb-uid" {
				t.Errorf("variables = %#v", request.Variables)
			}
			if !strings.Contains(request.Query, "getUserKeyQuery") {
				t.Errorf("query = %q", request.Query)
			}
			_, _ = w.Write([]byte(`{"data":{"getUser":{
				"fbUid":"fb-uid",
				"displayName":"Player One",
				"connectCode":{"code":"PLR#1"},
				"private":{"playKey":"play-key"}
			}}}`))
		default:
			http.NotFound(w, r)
		}
	}))

	user, err := Login("player@example.com", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if user.UID != "fb-uid" || user.DisplayName != "Player One" ||
		user.ConnectCode != "PLR#1" || user.PlayKey != "play-key" {
		t.Fatalf("user = %#v", user)
	}
}

func TestLoginFriendlyErrors(t *testing.T) {
	cases := map[string]string{
		"INVALID_LOGIN_CREDENTIALS":   "Incorrect email or password.",
		"EMAIL_NOT_FOUND":             "Incorrect email or password.",
		"INVALID_PASSWORD":            "Incorrect email or password.",
		"USER_DISABLED":               "This account has been disabled.",
		"TOO_MANY_ATTEMPTS_TRY_LATER": "Too many attempts. Try again later.",
		"SOMETHING_ELSE":              "Slippi sign-in failed (SOMETHING_ELSE).",
	}
	for code, expected := range cases {
		code, expected := code, expected
		t.Run(code, func(t *testing.T) {
			testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"message":"` + code + `"}}`))
			}))
			_, err := Login("player@example.com", "secret")
			if err == nil || err.Error() != expected {
				t.Fatalf("err = %v, want %q", err, expected)
			}
			if _, ok := err.(*AuthError); !ok {
				t.Fatalf("err type = %T", err)
			}
		})
	}
}

func TestLoginRequiresActivation(t *testing.T) {
	testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/identity" {
			_, _ = w.Write([]byte(`{"idToken":"id-token","localId":"fb-uid"}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"getUser":{
			"displayName":"Player",
			"connectCode":{"code":"PLR#1"},
			"private":{"playKey":""}
		}}}`))
	}))

	_, err := Login("player@example.com", "secret")
	if err == nil || !strings.Contains(err.Error(), "not activated for online play") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoginRequiresCredentials(t *testing.T) {
	if _, err := Login("", "secret"); err == nil {
		t.Fatal("missing email accepted")
	}
	if _, err := Login("player@example.com", ""); err == nil {
		t.Fatal("missing password accepted")
	}
}
