package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/config"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

var (
	flagEmail    string
	flagPassword string
)

var loginCmd = &cobra.Command{
	Use:   "login [url]",
	Short: "Authenticate with an Antenne instance",
	Long: `Stores the instance URL and the session token it returns, so later commands need
neither. The URL defaults to the one already stored, the address is prefilled
with the one the instance reports, and the password is read from the terminal
without echoing unless --password is given.

An instance that only federates through single sign-on cannot be signed into
from here yet; this command says so rather than asking for a password it would
refuse.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		// Loaded, not built: the stored document is read and only the two
		// fields this command owns are replaced, so a key added later survives
		// a login instead of being reset to its zero value.
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

		// Ask the instance what it accepts before prompting for anything. An
		// instance with no password login has no endpoint to send one to, and
		// the address to prefill is the instance's to decide.
		authConfig, err := api.AuthConfig(ctx)
		if err != nil {
			return err
		}
		if !authConfig.PasswordRequired {
			if authConfig.SSOOnly || authConfig.OIDCEnabled {
				return fmt.Errorf("%s signs in through single sign-on, and this client cannot run the browser flow yet — sign in from the dashboard", cfg.URL)
			}
			return fmt.Errorf("%s accepts no password login — set ANTENNE_ADMIN_PASSWORD on the instance and restart it", cfg.URL)
		}

		email := strings.TrimSpace(flagEmail)
		if email == "" {
			email, err = readEmail(authConfig.Email)
			if err != nil {
				return err
			}
		}
		if email == "" {
			return fmt.Errorf("no address given — pass --email or type one at the prompt")
		}

		password := flagPassword
		if password == "" {
			password, err = readPassword(email)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(password) == "" {
			return fmt.Errorf("no password given — pass --password or type one at the prompt")
		}

		token, err := api.Login(ctx, email, password)
		if err != nil {
			var apiErr *client.Error
			if errors.As(err, &apiErr) && apiErr.Unauthenticated() {
				// The instance answers the same way to an unknown address and
				// a wrong password, on purpose. Guessing which it was here
				// would undo that.
				return fmt.Errorf("wrong address or password for %s", cfg.URL)
			}
			return err
		}

		cfg.Token = token
		if err := config.Save(cfg); err != nil {
			return err
		}

		ui.Success("logged in to %s as %s", cfg.URL, email)
		ui.Hint("session stored in %s", config.Path())
		return nil
	},
}

// readEmail prompts on stderr with the instance's own address prefilled, so the
// common case is a bare Enter and an operator whose account moved off the
// bootstrap address can still type their own.
//
// Without a terminal there is nothing to prompt on and stdin belongs to the
// password, so the instance's address stands.
func readEmail(prefill string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return prefill, nil
	}

	fmt.Fprintf(os.Stderr, "Email [%s]: ", prefill)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if typed := strings.TrimSpace(line); typed != "" {
		return typed, nil
	}
	return prefill, nil
}

// readPassword prompts on stderr, keeping stdout clean for data, and never
// echoes. A pipe has no terminal to turn echo off on, so it is read as a line.
func readPassword(email string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		return strings.TrimRight(line, "\r\n"), err
	}

	fmt.Fprintf(os.Stderr, "Password for %s: ", email)
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(raw), err
}

func init() {
	loginCmd.Flags().StringVar(&flagEmail, "email", "", "Address to sign in with instead of prompting")
	loginCmd.Flags().StringVar(&flagPassword, "password", "", "Password to use instead of prompting")
	rootCmd.AddCommand(loginCmd)
}
