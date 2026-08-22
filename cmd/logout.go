package cmd

import (
	"github.com/spf13/cobra"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/config"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

// logoutCmd revokes the session on the instance and then forgets it locally.
//
// Both halves are deliberately tolerant, because the local credential leaving
// the disk is the part that matters. A server-side revoke that fails must not
// leave the token behind: the local copy is the one an attacker with this
// machine would use. Neither must a configuration file that cannot be parsed —
// logout is what somebody reaches for on a borrowed machine, and refusing
// because the JSON is malformed would leave a working credential exactly where
// they tried to remove it. Per CLI-STANDARD §6.6 it is also not an error to run
// this with no session stored.
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke the stored session",
	Long: `Revokes the session on the instance and clears it locally. The instance URL is
kept, so logging back in does not mean typing it again.

Running it with no session stored is not an error.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		cfg, err := config.Load()
		unreadable := err != nil
		if unreadable {
			ui.Warn("the stored configuration could not be read — clearing it anyway")
			cfg = config.Config{URL: config.DefaultURL}
		}
		cfg = withURLOverrides(cfg)

		if cfg.Token == "" && !unreadable {
			ui.Success("no session stored")
			return nil
		}

		var revokeErr error
		if cfg.Token != "" {
			revokeErr = client.New(cfg.URL, cfg.Token).Logout(ctx)
		}
		if err := config.Clear(); err != nil {
			return err
		}
		if revokeErr != nil {
			ui.Warn("cleared the local session, but the instance did not confirm the revoke — %s", revokeErr)
			return nil
		}

		ui.Success("logged out of %s", cfg.URL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
