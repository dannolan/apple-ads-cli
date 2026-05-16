package appleads

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dannolan/apple-ads-cli/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenURL   = "https://appleid.apple.com/auth/oauth2/token"
	APIBaseURL = "https://api.searchads.apple.com/api/v5"
)

type Client struct {
	HTTP        *http.Client
	Creds       config.Credentials
	BaseURL     string
	TokenURL    string
	token       string
	tokenExpiry time.Time
}

type RequestOptions struct {
	Method         string
	Path           string
	Query          url.Values
	Body           any
	SkipOrgContext bool
}

type Page struct {
	Data       []map[string]any `json:"data"`
	Pagination struct {
		TotalResults int `json:"totalResults"`
	} `json:"pagination"`
}

func NewClient(creds config.Credentials) *Client {
	return &Client{
		HTTP:     &http.Client{Timeout: 60 * time.Second},
		Creds:    creds,
		BaseURL:  APIBaseURL,
		TokenURL: TokenURL,
	}
}

func (c *Client) Request(opts RequestOptions) (map[string]any, error) {
	if opts.Method == "" {
		opts.Method = http.MethodGet
	}
	var body io.Reader
	if opts.Body != nil {
		data, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	endpoint := strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(opts.Path, "/")
	if len(opts.Query) > 0 {
		endpoint += "?" + opts.Query.Encode()
	}
	req, err := http.NewRequest(opts.Method, endpoint, body)
	if err != nil {
		return nil, err
	}
	token, err := c.accessToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if !opts.SkipOrgContext {
		req.Header.Set("X-AP-Context", fmt.Sprintf("orgId=%d", c.Creds.OrgID))
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNoContent {
		return map[string]any{}, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("apple ads api %s %s returned %d: %s", opts.Method, opts.Path, resp.StatusCode, string(data))
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Paginate(path string, q url.Values) ([]map[string]any, error) {
	if q == nil {
		q = url.Values{}
	}
	out := []map[string]any{}
	offset := 0
	for {
		pageQ := url.Values{}
		for k, vals := range q {
			pageQ[k] = append([]string{}, vals...)
		}
		pageQ.Set("limit", "1000")
		pageQ.Set("offset", strconv.Itoa(offset))
		resp, err := c.Request(RequestOptions{Method: http.MethodGet, Path: path, Query: pageQ})
		if err != nil {
			return nil, err
		}
		data, _ := resp["data"].([]any)
		for _, item := range data {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		total := paginationTotal(resp)
		offset += len(data)
		if len(data) == 0 || offset >= total {
			break
		}
	}
	return out, nil
}

func paginationTotal(resp map[string]any) int {
	p, _ := resp["pagination"].(map[string]any)
	switch v := p["totalResults"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func (c *Client) accessToken() (string, error) {
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}
	secret, err := c.clientSecret()
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.Creds.ClientID)
	form.Set("client_secret", secret)
	form.Set("scope", "searchadsorg")
	resp, err := c.HTTP.PostForm(c.TokenURL, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(data))
	}
	var token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(data, &token); err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", errors.New("token response did not include access_token")
	}
	if token.ExpiresIn == 0 {
		token.ExpiresIn = 3600
	}
	c.token = token.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn-300) * time.Second)
	return c.token, nil
}

func (c *Client) clientSecret() (string, error) {
	key, err := loadECPrivateKey(c.Creds.PrivateKeyPath)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": c.Creds.ClientID,
		"aud": "https://appleid.apple.com",
		"iat": now.Unix(),
		"exp": now.Add(180 * 24 * time.Hour).Unix(),
		"iss": c.Creds.TeamID,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = c.Creds.KeyID
	return tok.SignedString(key)
}

func loadECPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("private key is not PEM encoded")
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an EC private key")
	}
	return key, nil
}
