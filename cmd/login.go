package cmd

import (
	"bufio"
	"context"
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
	flagEmail      string
	flagPassword   string
	flagNoBrowser  bool
	flagNoListener bool
	flagListener   bool
)

const (
	ssoAppName = "Antenne"
	apiMount   = "/api"
)

var loginCmd = &cobra.Command{
	Use:   "login [url]",
	Short: "Authenticate with an Antenne instance",
	Long: `Prints an approval link and waits until you approve the sign-in in any browser,
on this machine or another one. Nothing listens on a port and nothing is typed,
so it works over SSH. The session lands in the config file.

The URL is the argument, then ANTENNE_URL, then the stored one, then
https://antenne.facile.studio.

--listener uses the older flow instead: a browser on this machine redirects to a
local port. An instance without device login falls back to it on its own.
--no-listener shows a code on the sign-in page for you to paste back. --email or
--password sign in with the instance password instead, on an instance that has one.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg = withURLOverrides(cfg)
		if len(args) == 1 {
			cfg.URL = config.NormalizeURL(args[0])
		}
		if cfg.URL == "" {
			cfg.URL = config.DefaultURL
		}

		api := client.New(cfg.URL, "")
		authConfig, err := api.AuthConfig(ctx)
		if err != nil {
			return err
		}

		passwordRequested := flagEmail != "" || flagPassword != ""
		var who string
		switch {
		case passwordRequested && authConfig.PasswordRequired:
			cfg.Token, who, err = passwordLogin(ctx, api, cfg.URL, authConfig)
		case authConfig.OIDCEnabled:
			cfg.Token, err = ssoLogin(ctx, api, cfg.URL)
			who = cfg.URL
		case authConfig.PasswordRequired:
			cfg.Token, who, err = passwordLogin(ctx, api, cfg.URL, authConfig)
		default:
			return fmt.Errorf("%s accepts no login, set ANTENNE_ADMIN_PASSWORD on the instance and restart it", cfg.URL)
		}
		if err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return err
		}
		ui.Success("logged in to %s", who)
		ui.Hint("session stored in %s", config.Path())
		return nil
	},
}

// passwordLogin signs in with the instance password and returns the token and
// the address it was issued to.
func passwordLogin(ctx context.Context, api *client.Client, serverURL string, authConfig client.AuthConfig) (string, string, error) {
	email := strings.TrimSpace(flagEmail)
	var err error
	if email == "" {
		email, err = readEmail(authConfig.Email)
		if err != nil {
			return "", "", err
		}
	}
	if email == "" {
		return "", "", fmt.Errorf("no address given, pass --email or type one at the prompt")
	}

	password := flagPassword
	if password == "" {
		password, err = readPassword(email)
		if err != nil {
			return "", "", err
		}
	}
	if strings.TrimSpace(password) == "" {
		return "", "", fmt.Errorf("no password given, pass --password or type one at the prompt")
	}

	token, err := api.Login(ctx, email, password)
	if err != nil {
		var apiErr *client.Error
		if errors.As(err, &apiErr) && apiErr.Unauthenticated() {
			return "", "", fmt.Errorf("wrong address or password for %s", serverURL)
		}
		return "", "", err
	}
	return token, fmt.Sprintf("%s as %s", serverURL, email), nil
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
	loginCmd.Flags().BoolVar(&flagNoBrowser, "no-browser", false, "Do not open a browser, print the sign-in URL")
	loginCmd.Flags().BoolVar(&flagNoListener, "no-listener", false, "Sign in from a browser on another machine, pasting the code back")
	loginCmd.Flags().BoolVar(&flagListener, "listener", false, "Use the loopback browser flow instead of the approval link")
	rootCmd.AddCommand(loginCmd)
}
