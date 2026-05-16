package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dannolan/apple-ads-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage local Apple Ads credentials and apps"}
	var orgID int64
	var clientID, teamID, keyID, privateKey string
	init := &cobra.Command{
		Use:   "init",
		Short: "Write API credentials non-interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := ctx.Store()
			if err != nil {
				return err
			}
			creds := config.Credentials{OrgID: orgID, ClientID: clientID, TeamID: teamID, KeyID: keyID, PrivateKeyPath: privateKey}
			if err := store.SaveCredentials(creds); err != nil {
				return err
			}
			return ctx.Print(map[string]any{"ok": true, "config_dir": store.Dir(), "credentials": "saved"})
		},
	}
	init.Flags().Int64Var(&orgID, "org-id", 0, "Apple Ads org ID")
	init.Flags().StringVar(&clientID, "client-id", "", "Apple Ads client ID")
	init.Flags().StringVar(&teamID, "team-id", "", "Apple Ads team ID")
	init.Flags().StringVar(&keyID, "key-id", "", "Apple Ads key ID")
	init.Flags().StringVar(&privateKey, "private-key", "", "path to EC private key PEM")
	_ = init.MarkFlagRequired("org-id")
	_ = init.MarkFlagRequired("client-id")
	_ = init.MarkFlagRequired("team-id")
	_ = init.MarkFlagRequired("key-id")
	_ = init.MarkFlagRequired("private-key")

	show := &cobra.Command{
		Use:   "show",
		Short: "Show local config with secrets redacted",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := ctx.Store()
			if err != nil {
				return err
			}
			creds, _ := store.LoadCredentials()
			apps, _ := store.LoadApps()
			if creds.ClientID != "" {
				creds.ClientID = redact(creds.ClientID)
				creds.PrivateKeyPath = redactPath(creds.PrivateKeyPath)
			}
			return ctx.Print(map[string]any{"config_dir": store.Dir(), "credentials": creds, "apps": apps})
		},
	}
	test := &cobra.Command{
		Use:   "test",
		Short: "Verify OAuth and API access",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(request("GET", "/me", nil, nil, true))
			if err != nil {
				return err
			}
			return ctx.Print(map[string]any{"ok": true, "me": resp["data"]})
		},
	}
	fromOnePassword := onePasswordCommand(ctx)

	app := &cobra.Command{Use: "app", Short: "Manage configured apps"}
	var appID int64
	var name, countries, currency string
	var bid, cpa float64
	add := &cobra.Command{
		Use:   "add",
		Short: "Add an app profile for agent workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := ctx.Store()
			if err != nil {
				return err
			}
			apps, err := store.LoadApps()
			if err != nil {
				return err
			}
			slug := config.Slug(name)
			apps.Apps[slug] = config.App{ID: appID, Name: name, DefaultCountries: parseCSV(countries), DefaultBid: bid, DefaultCPAGoal: cpa, DefaultCurrency: normalizeCurrency(currency)}
			apps.ActiveApp = slug
			if err := store.SaveApps(apps); err != nil {
				return err
			}
			return ctx.Print(map[string]any{"ok": true, "active_app": slug, "app": apps.Apps[slug]})
		},
	}
	add.Flags().Int64Var(&appID, "app-id", 0, "Apple app adam ID")
	add.Flags().StringVar(&name, "name", "", "app name")
	add.Flags().StringVar(&countries, "countries", "US", "default countries, comma separated")
	add.Flags().StringVar(&currency, "currency", "USD", "default money currency")
	add.Flags().Float64Var(&bid, "bid", 1.50, "default keyword bid")
	add.Flags().Float64Var(&cpa, "cpa-goal", 0, "default CPA goal")
	_ = add.MarkFlagRequired("app-id")
	_ = add.MarkFlagRequired("name")
	list := &cobra.Command{
		Use:   "list",
		Short: "List configured apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := ctx.Store()
			if err != nil {
				return err
			}
			apps, err := store.LoadApps()
			if err != nil {
				return err
			}
			return ctx.Print(apps)
		},
	}
	use := &cobra.Command{
		Use:   "use <slug>",
		Short: "Set active app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := ctx.Store()
			if err != nil {
				return err
			}
			apps, err := store.LoadApps()
			if err != nil {
				return err
			}
			if _, ok := apps.Apps[args[0]]; !ok {
				return fmt.Errorf("unknown app %q", args[0])
			}
			apps.ActiveApp = args[0]
			if err := store.SaveApps(apps); err != nil {
				return err
			}
			return ctx.Print(map[string]any{"ok": true, "active_app": args[0]})
		},
	}
	app.AddCommand(add, list, use)
	cmd.AddCommand(init, show, test, fromOnePassword, app)
	return cmd
}

func onePasswordCommand(ctx *appContext) *cobra.Command {
	var vault, item, keyDocument, keyPath string
	var appName, countries, currency string
	var appID int64
	var bid, cpa float64
	cmd := &cobra.Command{
		Use:   "from-1password",
		Short: "Hydrate ads config from 1Password CLI secret references",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := ctx.Store()
			if err != nil {
				return err
			}
			if _, err := exec.LookPath("op"); err != nil {
				return fmt.Errorf("1Password CLI not found in PATH: %w", err)
			}
			if err := os.MkdirAll(store.Dir(), 0o700); err != nil {
				return err
			}
			if keyPath == "" {
				keyPath = filepath.Join(store.Dir(), "private-key.pem")
			}
			if keyDocument != "" {
				op := exec.Command("op", "document", "get", keyDocument, "--vault", vault, "--out-file", keyPath, "--file-mode", "0600", "--force")
				if data, err := op.CombinedOutput(); err != nil {
					return fmt.Errorf("download private key from 1Password: %w: %s", err, trimCommandOutput(data))
				}
			}
			orgIDRaw, err := opRead(vault, item, "org_id")
			if err != nil {
				return err
			}
			orgID, err := strconv.ParseInt(orgIDRaw, 10, 64)
			if err != nil {
				return fmt.Errorf("parse org_id from 1Password: %w", err)
			}
			clientID, err := opRead(vault, item, "client_id")
			if err != nil {
				return err
			}
			teamID, err := opRead(vault, item, "team_id")
			if err != nil {
				return err
			}
			keyID, err := opRead(vault, item, "key_id")
			if err != nil {
				return err
			}
			creds := config.Credentials{OrgID: orgID, ClientID: clientID, TeamID: teamID, KeyID: keyID, PrivateKeyPath: keyPath}
			if err := store.SaveCredentials(creds); err != nil {
				return err
			}
			result := map[string]any{"ok": true, "config_dir": store.Dir(), "credentials": "saved", "private_key_path": "<redacted-path>"}
			if appID != 0 && appName != "" {
				apps, err := store.LoadApps()
				if err != nil {
					return err
				}
				slug := config.Slug(appName)
				apps.Apps[slug] = config.App{ID: appID, Name: appName, DefaultCountries: parseCSV(countries), DefaultBid: bid, DefaultCPAGoal: cpa, DefaultCurrency: normalizeCurrency(currency)}
				apps.ActiveApp = slug
				if err := store.SaveApps(apps); err != nil {
					return err
				}
				result["active_app"] = slug
			}
			return ctx.Print(result)
		},
	}
	cmd.Flags().StringVar(&vault, "vault", "Private", "1Password vault")
	cmd.Flags().StringVar(&item, "item", "Apple Ads API", "1Password item containing org_id, client_id, team_id, key_id")
	cmd.Flags().StringVar(&keyDocument, "key-document", "Apple Ads API Private Key", "1Password document containing private-key.pem; empty to skip download")
	cmd.Flags().StringVar(&keyPath, "private-key", "", "local private key output path; defaults to config dir")
	cmd.Flags().Int64Var(&appID, "app-id", 0, "optional app adam ID to configure")
	cmd.Flags().StringVar(&appName, "app-name", "", "optional app name to configure")
	cmd.Flags().StringVar(&countries, "countries", "US", "default app countries")
	cmd.Flags().StringVar(&currency, "currency", "USD", "default app currency")
	cmd.Flags().Float64Var(&bid, "bid", 1.50, "default app bid")
	cmd.Flags().Float64Var(&cpa, "cpa-goal", 0, "default app CPA goal")
	return cmd
}

func normalizeCurrency(currency string) string {
	currency = strings.TrimSpace(strings.ToUpper(currency))
	if currency == "" {
		return "USD"
	}
	return currency
}

func opRead(vault, item, field string) (string, error) {
	cmd := exec.Command("op", "read", opRef(vault, item, field))
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read %s from 1Password: %w: %s", field, err, trimCommandOutput(data))
	}
	return trimCommandOutput(data), nil
}

func redact(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func redactPath(s string) string {
	if s == "" {
		return ""
	}
	return "<redacted-path>"
}
