package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

var flagTailSource string

var errBothScopes = errors.New("--provider and --target each scope the whole query — use one")

var tailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Follow the event stream live",
	Long: `Follows the activity log as it happens, one line per event, until interrupted.

The instance's feed is unfiltered, so --source is applied here rather than
server-side. With --json each event is printed as its own document, one per
line, which is what a consumer piping into jq wants.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, cfg, err := connect()
		if err != nil {
			return err
		}

		// Fail on the first connection rather than looping silently: a wrong
		// URL or a missing session should be reported, not retried forever.
		if _, err := api.Session(ctx); err != nil {
			return err
		}

		if !flagJSON {
			ui.Step("following %s, press ctrl-c to stop", cfg.URL)
		}

		for {
			err := api.Stream(ctx, func(event client.Event) {
				warnedUnavailable = false
				onStreamed(event)
			})

			// A cancelled context is the user pressing ctrl-c, not a failure —
			// but it still exits 130 rather than 0, so a shell can tell a tail
			// that was stopped from one that ended on its own.
			if ctx.Err() != nil {
				return ErrInterrupted
			}
			if err != nil {
				var apiErr *client.Error
				if errors.As(err, &apiErr) && apiErr.Unauthenticated() {
					return err
				}
				// While the instance restarts — every deploy — the edge answers
				// with whatever it does for a missing backend, commonly a 404.
				// Reporting that verbatim reads like the route is gone, so say
				// what it means instead and stay quiet until it drags on.
				if unavailable(err) {
					if !warnedUnavailable {
						ui.Warn("instance is not answering, retrying until it is")
						warnedUnavailable = true
					}
				} else {
					ui.Warn("stream interrupted, reconnecting — %s", err)
					warnedUnavailable = false
				}
			}

			// The stream also ends cleanly when the instance restarts, which a
			// deploy does routinely. Wait before reconnecting so a server that
			// is down does not become a busy loop.
			select {
			case <-ctx.Done():
				return ErrInterrupted
			case <-time.After(2 * time.Second):
			}
		}
	},
}

// warnedUnavailable keeps a restart from printing one warning every two
// seconds. It resets as soon as the stream comes back or fails differently.
var warnedUnavailable bool

// unavailable reports whether an error means "not there right now" rather than
// "wrong". Those are the ones worth retrying quietly.
func unavailable(err error) bool {
	var apiErr *client.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case http.StatusNotFound, http.StatusBadGateway,
			http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		return false
	}
	// A transport error — connection refused, TLS reset, an HTTP/2 stream torn
	// down mid-response — is the same situation seen one layer lower.
	return true
}

func onStreamed(event client.Event) {
	if flagTailSource != "" && event.Source != flagTailSource {
		return
	}

	if flagJSON {
		// One document per line, not a growing array: a stream has no end at
		// which an array could be closed.
		encoded, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintln(os.Stdout, string(encoded))
		return
	}

	ui.Plain("%s  %-8s %-18s %-46s %s",
		ui.Dim(clockTime(event.Timestamp)),
		event.Source,
		truncate(event.ProviderName, 18),
		truncate(eventTitle(event), 46),
		deliverySummary(event),
	)
}

func init() {
	tailCmd.Flags().StringVar(&flagTailSource, "source", "", "Only show one source: webhook, website, imap, rss, system, test")
	rootCmd.AddCommand(tailCmd)
}
