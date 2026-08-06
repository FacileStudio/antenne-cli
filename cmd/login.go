package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/FacileStudio/nook-cli/internal/client"
	"github.com/FacileStudio/nook-cli/internal/config"
	"github.com/FacileStudio/nook-cli/internal/ui"
)

var flagPassword string

var loginCmd = &cobra.Command{
	Use:   "login [url]",
	Short: "Authenticate with a Nook instance",
	Long: `Stores the instance URL and the session it returns, so later commands need
neither. The URL defaults to the one already stored, and the password is read
from the terminal without echoing unless --password is given.

An instance running without an admin password needs no login at all; this
command says so and stores only the URL.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			cfg.URL = config.NormalizeURL(args[0])
		}
		if flagURL != "" {
			cfg.URL = config.NormalizeURL(flagURL)
		}
		if cfg.URL == "" {
			cfg.URL = config.DefaultURL
		}

		api := client.New(cfg.URL, "")

		// Ask the instance whether it wants a password before prompting for
		// one. A local install with no admin password would otherwise be asked
		// for a credential that does not exist.
		session, err := api.PublicSession(ctx)
		if err != nil {
			return err
		}
		if !session.PasswordRequired {
			cfg.Token = ""
			if err := config.Save(cfg); err != nil {
				return err
			}
			ui.Success("connected to %s", cfg.URL)
			ui.Hint("this instance has no admin password, so every caller is served as the admin")
			return nil
		}

		password := flagPassword
		if password == "" {
			password, err = readPassword(session.Username)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(password) == "" {
			return fmt.Errorf("no password given — pass --password or type one at the prompt")
		}

		token, err := api.Login(ctx, password)
		if err != nil {
			var apiErr *client.Error
			if errors.As(err, &apiErr) && apiErr.Unauthenticated() {
				return fmt.Errorf("wrong password for %s", session.Username)
			}
			return err
		}

		cfg.Token = token
		if err := config.Save(cfg); err != nil {
			return err
		}

		ui.Success("logged in to %s as %s", cfg.URL, session.Username)
		ui.Hint("session stored in %s", config.Path())
		return nil
	},
}

// readPassword prompts on stderr, keeping stdout clean for data, and never
// echoes. A pipe has no terminal to turn echo off on, so it is read as a line.
func readPassword(username string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		return strings.TrimRight(line, "\r\n"), err
	}

	fmt.Fprintf(os.Stderr, "Password for %s: ", username)
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(raw), err
}

func init() {
	loginCmd.Flags().StringVar(&flagPassword, "password", "", "Password to use instead of prompting")
	rootCmd.AddCommand(loginCmd)
}
