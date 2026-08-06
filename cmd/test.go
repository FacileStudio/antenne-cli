package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/nook-cli/internal/client"
	"github.com/FacileStudio/nook-cli/internal/ui"
)

var testCmd = &cobra.Command{
	Use:   "test [target]",
	Short: "Send a test delivery",
	Long: `With a target, sends straight to it, bypassing the routing rules — this is what
answers "does this target work at all". The target is matched by id, or by name
when that is unambiguous.

With no target, sends one alert through the whole pipeline instead, routing
rules included. That stays silent unless a target has opted into Nook's own
events by selecting the system provider, which is the point: it tests the
routing, not the channel.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, _, err := connect()
		if err != nil {
			return err
		}

		if len(args) == 0 {
			if err := api.TestAlert(ctx); err != nil {
				return err
			}
			ui.Success("test alert sent through the pipeline")
			ui.Hint("nothing arrived? only targets that selected the system provider receive it")
			return nil
		}

		target, err := resolveTarget(ctx, api, args[0])
		if err != nil {
			return err
		}

		ui.Step("sending to %s", target.Name)
		if err := api.TestTarget(ctx, target.ID); err != nil {
			return err
		}
		ui.Success("%s delivered the test", target.Name)
		return nil
	},
}

// resolveTarget matches by exact id first, then by name, case-insensitively.
//
// An ambiguous name is an error rather than a guess: sending a test to the
// wrong channel is noise in somebody's chat room, and the ids are one command
// away.
func resolveTarget(ctx context.Context, api *client.Client, query string) (client.Target, error) {
	payload, err := api.Settings(ctx)
	if err != nil {
		return client.Target{}, err
	}
	targets := payload.Settings.Delivery.Items

	for _, target := range targets {
		if target.ID == query {
			return target, nil
		}
	}

	matches := make([]client.Target, 0, 2)
	for _, target := range targets {
		if strings.EqualFold(target.Name, query) {
			matches = append(matches, target)
		}
	}
	if len(matches) == 0 {
		for _, target := range targets {
			if strings.Contains(strings.ToLower(target.Name), strings.ToLower(query)) {
				matches = append(matches, target)
			}
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return client.Target{}, fmt.Errorf("no delivery target matches %q — run `nook targets` to list them", query)
	}

	names := make([]string, 0, len(matches))
	for _, target := range matches {
		names = append(names, target.Name)
	}
	return client.Target{}, fmt.Errorf("%q matches several targets (%s) — use an id", query, strings.Join(names, ", "))
}

func init() {
	rootCmd.AddCommand(testCmd)
}
