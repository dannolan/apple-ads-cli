package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dannolan/apple-ads-cli/internal/appleads"
	"github.com/dannolan/apple-ads-cli/internal/config"
)

type appContext struct {
	ConfigDir string
	AppSlug   string
	JSON      bool
	Pretty    bool
	Table     bool
	Out       io.Writer
	Err       io.Writer
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

func (a *appContext) DefaultCurrency() string {
	app, _, err := a.ActiveApp()
	if err == nil && strings.TrimSpace(app.DefaultCurrency) != "" {
		return strings.ToUpper(strings.TrimSpace(app.DefaultCurrency))
	}
	return "USD"
}

func (a *appContext) Print(value any) error {
	out := a.Out
	if out == nil {
		out = os.Stdout
	}
	if a.Table && !a.JSON {
		if ok, err := a.printTable(out, value); ok || err != nil {
			return err
		}
	}
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
	_, err = fmt.Fprintln(out, string(data))
	return err
}

func (a *appContext) printTable(out io.Writer, value any) (bool, error) {
	rows, ok := tableRows(value)
	if !ok {
		return false, nil
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no rows")
		if err != nil {
			return true, err
		}
		return true, w.Flush()
	}
	cols := tableColumns(rows)
	_, _ = fmt.Fprintln(w, strings.Join(cols, "\t"))
	for _, row := range rows {
		values := make([]string, len(cols))
		for i, col := range cols {
			values[i] = tableCell(row[col])
		}
		_, _ = fmt.Fprintln(w, strings.Join(values, "\t"))
	}
	return true, w.Flush()
}

func tableRows(value any) ([]map[string]any, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	if rows, ok := sliceMaps(m["data"]); ok {
		return rows, true
	}
	if rows, ok := sliceMaps(m["campaigns"]); ok {
		return rows, true
	}
	if data, ok := m["data"].(map[string]any); ok {
		if response, ok := data["reportingDataResponse"].(map[string]any); ok {
			if rows, ok := sliceMaps(response["row"]); ok {
				return reportRows(rows), true
			}
		}
	}
	return nil, false
}

func sliceMaps(value any) ([]map[string]any, bool) {
	if rows, ok := value.([]map[string]any); ok {
		return rows, true
	}
	raw, ok := value.([]any)
	if !ok {
		return nil, false
	}
	rows := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows, true
}

func reportRows(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		flat := map[string]any{}
		if metadata, ok := row["metadata"].(map[string]any); ok {
			for k, v := range metadata {
				flat[k] = v
			}
		}
		if total, ok := row["total"].(map[string]any); ok {
			for _, key := range []string{"impressions", "taps", "tapInstalls", "localSpend", "avgCPT", "tapInstallCPI", "ttr", "tapInstallRate"} {
				if v, ok := total[key]; ok {
					flat[key] = v
				}
			}
		}
		out = append(out, flat)
	}
	return out
}

func tableColumns(rows []map[string]any) []string {
	preferred := []string{
		"name", "id", "status", "displayStatus", "servingStatus", "countriesOrRegions", "dailyBudgetAmount",
		"campaignName", "campaignId", "adGroupName", "adGroupId", "keyword", "text", "matchType",
		"localSpend", "taps", "tapInstalls", "tapInstallCPI", "impressions", "ttr", "tapInstallRate",
	}
	seen := map[string]bool{}
	cols := []string{}
	for _, col := range preferred {
		for _, row := range rows {
			if _, ok := row[col]; ok {
				cols = append(cols, col)
				seen[col] = true
				break
			}
		}
	}
	for _, row := range rows {
		for col := range row {
			if !seen[col] && len(cols) < 12 {
				cols = append(cols, col)
				seen[col] = true
			}
		}
	}
	return cols
}

func tableCell(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, tableCell(item))
		}
		return strings.Join(parts, ",")
	case map[string]any:
		if amount, ok := v["amount"]; ok {
			if currency, ok := v["currency"]; ok {
				return fmt.Sprintf("%v %v", amount, currency)
			}
			return fmt.Sprintf("%v", amount)
		}
		data, err := json.Marshal(v)
		if err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (a *appContext) PrintError(err error) error {
	out := a.Err
	if out == nil {
		out = os.Stderr
	}
	if a.JSON {
		data, marshalErr := json.Marshal(map[string]any{
			"ok":    false,
			"error": err.Error(),
			"hint":  errorHint(err),
		})
		if marshalErr != nil {
			return marshalErr
		}
		_, writeErr := fmt.Fprintln(out, string(data))
		return writeErr
	}
	_, writeErr := fmt.Fprintln(out, err.Error())
	return writeErr
}

func errorHint(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no active app configured"):
		return "Run ads config app add, or pass --app <slug>."
	case strings.Contains(msg, "load credentials"):
		return "Run ads config init, or hydrate config from 1Password with ads config from-1password."
	case strings.Contains(msg, "unknown app"):
		return "Run ads config app list --json to inspect configured app slugs."
	case strings.Contains(msg, "required flag"):
		return "Run the same command with --help to see required flags."
	default:
		return ""
	}
}

func Execute(args []string, out, errOut io.Writer) int {
	cmd, ctx := NewRootCommandWithContext()
	ctx.Out = out
	ctx.Err = errOut
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		ctx.JSON = ctx.JSON || shouldEmitJSON(args)
		if printErr := ctx.PrintError(err); printErr != nil && errOut != nil {
			_, _ = fmt.Fprintln(errOut, printErr.Error())
		}
		return 1
	}
	return 0
}

func shouldEmitJSON(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
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

func opRef(vault, item, field string) string {
	return fmt.Sprintf("op://%s/%s/%s", vault, item, field)
}

func trimCommandOutput(data []byte) string {
	return strings.TrimSpace(string(data))
}
