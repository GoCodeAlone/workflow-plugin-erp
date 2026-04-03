package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuth2Token holds an access token with expiry info.
type OAuth2Token struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

// SAPAuthConfig holds authentication configuration for SAP.
type SAPAuthConfig struct {
	AuthType     string // basic, oauth2, apikey
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
	TokenURL     string
	APIKey       string
}

// SAPAuth manages SAP authentication including CSRF tokens and OAuth2.
type SAPAuth struct {
	config     SAPAuthConfig
	httpClient *http.Client
	baseURL    string

	mu         sync.Mutex
	csrfToken  string
	oauthToken *OAuth2Token
}

// NewSAPAuth creates a new SAP auth manager.
func NewSAPAuth(baseURL string, config SAPAuthConfig, httpClient *http.Client) *SAPAuth {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &SAPAuth{
		config:     config,
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

// AuthHeader returns an AuthHeaderFunc that applies auth headers to requests.
func (a *SAPAuth) AuthHeader() AuthHeaderFunc {
	return func(req *http.Request) error {
		switch a.config.AuthType {
		case "basic":
			creds := base64.StdEncoding.EncodeToString([]byte(a.config.Username + ":" + a.config.Password))
			req.Header.Set("Authorization", "Basic "+creds)
		case "oauth2":
			token, err := a.getOAuthToken(req.Context())
			if err != nil {
				return fmt.Errorf("oauth2 token: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
		case "apikey":
			req.Header.Set("APIKey", a.config.APIKey)
		default:
			return fmt.Errorf("unsupported auth type: %s", a.config.AuthType)
		}
		return nil
	}
}

func (a *SAPAuth) getOAuthToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.oauthToken != nil && time.Now().Before(a.oauthToken.ExpiresAt) {
		return a.oauthToken.AccessToken, nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {a.config.ClientID},
		"client_secret": {a.config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.TokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}

	a.oauthToken = &OAuth2Token{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn-30) * time.Second),
	}
	return a.oauthToken.AccessToken, nil
}

// FetchCSRFToken retrieves a CSRF token from the SAP server.
func (a *SAPAuth) FetchCSRFToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-CSRF-Token", "Fetch")

	if err := a.AuthHeader()(req); err != nil {
		return "", err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("csrf fetch: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	token := resp.Header.Get("X-CSRF-Token")
	if token == "" {
		return "", fmt.Errorf("no X-CSRF-Token in response (status %d)", resp.StatusCode)
	}
	a.csrfToken = token
	return token, nil
}

// CSRFToken returns the cached CSRF token.
func (a *SAPAuth) CSRFToken() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.csrfToken
}
