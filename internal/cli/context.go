package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/dannolan/apple-ads-cli/internal/appleads"
	"github.com/dannolan/apple-ads-cli/internal/config"
)

type appContext struct {
	ConfigDir string
	AppSlug   string
	JSON      bool
	Pretty    bool
	store     *config.Store
}

func (a *appContext) Store() (*config.Store, error) {
	if a.store != nil {
		return a.store, nil
	}
	store, err := config.NewStore(a.ConfigDir)
	if err != nil {
		return nil, err
	}
	a.store = store
	return store, nil
}

func (a *appContext) Client() (*appleads.Client, error) {
	store, err := a.Store()
	if err != nil {
		return nil, err
	}
	creds, err := store.LoadCredentials()
	if err != nil {
		return nil, fmt.Errorf("load credentials: %w", err)
	}
	return appleads.NewClient(creds), nil
}

func (a *appContext) ActiveApp() (config.App, string, error) {
	store, err := a.Store()
	if err != nil {
		return config.App{}, "", err
	}
	return store.ActiveApp(a.AppSlug)
}

func (a *appContext) Print(value any) error {
	indent := ""
	if a.Pretty || !a.JSON {
		indent = "  "
	}
	var data []byte
	var err error
	if indent == "" {
		data, err = json.Marshal(value)
	} else {
		data, err = json.MarshalIndent(value, "", indent)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func parseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseQuery(items []string) (url.Values, error) {
	q := url.Values{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("query must be key=value, got %q", item)
		}
		q.Add(key, value)
	}
	return q, nil
}

func readBody(pathOrJSON string) (any, error) {
	if pathOrJSON == "" {
		return nil, nil
	}
	data := []byte(pathOrJSON)
	if strings.HasPrefix(pathOrJSON, "@") {
		var err error
		data, err = os.ReadFile(strings.TrimPrefix(pathOrJSON, "@"))
		if err != nil {
			return nil, err
		}
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func dryRunPayload(action, path string, body any) map[string]any {
	return map[string]any{
		"dry_run": true,
		"action":  action,
		"path":    path,
		"body":    body,
		"hint":    "Re-run with --apply to execute this mutation.",
	}
}
