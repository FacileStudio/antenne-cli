package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show what the instance is running",
	Long: `Probes the instance's health, then reports what it is watching, where it is
delivering, and how much of that is working.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, cfg, err := connect()
		if err != nil {
			return err
		}

		// Health needs no session, so probing it first tells "unreachable"
		// apart from "not logged in" — two failures with very different fixes.
		if err := api.Health(ctx); err != nil {
			return err
		}

		payload, err := api.Settings(ctx)
		if err != nil {
			return err
		}
		page, err := api.Events(ctx, client.EventQuery{Limit: 1})
		if err != nil {
			return err
		}
		pool, err := api.BusStats(ctx)
		if err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(map[string]any{
				"url":       cfg.URL,
				"runtime":   payload.Runtime,
				"providers": payload.Settings.Providers.Items,
				"targets":   payload.Settings.Delivery.Items,
				"events":    page.Stats,
				"pool":      pool,
			})
		}

		providers := payload.Settings.Providers.Items
		targets := payload.Settings.Delivery.Items

		enabledProviders := 0
		for _, provider := range providers {
			if provider.Enabled {
				enabledProviders++
			}
		}
		enabledTargets, silentTargets := 0, 0
		for _, target := range targets {
			if target.Enabled {
				enabledTargets++
			}
			if target.Silent() {
				silentTargets++
			}
		}

		ui.Success("%s is healthy", cfg.URL)
		ui.Fields([][2]string{
			{"providers", fmt.Sprintf("%d (%d enabled)", len(providers), enabledProviders)},
			{"targets", fmt.Sprintf("%d (%d enabled)", len(targets), enabledTargets)},
			{"events", strconv.Itoa(page.Stats.TotalEvents)},
			{"delivered", ui.Ok(strconv.Itoa(page.Stats.DeliverySuccessCount))},
			{"failed", failedField(page.Stats.DeliveryErrorCount)},
			{"pool", fmt.Sprintf("%d connected", pool.Connections)},
			{"listening on", strconv.Itoa(payload.Runtime.ServerPort)},
			{"settings", settingsSource(payload.Runtime.UsingPersistedSettings)},
		})

		// A target nothing routes to is the most common reason one looks broken,
		// and the dashboard is the only other place that says so.
		if silentTargets > 0 {
			ui.Warn("%d delivery target(s) have no provider or tag attached, so they receive nothing", silentTargets)
			ui.Hint("run `antenne targets` to see which")
		}
		return nil
	},
}

func failedField(count int) string {
	if count == 0 {
		return "0"
	}
	return ui.Failed(strconv.Itoa(count))
}

func settingsSource(persisted bool) string {
	if persisted {
		return "saved"
	}
	return "defaults"
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
