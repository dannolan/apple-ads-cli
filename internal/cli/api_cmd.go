package cli

import (
	"net/http"

	"github.com/dannolan/apple-ads-cli/internal/appleads"
	"github.com/spf13/cobra"
)

func newAPICommand(ctx *appContext) *cobra.Command {
	var query []string
	var body string
	var apply bool
	var noOrg bool
	cmd := &cobra.Command{
		Use:   "api <method> <path>",
		Short: "Raw Apple Ads API escape hatch for agents",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := parseQuery(query)
			if err != nil {
				return err
			}
			parsedBody, err := readBody(body)
			if err != nil {
				return err
			}
			method := args[0]
			path := args[1]
			if method != http.MethodGet && method != http.MethodHead && !apply {
				return ctx.Print(dryRunPayload(method, path, parsedBody))
			}
			client, err := ctx.Client()
			if err != nil {
				return err
			}
			resp, err := client.Request(appleads.RequestOptions{Method: method, Path: path, Query: q, Body: parsedBody, SkipOrgContext: noOrg})
			if err != nil {
				return err
			}
			return ctx.Print(resp)
		},
	}
	cmd.Flags().StringArrayVarP(&query, "query", "q", nil, "query parameter key=value; repeatable")
	cmd.Flags().StringVar(&body, "body", "", "JSON body, or @path/to/body.json")
	cmd.Flags().BoolVar(&apply, "apply", false, "execute mutating request")
	cmd.Flags().BoolVar(&noOrg, "no-org-context", false, "omit X-AP-Context")
	return cmd
}

func request(method, path string, q any, body any, noOrg bool) appleads.RequestOptions {
	return appleads.RequestOptions{Method: method, Path: path, Body: body, SkipOrgContext: noOrg}
}
