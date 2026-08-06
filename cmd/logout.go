package cmd

import (
	"github.com/spf13/cobra"

	"github.com/FacileStudio/nook-cli/internal/config"
	"github.com/FacileStudio/nook-cli/internal/ui"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke the stored session",
	Long: `Revokes the session on the instance and clears it locally. The instance URL is
kept, so logging back in does not mean typing it again.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, cfg, err := connect()
		if err != nil {
			return err
		}
		if cfg.Token == "" {
			ui.Success("no session stored")
			return nil
		}

		// A server-side revoke that fails must not leave the token on disk: the
		// local copy is the one an attacker with this machine would use.
		revokeErr := api.Logout(ctx)
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
