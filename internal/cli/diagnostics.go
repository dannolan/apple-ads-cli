package cli

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dannolan/apple-ads-cli/internal/appleads"
	"github.com/spf13/cobra"
)

func campaignHealth(ctx *appContext) *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Read-only campaign health check for live Apple Ads accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			out, err := buildCampaignHealth(client, days)
			if err != nil {
				return err
			}
			return ctx.Print(out)
		},
	}
	cmd.Flags().IntVar(&days, "days", 3, "lookback window for report-backed checks")
	return cmd
}

func newAccountCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "account", Short: "Account-level diagnostics and snapshots"}
	cmd.AddCommand(accountSnapshot(ctx))
	return cmd
}

func accountSnapshot(ctx *appContext) *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Read-only account snapshot for agent handoff",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, slug, err := ctx.ActiveApp()
			if err != nil {
				return err
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			out, err := buildAccountSnapshot(client, app, slug, days)
			if err != nil {
				return err
			}
			return ctx.Print(out)
		},
	}
	cmd.Flags().IntVar(&days, "days", 7, "lookback window for report metrics")
	return cmd
}

func reportDiagnose(ctx *appContext) *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "diagnose <campaign-id>",
		Short: "Explain why a campaign may not be serving or converting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			out, err := diagnoseCampaign(client, args[0], days)
			if err != nil {
				return err
			}
			return ctx.Print(out)
		},
	}
	cmd.Flags().IntVar(&days, "days", 7, "lookback window for report metrics")
	return cmd
}

func buildCampaignHealth(client *appleads.Client, days int) (map[string]any, error) {
	campaigns, err := client.Paginate("/campaigns", nil)
	if err != nil {
		return nil, err
	}
	rows, err := runReportRows(client, "/reports/campaigns", days)
	if err != nil {
		return nil, err
	}
	reportByCampaign := rowsByID(rows, "campaignId")
	findings := []map[string]any{}
	typeCounts := map[string]int{}
	runningTypes := map[string]int{}
	runningCampaigns := 0
	for _, c := range campaigns {
		id := idString(c["id"])
		typ := campaignType(c)
		if typ != "" {
			typeCounts[typ]++
		}
		running := isRunning(c)
		if running {
			runningCampaigns++
			if typ != "" {
				runningTypes[typ]++
			}
		}
		row := reportByCampaign[id]
		impressions := intMetric(row, "impressions")
		spend := floatMetric(row, "localSpend")
		name := fmt.Sprint(c["name"])
		if running && impressions == 0 {
			findings = append(findings, finding("warning", "running_zero_impressions", "Campaign is enabled or serving but has zero impressions in the lookback window", id, map[string]any{"name": name}))
		}
		if !running && spend > 0 {
			findings = append(findings, finding("warning", "paused_campaign_spend", "Campaign is not running but has spend in the lookback window", id, map[string]any{"name": name, "localSpend": spend}))
		}
		if channel := firstString(row, "adChannelType"); channel != "" && channel != firstString(c, "adChannelType") {
			findings = append(findings, finding("warning", "report_channel_mismatch", "Report adChannelType differs from the campaign object", id, map[string]any{"name": name, "campaign_channel": c["adChannelType"], "report_channel": channel}))
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), "archived") && running {
			findings = append(findings, finding("warning", "archived_campaign_running", "Archived campaign is enabled or serving", id, map[string]any{"name": name}))
		}
	}
	if len(runningTypes) < 4 {
		findings = append(findings, finding("info", "not_all_campaign_types_running", "Not all Brand/Category/Competitor/Discovery campaign types are running", "", map[string]any{"running_types": runningTypes}))
	}
	return map[string]any{
		"ok":        noWarnings(findings),
		"days":      days,
		"summary":   map[string]any{"campaign_count": len(campaigns), "running_campaign_count": runningCampaigns, "campaign_types": typeCounts, "running_campaign_types": runningTypes},
		"findings":  findings,
		"campaigns": campaigns,
		"reports":   rows,
	}, nil
}

func buildAccountSnapshot(client *appleads.Client, app any, slug string, days int) (map[string]any, error) {
	campaigns, err := client.Paginate("/campaigns", nil)
	if err != nil {
		return nil, err
	}
	campaignReports, err := runReportRows(client, "/reports/campaigns", days)
	if err != nil {
		return nil, err
	}
	campaignDetails := []map[string]any{}
	for _, c := range campaigns {
		campaignID := idString(c["id"])
		adgroups, _ := client.Paginate(fmt.Sprintf("/campaigns/%s/adgroups", campaignID), nil)
		negatives, _ := client.Paginate(fmt.Sprintf("/campaigns/%s/negativekeywords", campaignID), nil)
		adgroupSummaries := []map[string]any{}
		for _, adgroup := range adgroups {
			adgroupID := idString(adgroup["id"])
			keywords, _ := client.Paginate(fmt.Sprintf("/campaigns/%s/adgroups/%s/targetingkeywords", campaignID, adgroupID), nil)
			adgroupSummaries = append(adgroupSummaries, map[string]any{"adgroup": adgroup, "keyword_count": len(keywords), "keywords": keywords})
		}
		campaignDetails = append(campaignDetails, map[string]any{"campaign": c, "adgroups": adgroupSummaries, "negative_keyword_count": len(negatives), "negative_keywords": negatives})
	}
	return map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"days":         days,
		"app_slug":     slug,
		"app":          app,
		"campaigns":    campaignDetails,
		"reports":      campaignReports,
	}, nil
}

func diagnoseCampaign(client *appleads.Client, campaignID string, days int) (map[string]any, error) {
	campaignResp, err := client.Request(appleads.RequestOptions{Method: http.MethodGet, Path: fmt.Sprintf("/campaigns/%s", campaignID)})
	if err != nil {
		return nil, err
	}
	campaign := resourceMap(campaignResp)
	adgroups, err := client.Paginate(fmt.Sprintf("/campaigns/%s/adgroups", campaignID), nil)
	if err != nil {
		return nil, err
	}
	keywordsByAdgroup := map[string]int{}
	runningAdgroups := 0
	findings := []map[string]any{}
	for _, adgroup := range adgroups {
		adgroupID := idString(adgroup["id"])
		if isRunning(adgroup) {
			runningAdgroups++
		}
		keywords, _ := client.Paginate(fmt.Sprintf("/campaigns/%s/adgroups/%s/targetingkeywords", campaignID, adgroupID), nil)
		keywordsByAdgroup[adgroupID] = len(keywords)
		if len(keywords) == 0 && !boolField(adgroup, "automatedKeywordsOptIn") {
			findings = append(findings, finding("warning", "adgroup_no_keywords", "Ad group has no targeting keywords and Search Match is off", campaignID, map[string]any{"adgroup_id": adgroupID, "adgroup_name": adgroup["name"]}))
		}
	}
	negatives, err := client.Paginate(fmt.Sprintf("/campaigns/%s/negativekeywords", campaignID), nil)
	if err != nil {
		return nil, err
	}
	rows, err := runReportRows(client, fmt.Sprintf("/reports/campaigns/%s/adgroups", campaignID), days)
	if err != nil {
		return nil, err
	}
	if !isRunning(campaign) {
		findings = append(findings, finding("warning", "campaign_not_running", "Campaign status/displayStatus/servingStatus is not running", campaignID, nil))
	}
	if runningAdgroups == 0 {
		findings = append(findings, finding("warning", "no_running_adgroups", "Campaign has no running ad groups", campaignID, nil))
	}
	if reportTotals(rows, "impressions") == 0 {
		findings = append(findings, finding("warning", "zero_impressions", "Campaign ad groups have zero impressions in the lookback window", campaignID, nil))
	}
	return map[string]any{
		"ok":                  noWarnings(findings),
		"days":                days,
		"campaign":            campaign,
		"adgroup_count":       len(adgroups),
		"running_adgroups":    runningAdgroups,
		"keywords_by_adgroup": keywordsByAdgroup,
		"negative_keywords":   len(negatives),
		"reports":             rows,
		"findings":            findings,
	}, nil
}

func buildOptimizePlan(client *appleads.Client, goal float64, days int) (map[string]any, error) {
	campaigns, err := client.Paginate("/campaigns", nil)
	if err != nil {
		return nil, err
	}
	promotions := []map[string]any{}
	negatives := []map[string]any{}
	for _, campaign := range campaigns {
		if campaignType(campaign) != "discovery" {
			continue
		}
		campaignID := idString(campaign["id"])
		rows, err := runReportRows(client, fmt.Sprintf("/reports/campaigns/%s/searchterms", campaignID), days)
		if err != nil {
			return nil, err
		}
		seenPromotionTerms := map[string]bool{}
		seenNegativeTerms := map[string]bool{}
		for _, row := range rows {
			term := strings.TrimSpace(firstString(row, "searchTermText", "searchTerm", "query"))
			if term == "" {
				continue
			}
			installs := intMetric(row, "tapInstalls")
			spend := floatMetric(row, "localSpend")
			cpi := floatMetric(row, "tapInstallCPI")
			if cpi == 0 && installs > 0 {
				cpi = spend / float64(installs)
			}
			base := map[string]any{"campaign_id": campaignID, "campaign_name": campaign["name"], "search_term": term, "installs": installs, "localSpend": spend, "cpi": cpi}
			promotionKey := strings.ToLower(term)
			if installs >= 2 && (goal == 0 || cpi <= goal) && !seenPromotionTerms[promotionKey] {
				promotions = append(promotions, base)
				seenPromotionTerms[promotionKey] = true
			}
			negativeKey := promotionKey
			if installs == 0 && spend > 0 && !seenNegativeTerms[negativeKey] {
				negatives = append(negatives, base)
				seenNegativeTerms[negativeKey] = true
			}
		}
	}
	sort.Slice(promotions, func(i, j int) bool {
		return floatMetric(promotions[i], "localSpend") > floatMetric(promotions[j], "localSpend")
	})
	sort.Slice(negatives, func(i, j int) bool {
		return floatMetric(negatives[i], "localSpend") > floatMetric(negatives[j], "localSpend")
	})
	return map[string]any{
		"dry_run":                 true,
		"days":                    days,
		"cpa_goal":                goal,
		"promote_exact_terms":     promotions,
		"add_discovery_negatives": promotions,
		"block_spend_no_install":  negatives,
		"commands":                optimizeCommands(promotions, negatives),
	}, nil
}

func optimizeCommands(promotions, negatives []map[string]any) []string {
	out := []string{}
	for _, item := range promotions {
		term := strings.ReplaceAll(fmt.Sprint(item["search_term"]), `"`, `\"`)
		out = append(out, fmt.Sprintf("ads keywords add <category-campaign-id> <category-adgroup-id> --text %q --match EXACT --bid <bid> --skip-existing --apply", term))
		out = append(out, fmt.Sprintf("ads keywords add-negatives %s --text %q --match EXACT --apply", item["campaign_id"], term))
	}
	for _, item := range negatives {
		term := strings.ReplaceAll(fmt.Sprint(item["search_term"]), `"`, `\"`)
		out = append(out, fmt.Sprintf("ads keywords add-negatives %s --text %q --match EXACT --apply", item["campaign_id"], term))
	}
	return out
}

func runReportRows(client *appleads.Client, path string, days int) ([]map[string]any, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)
	body := buildReportBody(startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), path)
	resp, err := client.Request(appleads.RequestOptions{Method: http.MethodPost, Path: path, Body: body})
	if err != nil {
		return nil, err
	}
	if rows, ok := tableRows(resp); ok {
		return rows, nil
	}
	return []map[string]any{}, nil
}

func rowsByID(rows []map[string]any, key string) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, row := range rows {
		out[idString(row[key])] = row
	}
	return out
}

func finding(severity, code, message, campaignID string, details map[string]any) map[string]any {
	out := map[string]any{"severity": severity, "code": code, "message": message}
	if campaignID != "" {
		out["campaign_id"] = campaignID
	}
	if len(details) > 0 {
		out["details"] = details
	}
	return out
}

func noWarnings(findings []map[string]any) bool {
	for _, f := range findings {
		if f["severity"] == "warning" || f["severity"] == "error" {
			return false
		}
	}
	return true
}

func campaignType(c map[string]any) string {
	name := strings.ToLower(strings.TrimSpace(fmt.Sprint(c["name"])))
	name = strings.TrimPrefix(name, "archived - ")
	for _, typ := range []string{"brand", "category", "competitor", "discovery"} {
		if strings.Contains(name, typ) {
			return typ
		}
	}
	return ""
}

func isRunning(m map[string]any) bool {
	for _, key := range []string{"displayStatus", "servingStatus", "status"} {
		value := strings.ToUpper(strings.TrimSpace(fmt.Sprint(m[key])))
		if value == "RUNNING" || value == "ENABLED" {
			return true
		}
	}
	return false
}

func boolField(m map[string]any, key string) bool {
	v, ok := m[key].(bool)
	return ok && v
}

func intMetric(m map[string]any, key string) int {
	return int(floatMetric(m, key))
}

func reportTotals(rows []map[string]any, key string) int {
	total := 0
	for _, row := range rows {
		total += intMetric(row, key)
	}
	return total
}

func floatMetric(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	case map[string]any:
		return floatMetric(v, "amount")
	case map[string]string:
		n, _ := strconv.ParseFloat(v["amount"], 64)
		return n
	default:
		return 0
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fmt.Sprint(m[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func resourceMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		if data, ok := m["data"].(map[string]any); ok {
			return data
		}
		return m
	}
	if values, ok := value.([]any); ok && len(values) > 0 {
		if m, ok := values[0].(map[string]any); ok {
			if data, ok := m["data"].(map[string]any); ok {
				return data
			}
			return m
		}
	}
	return map[string]any{}
}

func filterExistingKeywords(items, existing []map[string]any) ([]map[string]any, []map[string]any) {
	seen := map[string]bool{}
	for _, keyword := range existing {
		seen[keywordKey(keyword)] = true
	}
	planned := []map[string]any{}
	skipped := []map[string]any{}
	for _, item := range items {
		if seen[keywordKey(item)] {
			skipped = append(skipped, item)
			continue
		}
		planned = append(planned, item)
	}
	return planned, skipped
}

func keywordKey(m map[string]any) string {
	return strings.ToLower(strings.TrimSpace(fmt.Sprint(m["text"]))) + "|" + strings.ToUpper(strings.TrimSpace(fmt.Sprint(m["matchType"])))
}
