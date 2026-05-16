package cli

import (
	"fmt"
	"net/http"

	"github.com/dannolan/apple-ads-cli/internal/appleads"
	"github.com/spf13/cobra"
)

func newSmokeCommand(ctx *appContext) *cobra.Command {
	var includeCampaigns bool
	cmd := &cobra.Command{
		Use:   "smoke",
		Short: "Run read-only live API smoke checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			result := map[string]any{
				"ok":      true,
				"version": Version,
				"checks":  []map[string]any{},
			}
			checks := []map[string]any{}
			addCheck := func(name string, ok bool, details any, err error) {
				check := map[string]any{"name": name, "ok": ok}
				if details != nil {
					check["details"] = details
				}
				if err != nil {
					check["error"] = err.Error()
					result["ok"] = false
				}
				checks = append(checks, check)
			}

			me, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: "/me", SkipOrgContext: true})
			addCheck("auth.me", err == nil, compactKeys(me), err)

			countries, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: "/countries-or-regions"})
			addCheck("metadata.countries", err == nil, compactKeys(countries), err)

			app, slug, appErr := ctx.ActiveApp()
			if appErr != nil {
				addCheck("config.active_app", false, nil, appErr)
			} else {
				addCheck("config.active_app", true, map[string]any{"slug": slug, "app_id": app.ID, "name": app.Name}, nil)
				eligibility, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: fmt.Sprintf("/apps/%d/eligibilities/find", app.ID), Body: map[string]any{}})
				addCheck("app.eligibility", err == nil, compactKeys(eligibility), err)
			}

			if includeCampaigns {
				campaigns, err := client.Paginate("/campaigns", nil)
				addCheck("campaigns.list", err == nil, map[string]any{"count": len(campaigns)}, err)
			}

			result["checks"] = checks
			return ctx.Print(result)
		},
	}
	cmd.Flags().BoolVar(&includeCampaigns, "include-campaigns", true, "include campaign list check")
	return cmd
}

func compactKeys(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := map[string]any{}
	if data, ok := value["data"]; ok {
		switch v := data.(type) {
		case []any:
			out["data_count"] = len(v)
		case map[string]any:
			out["data_keys"] = keys(v)
		default:
			out["has_data"] = true
		}
	}
	if pagination, ok := value["pagination"]; ok {
		out["pagination"] = pagination
	}
	return out
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
