package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func NewRootCommand() *cobra.Command {
	cmd, _ := NewRootCommandWithContext()
	return cmd
}

func NewRootCommandWithContext() (*cobra.Command, *appContext) {
	ctx := &appContext{}
	root := &cobra.Command{
		Use:           "ads",
		Short:         "Agent-first CLI for Apple Ads",
		Long:          "ads is an agent-first command line wrapper for the Apple Ads API. Mutations dry-run by default and structured output is available everywhere.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&ctx.ConfigDir, "config-dir", "", "config directory; defaults to ~/.apple-ads-cli")
	root.PersistentFlags().StringVar(&ctx.AppSlug, "app", "", "configured app slug to operate on")
	root.PersistentFlags().BoolVar(&ctx.JSON, "json", false, "emit compact JSON")
	root.PersistentFlags().BoolVar(&ctx.Pretty, "pretty", true, "pretty-print JSON output")
	root.PersistentFlags().BoolVar(&ctx.Table, "table", false, "emit table output for common list and report responses")
	root.AddCommand(
		newConfigCommand(ctx),
		newAPICommand(ctx),
		newACLCommand(ctx),
		newCampaignsCommand(ctx),
		newAdGroupsCommand(ctx),
		newKeywordsCommand(ctx),
		newReportsCommand(ctx),
		newAccountCommand(ctx),
		newBudgetCommand(ctx),
		newGeoCommand(ctx),
		newAdsCommand(ctx),
		newOptimizeCommand(ctx),
		newSmokeCommand(ctx),
		newVersionCommand(ctx),
	)
	root.AddCommand(newManifestCommand(ctx, root))
	return root, ctx
}

func newVersionCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ctx.Print(map[string]any{
				"version": Version,
				"commit":  Commit,
				"date":    Date,
			})
		},
	}
}

func newManifestCommand(ctx *appContext, root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "manifest",
		Short: "Print an agent-readable command manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			commands := []map[string]any{}
			var walk func(*cobra.Command)
			walk = func(parent *cobra.Command) {
				for _, child := range parent.Commands() {
					if child.Hidden {
						continue
					}
					flags := []string{}
					child.Flags().VisitAll(func(flag *pflag.Flag) {
						flags = append(flags, "--"+flag.Name)
					})
					commands = append(commands, map[string]any{
						"path":     child.CommandPath(),
						"use":      child.UseLine(),
						"short":    child.Short,
						"mutation": child.Flags().Lookup("apply") != nil,
						"flags":    flags,
					})
					walk(child)
				}
			}
			walk(root)
			return ctx.Print(map[string]any{
				"name":     "apple-ads-cli",
				"binary":   "ads",
				"version":  Version,
				"commands": commands,
				"policy": map[string]any{
					"json":             "Use --json for agent-readable output.",
					"mutation_safety":  "Mutating commands dry-run by default and require --apply.",
					"raw_api_fallback": "Use ads api for endpoints that are not wrapped yet.",
				},
			})
		},
	}
}
