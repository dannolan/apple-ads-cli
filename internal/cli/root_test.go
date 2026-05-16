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

func TestManifestIncludesSmoke(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"manifest", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ads smoke") {
		t.Fatalf("manifest did not include smoke command: %s", stdout.String())
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

func TestKeywordBidDryRunUsesConfiguredCurrency(t *testing.T) {
	configDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"config", "app", "add", "--app-id", "123456", "--name", "My App", "--currency", "AUD", "--config-dir", configDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("config app add failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"keywords", "update-bid", "1", "2", "3", "--bid", "3", "--config-dir", configDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("keywords update-bid dry-run failed: %s", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &payload); err != nil {
		t.Fatalf("stdout was not JSON: %v\n%s", err, stdout.String())
	}
	body := payload["body"].(map[string]any)
	bid := body["bidAmount"].(map[string]any)
	if bid["currency"] != "AUD" {
		t.Fatalf("expected configured AUD currency, got %#v", body)
	}
}

func TestTypedCampaignCommandsBuildSafePayloads(t *testing.T) {
	configDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"config", "app", "add", "--app-id", "123456", "--name", "My App", "--currency", "AUD", "--config-dir", configDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("config app add failed: %s", stderr.String())
	}

	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, payload map[string]any)
	}{
		{"set budget", []string{"campaigns", "set-budget", "1", "--amount", "20", "--config-dir", configDir, "--json"}, func(t *testing.T, payload map[string]any) {
			campaign := payload["body"].(map[string]any)["campaign"].(map[string]any)
			budget := campaign["dailyBudgetAmount"].(map[string]any)
			if budget["amount"] != "20.00" || budget["currency"] != "AUD" {
				t.Fatalf("unexpected budget payload: %#v", payload)
			}
		}},
		{"set countries", []string{"campaigns", "set-countries", "1", "--countries", "AU,US", "--config-dir", configDir, "--json"}, func(t *testing.T, payload map[string]any) {
			if payload["body"].(map[string]any)["clearGeoTargetingOnCountryOrRegionChange"] != false {
				t.Fatalf("expected geo clear flag false: %#v", payload)
			}
		}},
		{"rename", []string{"campaigns", "rename", "1", "--name", "ARCHIVED - Discovery", "--config-dir", configDir, "--json"}, func(t *testing.T, payload map[string]any) {
			campaign := payload["body"].(map[string]any)["campaign"].(map[string]any)
			if campaign["name"] != "ARCHIVED - Discovery" {
				t.Fatalf("unexpected rename payload: %#v", payload)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout.Reset()
			stderr.Reset()
			code = Execute(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("%s failed: %s", tt.name, stderr.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &payload); err != nil {
				t.Fatalf("stdout was not JSON: %v\n%s", err, stdout.String())
			}
			tt.check(t, payload)
		})
	}
}

func TestAdGroupSetBidDryRunUsesConfiguredCurrency(t *testing.T) {
	configDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"config", "app", "add", "--app-id", "123456", "--name", "My App", "--currency", "AUD", "--config-dir", configDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("config app add failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"adgroups", "set-bid", "1", "2", "--bid", "2.5", "--config-dir", configDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("adgroups set-bid dry-run failed: %s", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &payload); err != nil {
		t.Fatalf("stdout was not JSON: %v\n%s", err, stdout.String())
	}
	bid := payload["body"].(map[string]any)["defaultBidAmount"].(map[string]any)
	if bid["amount"] != "2.50" || bid["currency"] != "AUD" {
		t.Fatalf("expected AUD defaultBidAmount payload: %#v", payload)
	}
}

func TestTableOutputForCampaignData(t *testing.T) {
	var out bytes.Buffer
	ctx := &appContext{Table: true, Out: &out}
	err := ctx.Print(map[string]any{
		"data": []any{
			map[string]any{
				"name":               "Category",
				"id":                 123,
				"status":             "PAUSED",
				"countriesOrRegions": []any{"AU", "US"},
				"dailyBudgetAmount":  map[string]any{"amount": "20", "currency": "AUD"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"name", "Category", "AU,US", "20 AUD"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in table output: %s", want, got)
		}
	}
}

func TestBuildReportBodyOmitsGroupBy(t *testing.T) {
	body := buildReportBody("2026-05-01", "2026-05-16", "/reports/campaigns")
	if _, ok := body["groupBy"]; ok {
		t.Fatalf("standard Apple Ads report bodies must not include groupBy: %#v", body)
	}
	if body["returnRecordsWithNoMetrics"] != true {
		t.Fatalf("expected returnRecordsWithNoMetrics=true: %#v", body)
	}
	if body["timeZone"] != "UTC" {
		t.Fatalf("expected UTC timezone: %#v", body)
	}
}

func TestBuildReportBodyUsesOrgTimezoneForSearchTerms(t *testing.T) {
	body := buildReportBody("2026-05-01", "2026-05-16", "/reports/campaigns/123/searchterms")
	if body["timeZone"] != "ORTZ" {
		t.Fatalf("expected ORTZ timezone for search term reports: %#v", body)
	}
	if body["returnRecordsWithNoMetrics"] != false {
		t.Fatalf("expected returnRecordsWithNoMetrics=false for search term reports: %#v", body)
	}
}

func TestCampaignUpdatePayloadWrapsSimpleBody(t *testing.T) {
	payload := campaignUpdatePayload(map[string]any{
		"name": "ARCHIVED - Discovery",
		"dailyBudgetAmount": map[string]any{
			"amount":   "20",
			"currency": "AUD",
		},
	}).(map[string]any)
	campaign := payload["campaign"].(map[string]any)
	if campaign["name"] != "ARCHIVED - Discovery" {
		t.Fatalf("expected campaign update envelope: %#v", payload)
	}
	if _, ok := payload["dailyBudgetAmount"]; ok {
		t.Fatalf("did not expect top-level campaign field: %#v", payload)
	}
}

func TestCampaignUpdatePayloadPreservesEnvelopeOptions(t *testing.T) {
	payload := campaignUpdatePayload(map[string]any{
		"clearGeoTargetingOnCountryOrRegionChange": true,
		"countriesOrRegions":                       []string{"US", "AU"},
	}).(map[string]any)
	if payload["clearGeoTargetingOnCountryOrRegionChange"] != true {
		t.Fatalf("expected top-level geo clearing option: %#v", payload)
	}
	campaign := payload["campaign"].(map[string]any)
	if _, ok := campaign["countriesOrRegions"]; !ok {
		t.Fatalf("expected countries in campaign envelope: %#v", payload)
	}
}

func TestCampaignUpdatePayloadDefaultsGeoClearFlagForCountryUpdates(t *testing.T) {
	payload := campaignUpdatePayload(map[string]any{
		"countriesOrRegions": []string{"US", "AU"},
	}).(map[string]any)
	if payload["clearGeoTargetingOnCountryOrRegionChange"] != false {
		t.Fatalf("expected default geo clearing flag for country updates: %#v", payload)
	}
}
