package cmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/nook-cli/internal/ui"
)

var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Show the Nook Pool's connected apps",
	Long: `Reports the pool's boot epoch and every app currently connected, with the
channels it subscribed to.

The epoch changes on every restart. An app that reconnects and sees a new one
knows to resync from its own cursor rather than trust its offsets across the
gap.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, _, err := connect()
		if err != nil {
			return err
		}

		stats, err := api.PoolStats(ctx)
		if err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(stats)
		}

		ui.Fields([][2]string{
			{"epoch", shortEpoch(stats.Epoch)},
			{"connections", strconv.Itoa(stats.Connections)},
		})

		if len(stats.Apps) == 0 {
			ui.Plain("")
			ui.Plain("No apps connected")
			return nil
		}

		rows := make([][]string, 0, len(stats.Apps))
		for _, app := range stats.Apps {
			rows = append(rows, []string{
				ui.Live(),
				app.App,
				strconv.Itoa(len(app.Channels)),
				truncate(strings.Join(app.Channels, ", "), 48),
			})
		}
		ui.Plain("")
		ui.Table([]string{"", "APP", "CHANNELS", "SUBSCRIBED TO"}, rows)
		return nil
	},
}

// shortEpoch trims the boot id to its first segment. The whole value is a UUID
// and only its identity matters — whether it changed since last time.
func shortEpoch(epoch string) string {
	if index := strings.Index(epoch, "-"); index > 0 {
		return epoch[:index]
	}
	return epoch
}

func init() {
	rootCmd.AddCommand(poolCmd)
}
