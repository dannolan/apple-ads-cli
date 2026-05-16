package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteJSONError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"campaigns", "create", "--name", "Test", "--json", "--config-dir", t.TempDir()}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stderr.Bytes()), &payload); err != nil {
		t.Fatalf("stderr was not JSON: %v\n%s", err, stderr.String())
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload)
	}
	if !strings.Contains(payload["error"].(string), "no active app configured") {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
}

func TestManifestIncludesMutationMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"manifest", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit %d: %s", code, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &payload); err != nil {
		t.Fatalf("stdout was not JSON: %v\n%s", err, stdout.String())
	}
	commands := payload["commands"].([]any)
	var found bool
	for _, raw := range commands {
		cmd := raw.(map[string]any)
		if cmd["path"] == "ads campaigns create" {
			found = true
			if cmd["mutation"] != true {
				t.Fatalf("campaigns create should be marked as mutation: %#v", cmd)
			}
		}
	}
	if !found {
		t.Fatal("manifest did not include ads campaigns create")
	}
}

func TestCampaignCreateDryRunUsesConfiguredApp(t *testing.T) {
	configDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"config", "app", "add", "--app-id", "123456", "--name", "My App", "--config-dir", configDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("config app add failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"campaigns", "create", "--name", "My App Brand", "--config-dir", configDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("campaigns create dry-run failed: %s", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &payload); err != nil {
		t.Fatalf("stdout was not JSON: %v\n%s", err, stdout.String())
	}
	if payload["dry_run"] != true {
		t.Fatalf("expected dry-run payload: %#v", payload)
	}
	body := payload["body"].(map[string]any)
	if body["adamId"].(float64) != 123456 {
		t.Fatalf("expected adamId from app config, got %#v", body)
	}
	if _, err := os.Stat(filepath.Join(configDir, "apps.json")); err != nil {
		t.Fatal("expected apps config to be written")
	}
}
