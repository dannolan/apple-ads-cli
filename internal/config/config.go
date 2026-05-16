package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const DirName = ".apple-ads-cli"

type Credentials struct {
	OrgID          int64  `json:"org_id"`
	ClientID       string `json:"client_id"`
	TeamID         string `json:"team_id"`
	KeyID          string `json:"key_id"`
	PrivateKeyPath string `json:"private_key_path"`
}

type App struct {
	ID               int64    `json:"app_id"`
	Name             string   `json:"app_name"`
	DefaultCountries []string `json:"default_countries"`
	DefaultBid       float64  `json:"default_bid"`
	DefaultCPAGoal   float64  `json:"default_cpa_goal,omitempty"`
	DefaultCurrency  string   `json:"default_currency,omitempty"`
}

type AppsConfig struct {
	ActiveApp string         `json:"active_app,omitempty"`
	Apps      map[string]App `json:"apps"`
}

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, DirName)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) credentialsPath() string { return filepath.Join(s.dir, "credentials.json") }
func (s *Store) appsPath() string        { return filepath.Join(s.dir, "apps.json") }

func (s *Store) ensure() error {
	return os.MkdirAll(s.dir, 0o700)
}

func (s *Store) LoadCredentials() (Credentials, error) {
	var out Credentials
	if err := readJSON(s.credentialsPath(), &out); err != nil {
		return out, err
	}
	if out.OrgID == 0 || out.ClientID == "" || out.TeamID == "" || out.KeyID == "" || out.PrivateKeyPath == "" {
		return out, errors.New("credentials are incomplete")
	}
	return out, nil
}

func (s *Store) SaveCredentials(c Credentials) error {
	if err := s.ensure(); err != nil {
		return err
	}
	if err := writeJSON(s.credentialsPath(), c, 0o600); err != nil {
		return err
	}
	return nil
}

func (s *Store) LoadApps() (AppsConfig, error) {
	out := AppsConfig{Apps: map[string]App{}}
	if _, err := os.Stat(s.appsPath()); errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err := readJSON(s.appsPath(), &out); err != nil {
		return out, err
	}
	if out.Apps == nil {
		out.Apps = map[string]App{}
	}
	return out, nil
}

func (s *Store) SaveApps(c AppsConfig) error {
	if err := s.ensure(); err != nil {
		return err
	}
	if c.Apps == nil {
		c.Apps = map[string]App{}
	}
	return writeJSON(s.appsPath(), c, 0o600)
}

func (s *Store) ActiveApp(slug string) (App, string, error) {
	apps, err := s.LoadApps()
	if err != nil {
		return App{}, "", err
	}
	if slug == "" {
		slug = apps.ActiveApp
	}
	if slug == "" && len(apps.Apps) == 1 {
		for k, v := range apps.Apps {
			return v, k, nil
		}
	}
	if slug == "" {
		return App{}, "", errors.New("no active app configured; run ads config app add")
	}
	app, ok := apps.Apps[slug]
	if !ok {
		return App{}, "", fmt.Errorf("app %q is not configured", slug)
	}
	return app, slug, nil
}

func Slug(name string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return strings.Trim(re.ReplaceAllString(strings.ToLower(name), ""), "-")
}

func readJSON(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func writeJSON(path string, value any, perm os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}
