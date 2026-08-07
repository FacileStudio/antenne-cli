package cmd

import (
	"github.com/spf13/cobra"

	"github.com/FacileStudio/antenne-cli/internal/ui"
)

var replayCmd = &cobra.Command{
	Use:   "replay <event> <target>",
	Short: "Send a logged event again",
	Long: `Re-sends the stored envelope of a logged event to one delivery target, bypassing
the routing rules. Only events that carry an envelope can be replayed.

The target is matched by id, or by name when that is unambiguous. Event ids come
from ` + "`antenne events --json`" + `.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, _, err := connect()
		if err != nil {
			return err
		}

		target, err := resolveTarget(ctx, api, args[1])
		if err != nil {
			return err
		}

		if err := api.Replay(ctx, args[0], target.ID); err != nil {
			return err
		}
		ui.Success("replayed to %s", target.Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(replayCmd)
}
