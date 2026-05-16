package cli

import (
	"fmt"

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

	app := &cobra.Command{Use: "app", Short: "Manage configured apps"}
	var appID int64
	var name, countries string
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
			apps.Apps[slug] = config.App{ID: appID, Name: name, DefaultCountries: parseCSV(countries), DefaultBid: bid, DefaultCPAGoal: cpa}
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
	cmd.AddCommand(init, show, test, app)
	return cmd
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
