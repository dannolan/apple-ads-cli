package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dannolan/apple-ads-cli/internal/appleads"
	"github.com/spf13/cobra"
)

func newACLCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "acl", Short: "Inspect account access and Apple Ads eligibility"}
	cmd.AddCommand(simpleGet(ctx, "list", "/acls", true), simpleGet(ctx, "me", "/me", true))
	var term string
	searchApps := &cobra.Command{
		Use:   "search-apps",
		Short: "Search apps eligible for Apple Ads",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{"query": []string{term}}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: "/search/apps", Query: q})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	searchApps.Flags().StringVar(&term, "query", "", "app search query")
	_ = searchApps.MarkFlagRequired("query")
	cmd.AddCommand(searchApps)
	cmd.AddCommand(idGet(ctx, "eligibility <app-id>", "/apps/%s/eligibility", false))
	cmd.AddCommand(simpleGet(ctx, "countries", "/countries-or-regions", false))
	return cmd
}

func newCampaignsCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "campaigns", Short: "Inspect and mutate campaigns"}
	cmd.AddCommand(listEndpoint(ctx, "list", "/campaigns"))
	cmd.AddCommand(idGet(ctx, "get <campaign-id>", "/campaigns/%s", false))
	cmd.AddCommand(statusCommand(ctx, "pause <campaign-id>", "/campaigns/%s", "PAUSED"))
	cmd.AddCommand(statusCommand(ctx, "enable <campaign-id>", "/campaigns/%s", "ENABLED"))
	cmd.AddCommand(deleteCommand(ctx, "delete <campaign-id>", "/campaigns/%s"))
	cmd.AddCommand(campaignCreate(ctx), campaignUpdate(ctx), campaignAudit(ctx), campaignSetup(ctx))
	return cmd
}

func campaignCreate(ctx *appContext) *cobra.Command {
	var name, countries string
	var dailyBudget float64
	var apply bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a campaign; dry-run unless --apply is set",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, _, err := ctx.ActiveApp()
			if err != nil {
				return err
			}
			body := map[string]any{
				"name":                name,
				"adamId":              app.ID,
				"dailyBudgetAmount":   money(dailyBudget),
				"countriesOrRegions":  parseCSV(countries),
				"status":              "ENABLED",
				"adChannelType":       "SEARCH",
				"billingEvent":        "TAPS",
				"supplySources":       []string{"APPSTORE_SEARCH_RESULTS"},
				"budgetAmount":        nil,
				"campaignDisplayHint": "daily budgets are preferred because Apple retires lifetime budgets on 2026-06-16",
			}
			if !apply {
				return ctx.Print(dryRunPayload("POST", "/campaigns", body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: "/campaigns", Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "campaign name")
	cmd.Flags().StringVar(&countries, "countries", "US", "countries, comma separated")
	cmd.Flags().Float64Var(&dailyBudget, "daily-budget", 50, "daily budget amount")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func campaignUpdate(ctx *appContext) *cobra.Command {
	var bodyArg string
	var apply bool
	cmd := &cobra.Command{
		Use:   "update <campaign-id>",
		Short: "Update a campaign with a JSON merge body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readBody(bodyArg)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/campaigns/%s", args[0])
			if !apply {
				return ctx.Print(dryRunPayload("PUT", path, body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPut, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&bodyArg, "body", "", "JSON body or @file")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func campaignAudit(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Agent-readable audit of the four-campaign structure",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			campaigns, err := client.Paginate("/campaigns", nil)
			if err != nil {
				return err
			}
			expected := []string{"brand", "category", "competitor", "discovery"}
			found := map[string]bool{}
			for _, c := range campaigns {
				name := strings.ToLower(fmt.Sprint(c["name"]))
				for _, e := range expected {
					if strings.Contains(name, e) {
						found[e] = true
					}
				}
			}
			missing := []string{}
			for _, e := range expected {
				if !found[e] {
					missing = append(missing, e)
				}
			}
			return ctx.Print(map[string]any{"ok": len(missing) == 0, "missing_campaign_types": missing, "campaign_count": len(campaigns), "campaigns": campaigns})
		},
	}
}

func campaignSetup(ctx *appContext) *cobra.Command {
	var prefix, countries string
	var dailyBudget float64
	var apply bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Plan or create Brand/Category/Competitor/Discovery campaigns",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, _, err := ctx.ActiveApp()
			if err != nil {
				return err
			}
			types := []string{"Brand", "Category", "Competitor", "Discovery"}
			plans := []map[string]any{}
			for _, typ := range types {
				plans = append(plans, map[string]any{
					"name":               strings.TrimSpace(prefix + " " + typ),
					"adamId":             app.ID,
					"dailyBudgetAmount":  money(dailyBudget),
					"countriesOrRegions": parseCSV(countries),
					"status":             "ENABLED",
					"adChannelType":      "SEARCH",
					"billingEvent":       "TAPS",
					"supplySources":      []string{"APPSTORE_SEARCH_RESULTS"},
				})
			}
			if !apply {
				return ctx.Print(map[string]any{"dry_run": true, "campaigns": plans, "hint": "Re-run with --apply to create these campaigns."})
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			created := []map[string]any{}
			for _, body := range plans {
				resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: "/campaigns", Body: body})
				if err != nil {
					return err
				}
				created = append(created, resp)
			}
			return ctx.Print(map[string]any{"ok": true, "created": created})
		},
	}
	cmd.Flags().StringVar(&prefix, "prefix", "", "campaign name prefix, usually app name")
	cmd.Flags().StringVar(&countries, "countries", "US", "countries, comma separated")
	cmd.Flags().Float64Var(&dailyBudget, "daily-budget", 50, "daily budget per campaign")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	return cmd
}

func newAdGroupsCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "adgroups", Short: "Manage campaign ad groups"}
	cmd.AddCommand(campaignChildList(ctx, "list <campaign-id>", "/campaigns/%s/adgroups"))
	cmd.AddCommand(adGroupCreate(ctx))
	cmd.AddCommand(statusCommand(ctx, "pause <campaign-id> <adgroup-id>", "/campaigns/%s/adgroups/%s", "PAUSED"))
	cmd.AddCommand(statusCommand(ctx, "enable <campaign-id> <adgroup-id>", "/campaigns/%s/adgroups/%s", "ENABLED"))
	cmd.AddCommand(deleteCommand(ctx, "delete <campaign-id> <adgroup-id>", "/campaigns/%s/adgroups/%s"))
	return cmd
}

func adGroupCreate(ctx *appContext) *cobra.Command {
	var name string
	var bid float64
	var searchMatch bool
	var apply bool
	cmd := &cobra.Command{
		Use:   "create <campaign-id>",
		Short: "Create an ad group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"name": name, "status": "ENABLED", "defaultBidAmount": money(bid), "automatedKeywordsOptIn": searchMatch}
			path := fmt.Sprintf("/campaigns/%s/adgroups", args[0])
			if !apply {
				return ctx.Print(dryRunPayload("POST", path, body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "ad group name")
	cmd.Flags().Float64Var(&bid, "bid", 1.50, "default bid")
	cmd.Flags().BoolVar(&searchMatch, "search-match", false, "enable Search Match")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newKeywordsCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "keywords", Short: "Manage targeting and negative keywords"}
	cmd.AddCommand(campaignAdGroupList(ctx, "list <campaign-id> <adgroup-id>", "/campaigns/%s/adgroups/%s/targetingkeywords"))
	cmd.AddCommand(keywordAdd(ctx))
	cmd.AddCommand(keywordNegativeAdd(ctx))
	cmd.AddCommand(keywordFind(ctx))
	cmd.AddCommand(keywordUpdateBid(ctx))
	cmd.AddCommand(statusCommand(ctx, "pause <campaign-id> <adgroup-id> <keyword-id>", "/campaigns/%s/adgroups/%s/targetingkeywords/%s", "PAUSED"))
	cmd.AddCommand(statusCommand(ctx, "enable <campaign-id> <adgroup-id> <keyword-id>", "/campaigns/%s/adgroups/%s/targetingkeywords/%s", "ENABLED"))
	cmd.AddCommand(deleteCommand(ctx, "delete <campaign-id> <adgroup-id> <keyword-id>", "/campaigns/%s/adgroups/%s/targetingkeywords/%s"))
	cmd.AddCommand(campaignChildList(ctx, "list-negatives <campaign-id>", "/campaigns/%s/negativekeywords"))
	cmd.AddCommand(deleteCommand(ctx, "delete-negative <campaign-id> <negative-keyword-id>", "/campaigns/%s/negativekeywords/%s"))
	return cmd
}

func keywordAdd(ctx *appContext) *cobra.Command {
	var texts, matchType string
	var bid float64
	var apply bool
	cmd := &cobra.Command{
		Use:   "add <campaign-id> <adgroup-id>",
		Short: "Add targeting keywords in bulk",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			items := []map[string]any{}
			for _, text := range parseCSV(texts) {
				items = append(items, map[string]any{"text": text, "matchType": strings.ToUpper(matchType), "bidAmount": money(bid), "status": "ACTIVE"})
			}
			body := map[string]any{"keywords": items}
			path := fmt.Sprintf("/campaigns/%s/adgroups/%s/targetingkeywords/bulk", args[0], args[1])
			if !apply {
				return ctx.Print(dryRunPayload("POST", path, body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&texts, "text", "", "comma separated keyword texts")
	cmd.Flags().StringVar(&matchType, "match", "EXACT", "EXACT or BROAD")
	cmd.Flags().Float64Var(&bid, "bid", 1.50, "keyword bid")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func keywordNegativeAdd(ctx *appContext) *cobra.Command {
	var texts, matchType string
	var apply bool
	cmd := &cobra.Command{
		Use:   "add-negatives <campaign-id>",
		Short: "Add campaign negative keywords in bulk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			items := []map[string]any{}
			for _, text := range parseCSV(texts) {
				items = append(items, map[string]any{"text": text, "matchType": strings.ToUpper(matchType), "status": "ACTIVE"})
			}
			body := map[string]any{"negativeKeywords": items}
			path := fmt.Sprintf("/campaigns/%s/negativekeywords/bulk", args[0])
			if !apply {
				return ctx.Print(dryRunPayload("POST", path, body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&texts, "text", "", "comma separated negative keyword texts")
	cmd.Flags().StringVar(&matchType, "match", "EXACT", "EXACT or BROAD")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func keywordFind(ctx *appContext) *cobra.Command {
	var text string
	cmd := &cobra.Command{
		Use:   "find",
		Short: "Find targeting keywords across campaigns and ad groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			camps, err := client.Paginate("/campaigns", nil)
			if err != nil {
				return err
			}
			matches := []map[string]any{}
			for _, camp := range camps {
				cid := idString(camp["id"])
				adgroups, err := client.Paginate(fmt.Sprintf("/campaigns/%s/adgroups", cid), nil)
				if err != nil {
					return err
				}
				for _, ag := range adgroups {
					aid := idString(ag["id"])
					kws, err := client.Paginate(fmt.Sprintf("/campaigns/%s/adgroups/%s/targetingkeywords", cid, aid), nil)
					if err != nil {
						return err
					}
					for _, kw := range kws {
						if strings.Contains(strings.ToLower(fmt.Sprint(kw["text"])), strings.ToLower(text)) {
							kw["campaignId"] = cid
							kw["adGroupId"] = aid
							matches = append(matches, kw)
						}
					}
				}
			}
			return ctx.Print(map[string]any{"matches": matches, "count": len(matches)})
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "case-insensitive text filter")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func keywordUpdateBid(ctx *appContext) *cobra.Command {
	var bid float64
	var apply bool
	cmd := &cobra.Command{
		Use:   "update-bid <campaign-id> <adgroup-id> <keyword-id>",
		Short: "Update one keyword bid",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"bidAmount": money(bid)}
			path := fmt.Sprintf("/campaigns/%s/adgroups/%s/targetingkeywords/%s", args[0], args[1], args[2])
			if !apply {
				return ctx.Print(dryRunPayload("PUT", path, body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPut, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().Float64Var(&bid, "bid", 0, "new bid amount")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("bid")
	return cmd
}

func newReportsCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "reports", Short: "Generate Apple Ads reports"}
	cmd.AddCommand(reportCommand(ctx, "summary", "/reports/campaigns", "CAMPAIGN"))
	cmd.AddCommand(reportCommand(ctx, "keywords", "/reports/campaigns/%s/adgroups/%s/targetingkeywords", "KEYWORD"))
	cmd.AddCommand(reportCommand(ctx, "adgroups", "/reports/campaigns/%s/adgroups", "ADGROUP"))
	cmd.AddCommand(reportCommand(ctx, "search-terms", "/reports/campaigns/%s/adgroups/%s/searchterms", "SEARCHTERM"))
	cmd.AddCommand(reportCommand(ctx, "ads", "/reports/campaigns/%s/adgroups/%s/ads", "AD"))
	cmd.AddCommand(reportCommand(ctx, "impression-share", "/reports/campaigns/%s/searchterms/insights", "SEARCHTERM"))
	cmd.AddCommand(customReport(ctx), simpleGet(ctx, "custom-list", "/custom-reports", false), idGet(ctx, "custom-get <report-id>", "/custom-reports/%s", false))
	cmd.AddCommand(campaignAdGroupList(ctx, "bid-recommendations <campaign-id> <adgroup-id>", "/campaigns/%s/adgroups/%s/targetingkeywords/recommendations"))
	return cmd
}

func reportCommand(ctx *appContext, use, pathTemplate, selector string) *cobra.Command {
	var days int
	var start, end string
	cmd := &cobra.Command{
		Use:   use + optionalArgs(pathTemplate),
		Short: "Run " + use + " report",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := fillPath(pathTemplate, args)
			if err != nil {
				return err
			}
			if start == "" || end == "" {
				endDate := time.Now()
				startDate := endDate.AddDate(0, 0, -days)
				start = startDate.Format("2006-01-02")
				end = endDate.Format("2006-01-02")
			}
			body := map[string]any{
				"startTime":         start,
				"endTime":           end,
				"selector":          map[string]any{"orderBy": []map[string]any{{"field": "localSpend", "sortOrder": "DESCENDING"}}, "pagination": map[string]any{"offset": 0, "limit": 1000}},
				"groupBy":           []string{selector},
				"timeZone":          "UTC",
				"returnRowTotals":   true,
				"returnGrandTotals": true,
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().IntVar(&days, "days", 7, "lookback days when start/end are omitted")
	cmd.Flags().StringVar(&start, "start", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&end, "end", "", "end date YYYY-MM-DD")
	return cmd
}

func customReport(ctx *appContext) *cobra.Command {
	var bodyArg string
	var apply bool
	cmd := &cobra.Command{
		Use:   "custom",
		Short: "Create an async custom report from JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readBody(bodyArg)
			if err != nil {
				return err
			}
			if !apply {
				return ctx.Print(dryRunPayload("POST", "/custom-reports", body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: "/custom-reports", Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&bodyArg, "body", "", "custom report JSON body or @file")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newBudgetCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "budget", Short: "Inspect and create budget orders"}
	cmd.AddCommand(listEndpoint(ctx, "list", "/budgetorders"), idGet(ctx, "get <budget-order-id>", "/budgetorders/%s", false), budgetStatus(ctx), budgetCreate(ctx))
	return cmd
}

func budgetStatus(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Summarize campaign budget health",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			campaigns, err := client.Paginate("/campaigns", nil)
			if err != nil {
				return err
			}
			return ctx.Print(map[string]any{"campaigns": campaigns, "note": "Agents should inspect dailyBudgetAmount and localSpend from reports before increasing budgets."})
		},
	}
}

func budgetCreate(ctx *appContext) *cobra.Command {
	var name, start, end string
	var amount float64
	var apply bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a budget order",
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"name": name, "budgetAmount": money(amount), "startTime": start, "endTime": end}
			if !apply {
				return ctx.Print(dryRunPayload("POST", "/budgetorders", body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: "/budgetorders", Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "budget order name")
	cmd.Flags().Float64Var(&amount, "amount", 0, "budget amount")
	cmd.Flags().StringVar(&start, "start", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&end, "end", "", "end date YYYY-MM-DD")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("start")
	return cmd
}

func newGeoCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "geo", Short: "Search and set geo targeting"}
	cmd.AddCommand(geoSearch(ctx), idGet(ctx, "show <campaign-id>", "/campaigns/%s", false), geoSet(ctx))
	return cmd
}

func geoSearch(ctx *appContext) *cobra.Command {
	var term string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search geo locations",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: "/search/geo", Query: url.Values{"query": []string{term}}})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&term, "query", "", "geo query")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func geoSet(ctx *appContext) *cobra.Command {
	var countries string
	var apply bool
	cmd := &cobra.Command{
		Use:   "set <campaign-id>",
		Short: "Set campaign countriesOrRegions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"countriesOrRegions": parseCSV(countries)}
			path := fmt.Sprintf("/campaigns/%s", args[0])
			if !apply {
				return ctx.Print(dryRunPayload("PUT", path, body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPut, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&countries, "countries", "", "countries, comma separated")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("countries")
	return cmd
}

func newAdsCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "ads", Short: "Manage ad variations and creative assets"}
	cmd.AddCommand(campaignAdGroupList(ctx, "list <campaign-id> <adgroup-id>", "/campaigns/%s/adgroups/%s/ads"))
	cmd.AddCommand(adCreate(ctx))
	cmd.AddCommand(deleteCommand(ctx, "delete <campaign-id> <adgroup-id> <ad-id>", "/campaigns/%s/adgroups/%s/ads/%s"))
	cmd.AddCommand(listEndpoint(ctx, "creatives", "/creatives"))
	cmd.AddCommand(idGet(ctx, "creative <creative-id>", "/creatives/%s", false))
	cmd.AddCommand(appProductPages(ctx))
	cmd.AddCommand(rejections(ctx))
	return cmd
}

func adCreate(ctx *appContext) *cobra.Command {
	var bodyArg string
	var apply bool
	cmd := &cobra.Command{
		Use:   "create <campaign-id> <adgroup-id>",
		Short: "Create an ad from JSON",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readBody(bodyArg)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/campaigns/%s/adgroups/%s/ads", args[0], args[1])
			if !apply {
				return ctx.Print(dryRunPayload("POST", path, body))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: path, Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&bodyArg, "body", "", "ad JSON body or @file")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func appProductPages(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "product-pages",
		Short: "List product pages for the active app",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, _, err := ctx.ActiveApp()
			if err != nil {
				return err
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: fmt.Sprintf("/product-pages/%d", app.ID)})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
}

func rejections(ctx *appContext) *cobra.Command {
	var bodyArg string
	cmd := &cobra.Command{
		Use:   "rejections",
		Short: "Find product page rejection reasons",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readBody(bodyArg)
			if err != nil {
				return err
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: "/product-page-reasons/find", Body: body})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringVar(&bodyArg, "body", "{}", "JSON body or @file")
	return cmd
}

func newOptimizeCommand(ctx *appContext) *cobra.Command {
	var apply bool
	var days int
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Generate an agent-readable optimization plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apply {
				return fmt.Errorf("optimize is intentionally plan-only in the Go agent CLI; execute individual keyword mutations after reviewing the plan")
			}
			return ctx.Print(map[string]any{
				"dry_run": true,
				"days":    days,
				"workflow": []string{
					"Run reports search-terms for Discovery ad groups.",
					"Promote winners with >=2 installs and CPA within goal using keywords add.",
					"Add Discovery negatives for promoted exact terms.",
					"Block losers with spend and zero installs using keywords add-negatives.",
					"Verify with reports summary and keyword reports.",
				},
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 14, "lookback window for the optimization workflow")
	cmd.Flags().BoolVar(&apply, "apply", false, "reserved; optimize is plan-only")
	return cmd
}

func simpleGet(ctx *appContext, use, path string, noOrg bool) *cobra.Command {
	return &cobra.Command{Use: use, RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ctx.Client()
		if err != nil {
			return err
		}
		resp, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: path, SkipOrgContext: noOrg})
		if err != nil {
			return err
		}
		return ctx.Print(resp)
	}}
}

func listEndpoint(ctx *appContext, use, path string) *cobra.Command {
	return &cobra.Command{Use: use, RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ctx.Client()
		if err != nil {
			return err
		}
		items, err := client.Paginate(path, nil)
		if err != nil {
			return err
		}
		return ctx.Print(map[string]any{"data": items, "count": len(items)})
	}}
}

func idGet(ctx *appContext, use, tmpl string, noOrg bool) *cobra.Command {
	return &cobra.Command{Use: use, Args: cobra.ExactArgs(strings.Count(tmpl, "%s")), RunE: func(cmd *cobra.Command, args []string) error {
		path := fmt.Sprintf(tmpl, anyArgs(args)...)
		client, err := ctx.Client()
		if err != nil {
			return err
		}
		resp, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: path, SkipOrgContext: noOrg})
		if err != nil {
			return err
		}
		return ctx.Print(resp)
	}}
}

func campaignChildList(ctx *appContext, use, tmpl string) *cobra.Command {
	return &cobra.Command{Use: use, Args: cobra.ExactArgs(strings.Count(tmpl, "%s")), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ctx.Client()
		if err != nil {
			return err
		}
		items, err := client.Paginate(fmt.Sprintf(tmpl, anyArgs(args)...), nil)
		if err != nil {
			return err
		}
		return ctx.Print(map[string]any{"data": items, "count": len(items)})
	}}
}

func campaignAdGroupList(ctx *appContext, use, tmpl string) *cobra.Command {
	return campaignChildList(ctx, use, tmpl)
}

func statusCommand(ctx *appContext, use, tmpl, status string) *cobra.Command {
	var apply bool
	cmd := &cobra.Command{Use: use, Args: cobra.ExactArgs(strings.Count(tmpl, "%s")), RunE: func(cmd *cobra.Command, args []string) error {
		path := fmt.Sprintf(tmpl, anyArgs(args)...)
		body := map[string]any{"status": status}
		if !apply {
			return ctx.Print(dryRunPayload("PUT", path, body))
		}
		client, err := ctx.Client()
		if err != nil {
			return err
		}
		resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPut, Path: path, Body: body})
		if err != nil {
			return err
		}
		return ctx.Print(resp)
	}}
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	return cmd
}

func deleteCommand(ctx *appContext, use, tmpl string) *cobra.Command {
	var apply bool
	cmd := &cobra.Command{Use: use, Args: cobra.ExactArgs(strings.Count(tmpl, "%s")), RunE: func(cmd *cobra.Command, args []string) error {
		path := fmt.Sprintf(tmpl, anyArgs(args)...)
		if !apply {
			return ctx.Print(dryRunPayload("DELETE", path, nil))
		}
		client, err := ctx.Client()
		if err != nil {
			return err
		}
		resp, err := client.Request(appleads.RequestOptions{Method: http.MethodDelete, Path: path})
		if err != nil {
			return err
		}
		return ctx.Print(resp)
	}}
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutation")
	return cmd
}

func fillPath(tmpl string, args []string) (string, error) {
	want := strings.Count(tmpl, "%s")
	if len(args) != want {
		return "", fmt.Errorf("expected %d path args, got %d", want, len(args))
	}
	return fmt.Sprintf(tmpl, anyArgs(args)...), nil
}

func optionalArgs(tmpl string) string {
	switch strings.Count(tmpl, "%s") {
	case 1:
		return " <campaign-id>"
	case 2:
		return " <campaign-id> <adgroup-id>"
	default:
		return ""
	}
}

func anyArgs(args []string) []any {
	out := make([]any, len(args))
	for i := range args {
		out[i] = args[i]
	}
	return out
}

func money(amount float64) map[string]string {
	return map[string]string{"amount": strconv.FormatFloat(amount, 'f', 2, 64), "currency": "USD"}
}

func idString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	default:
		return fmt.Sprint(x)
	}
}
