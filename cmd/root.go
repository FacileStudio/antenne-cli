// Package cmd implements the antenne command tree.
package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/config"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

var version = "dev"

var (
	flagURL     string
	flagJSON    bool
	flagNoColor bool
)

var rootCmd = &cobra.Command{
	Use:   "antenne",
	Short: "Terminal client for an Antenne instance",
	Long: `Antenne watches providers, routes what they emit to delivery targets, and hosts the event
bus that other Facile apps sync through. This is its terminal client: it
reads the activity log, follows it live, and exercises delivery targets without
opening the dashboard.

Configuration stays in the instance. This client never writes it.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	// Set once the command's own body starts. Cobra validates flags and args
	// before this runs, so an error arriving with it still false is a usage
	// error rather than a failure of the work — and those exit 2, not 1.
	PersistentPreRun: func(cmd *cobra.Command, args []string) { commandStarted = true },
}

var commandStarted bool

func init() {
	rootCmd.Version = version
	// cobra's default is `<bin> version <v>`, which the installer cannot parse.
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "Antenne instance URL, overriding the stored one")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Print one JSON document and nothing else")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")

	cobra.OnInitialize(func() {
		// Structured output forces colour off: a caller piping JSON into jq
		// must not receive escape codes.
		if flagNoColor || flagJSON {
			ui.DisableColor()
		}
	})
}

// ErrInterrupted marks a command stopped by a signal rather than a failure.
var ErrInterrupted = errors.New("interrupted")

// Execute runs the command tree and maps the outcome onto an exit code:
// 0 success, 1 error, 2 usage error (cobra's own), 130 on SIGINT.
func Execute() {
	err := rootCmd.Execute()
	switch {
	case err == nil:
		return
	case !commandStarted:
		ui.Error("%s", err)
		ui.Hint("run `antenne <command> --help` for usage")
		os.Exit(2)
	case errors.Is(err, ErrInterrupted):
		// 128 + SIGINT, which is what a shell and every `while` loop expect
		// from a process the user stopped.
		os.Exit(130)
	default:
		report(err)
		os.Exit(1)
	}
}

// report prints a failure through the standard vocabulary, adding the one hint
// that actually resolves it where the cause is knowable.
//
// The login command handles its own 401 — telling somebody who is running
// `antenne login` to run `antenne login` explains nothing.
func report(err error) {
	var apiErr *client.Error
	if errors.As(err, &apiErr) && apiErr.Unauthenticated() {
		ui.Error("not authenticated — run `antenne login`")
		return
	}
	ui.Error("%s", err)
}

// signalContext cancels on SIGINT and SIGTERM, so a long-running follow stops
// cleanly rather than being killed mid-write.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

// connect builds a client from the stored configuration, with --url and ANTENNE_URL
// overriding it in that order of specificity.
func connect() (*client.Client, config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cfg, err
	}

	if fromEnv := os.Getenv("ANTENNE_URL"); fromEnv != "" {
		cfg.URL = config.NormalizeURL(fromEnv)
	}
	if flagURL != "" {
		cfg.URL = config.NormalizeURL(flagURL)
	}

	return client.New(cfg.URL, cfg.Token), cfg, nil
}
