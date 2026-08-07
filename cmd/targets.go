package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

var targetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "List the delivery targets",
	Long: `Lists every delivery target and what routes to it. Antenne's routing is opt-in with
no fallthrough: a target naming no provider and no tag receives nothing, and is
called out here as such.`,
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
		targets := payload.Settings.Delivery.Items

		if flagJSON {
			return ui.JSON(targets)
		}
		if len(targets) == 0 {
			ui.Plain("No delivery targets")
			return nil
		}

		rows := make([][]string, 0, len(targets))
		for _, target := range targets {
			state := ui.Idle()
			if target.Enabled {
				state = ui.Live()
			}
			rows = append(rows, []string{
				state,
				target.Name,
				target.Type,
				routedBy(target),
			})
		}
		ui.Table([]string{"", "NAME", "TYPE", "ROUTED BY"}, rows)
		return nil
	},
}

// routedBy describes what selects a target, in the terms the operator set it
// up with rather than as raw ids.
func routedBy(target client.Target) string {
	if target.Silent() {
		return ui.Failed("nothing attached")
	}

	parts := make([]string, 0, 2)
	if count := len(target.ProviderIDs); count > 0 {
		parts = append(parts, fmt.Sprintf("%d provider%s", count, plural(count)))
	}
	if tags := target.ProviderTags; len(tags) > 0 {
		parts = append(parts, "tags: "+strings.Join(tags, ", "))
	}
	return strings.Join(parts, ", ")
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func init() {
	rootCmd.AddCommand(targetsCmd)
}
