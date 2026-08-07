package cmd

import (
	"github.com/spf13/cobra"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

var (
	flagLimit    int
	flagSource   string
	flagSearch   string
	flagProvider string
	flagTarget   string
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List recent events",
	Long: `Lists the activity log, newest first. Filters compose except --provider and
--target, which each scope the whole query and so are mutually exclusive.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		if flagProvider != "" && flagTarget != "" {
			return errBothScopes
		}

		api, _, err := connect()
		if err != nil {
			return err
		}

		page, err := api.Events(ctx, client.EventQuery{
			Limit:      flagLimit,
			Search:     flagSearch,
			Source:     flagSource,
			ProviderID: flagProvider,
			TargetID:   flagTarget,
		})
		if err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(page)
		}
		if len(page.Events) == 0 {
			ui.Plain("No events")
			return nil
		}

		rows := make([][]string, 0, len(page.Events))
		for _, event := range page.Events {
			rows = append(rows, []string{
				relativeTime(event.Timestamp),
				event.Source,
				truncate(event.ProviderName, 20),
				truncate(eventTitle(event), 48),
				deliverySummary(event),
			})
		}
		ui.Table([]string{"WHEN", "SOURCE", "PROVIDER", "EVENT", "DELIVERY"}, rows)

		// The page is a window onto a filtered set; saying so stops "20 events"
		// being read as "20 events exist".
		if page.Total > len(page.Events) {
			ui.Hint("showing %d of %d — raise --limit to see more", len(page.Events), page.Total)
		}
		return nil
	},
}

func init() {
	eventsCmd.Flags().IntVarP(&flagLimit, "limit", "n", 20, "How many events to show")
	eventsCmd.Flags().StringVar(&flagSource, "source", "", "Filter by source: webhook, website, imap, rss, system, test")
	eventsCmd.Flags().StringVarP(&flagSearch, "search", "s", "", "Full-text search across the log")
	eventsCmd.Flags().StringVar(&flagProvider, "provider", "", "Scope to one provider id")
	eventsCmd.Flags().StringVar(&flagTarget, "target", "", "Scope to one delivery target id")
	rootCmd.AddCommand(eventsCmd)
}
