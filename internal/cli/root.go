package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	ctx := &appContext{}
	root := &cobra.Command{
		Use:   "ads",
		Short: "Agent-first CLI for Apple Ads",
		Long:  "ads is an agent-first command line wrapper for the Apple Ads API. Mutations dry-run by default and structured output is available everywhere.",
	}
	root.PersistentFlags().StringVar(&ctx.ConfigDir, "config-dir", "", "config directory; defaults to ~/.apple-ads-cli")
	root.PersistentFlags().StringVar(&ctx.AppSlug, "app", "", "configured app slug to operate on")
	root.PersistentFlags().BoolVar(&ctx.JSON, "json", false, "emit compact JSON")
	root.PersistentFlags().BoolVar(&ctx.Pretty, "pretty", true, "pretty-print JSON output")
	root.AddCommand(
		newConfigCommand(ctx),
		newAPICommand(ctx),
		newACLCommand(ctx),
		newCampaignsCommand(ctx),
		newAdGroupsCommand(ctx),
		newKeywordsCommand(ctx),
		newReportsCommand(ctx),
		newBudgetCommand(ctx),
		newGeoCommand(ctx),
		newAdsCommand(ctx),
		newOptimizeCommand(ctx),
	)
	return root
}
