package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/FacileStudio/porte/loopback"
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
)

const (
	ssoAppName = "Antenne"
	apiMount   = "/api"
)

var loginCmd = &cobra.Command{
	Use:   "login [url]",
	Short: "Authenticate with an Antenne instance",
	Long: `Signs in through your browser. A session already open with another Facile tool
completes the login without a prompt, and the session lands in the config file.

The URL is the argument, then ANTENNE_URL, then the stored one, then
https://antenne.facile.studio.

A machine with no browser, or --no-listener, shows a code on the sign-in page
for you to paste back. --email or --password sign in with the instance password
instead, on an instance that has one.`,
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
		case authConfig.OIDCEnabled && !passwordRequested && (authConfig.SSOOnly || flagNoListener || browserAvailable()):
			flagNoListener = flagNoListener || !browserAvailable()
			cfg.Token, err = ssoLogin(ctx, api, cfg.URL)
			who = cfg.URL
		case authConfig.PasswordRequired:
			cfg.Token, who, err = passwordLogin(ctx, api, cfg.URL, authConfig)
		case authConfig.OIDCEnabled:
			flagNoListener = true
			cfg.Token, err = ssoLogin(ctx, api, cfg.URL)
			who = cfg.URL
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

// browserAvailable reports whether opening a URL can plausibly reach a person:
// an interactive terminal, no --no-browser, an opener on PATH and a display.
func browserAvailable() bool {
	if flagNoBrowser || !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	opener := "xdg-open"
	switch runtime.GOOS {
	case "darwin":
		opener = "open"
	case "windows":
		opener = "cmd"
	}
	if _, err := exec.LookPath(opener); err != nil {
		return false
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return os.Getenv("SSH_CONNECTION") == ""
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// ssoLogin signs in through the browser and returns the session token. The
// server owns the exchange: the browser lands on a loopback port with a
// one-time code, or shows the code for pasting when there is no listener.
func ssoLogin(ctx context.Context, api *client.Client, serverURL string) (string, error) {
	if flagNoListener {
		return ssoLoginByPaste(ctx, api, serverURL)
	}

	listener, err := loopback.Listen()
	if err != nil {
		return "", fmt.Errorf("cannot open a loopback port to receive the login: %w", err)
	}
	defer listener.Close()
	listener.AppName = ssoAppName

	state, err := loopback.RandomState()
	if err != nil {
		return "", err
	}

	authURL := listener.LoginURL(serverURL, apiMount, state)
	if !loopback.OpenBrowser(authURL) {
		ui.Step("open %s", authURL)
	}
	ui.Step("waiting for the browser to sign you in to %s", serverURL)

	code, err := listener.WaitForCode(ctx, state)
	if err != nil {
		if errors.Is(err, loopback.ErrTimeout) {
			return "", fmt.Errorf("timed out waiting for the browser, run 'antenne login' again")
		}
		return "", err
	}
	return api.Exchange(ctx, code)
}

// ssoLoginByPaste leaves the loopback port off the login URL, which makes the
// server show the code on the page instead of redirecting it to 127.0.0.1.
func ssoLoginByPaste(ctx context.Context, api *client.Client, serverURL string) (string, error) {
	authURL := strings.TrimRight(serverURL, "/") + apiMount + "/auth/oidc?flow=cli"
	if !flagNoBrowser && browserAvailable() {
		loopback.OpenBrowser(authURL)
	}
	ui.Step("open %s", authURL)

	fmt.Fprint(os.Stderr, "Paste the code from that page: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("failed to read the login code: %w", err)
	}
	code := strings.TrimSpace(line)
	if code == "" {
		return "", fmt.Errorf("no login code was pasted, the code is good for sixty seconds so run 'antenne login --no-listener' again")
	}
	return api.Exchange(ctx, code)
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
	rootCmd.AddCommand(loginCmd)
}
