package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/nook-cli/internal/ui"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List the configured sources",
	Long: `Lists every provider the instance watches or receives from, with the address it
uses and the tags a delivery target can route on.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, _, err := connect()
		if err != nil {
			return err
		}

		payload, err := api.Settings(ctx)
		if err != nil {
			return err
		}
		providers := payload.Settings.Providers.Items

		if flagJSON {
			return ui.JSON(providers)
		}
		if len(providers) == 0 {
			ui.Plain("No providers")
			return nil
		}

		rows := make([][]string, 0, len(providers))
		for _, provider := range providers {
			state := ui.Idle()
			if provider.Enabled {
				state = ui.Live()
			}
			rows = append(rows, []string{
				state,
				provider.Name,
				provider.Type,
				provider.Endpoint(),
				strings.Join(provider.Tags, ", "),
			})
		}
		ui.Table([]string{"", "NAME", "TYPE", "ENDPOINT", "TAGS"}, rows)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(providersCmd)
}
